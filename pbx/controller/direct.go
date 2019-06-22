package controller

import (
	"fmt"
	"strings"
	"sync"

	"xiu/pbx/models"
	"xiu/pbx/util"
)

type Outcall struct {
	CallString string
}

const (
	// {ignore_early_media=true,originate_continue_on_timeout=true}[leg_timeout=15]sofia/gateway/  zqzj/13675017141|[leg_timeout=25]sofia/gateway/zqzj/83127866"
	globalParam  = `{ignore_early_media=ring_ready,originate_continue_on_timeout=true,sip_h_Diversion=<sip:%s@ip>}`
	privateParam = `[leg_timeout=%d]sofia/gateway/%s/%s`
)

func init() {
	// genome editing
}

type OpenTimeplanBindPhone struct {
	Id     int64
	IsRest bool
}

func (self *Outcall) GetCallString(diversion, caller, bpIds string) {
	// `{originate_timeout=%d,sip_h_Diversion=<sip:%s@ip>}sofia/gateway/zqzj/13675017141`

	// 是否是黑名单
	if count := models.IsCallBlacklist(diversion, caller); count > 0 {
		return
	}

	// 获取bind_phoner表里的记录
	bps := models.GetBindPhoneByDialplan(bpIds)

	// 区域设置使用channel
	areaBindPhones := make([]models.BindPhone, 0)

	areaWorkers := make([]<-chan models.BindPhone, len(bps))
	for i, item := range bps {
		areaWorkers[i] = filterBindPhoneArea(item)
	}
	for t := range areaMerge(areaWorkers...) {
		areaBindPhones = append(areaBindPhones, t)
	}

	// 时间设置使用channel
	timeplanBindPhones := make([]models.BindPhone, 0)

	timeplanWorkers := make([]<-chan models.BindPhone, len(bps))
	for i, item := range bps {
		timeplanWorkers[i] = filterBindPhoneTimeplan(item)
	}
	for t := range timeplanMerge(timeplanWorkers...) {
		timeplanBindPhones = append(timeplanBindPhones, t)
	}

	util.Info("direct.go", "61", areaBindPhones)
	util.Info("direct.go", "62", timeplanBindPhones)

	// 区域设置和时间设置都符合的才是最后要出局的号码
	resultBindPhones := make([]models.BindPhone, 0)
	for _, abp := range areaBindPhones {
		for _, tbp := range timeplanBindPhones {
			if abp.Id == tbp.Id {
				resultBindPhones = append(resultBindPhones, abp)
				break
			}
		}
	}

	// 拼接呼叫字符串
	pPs := make([]string, 0)
	var pP string

	if len(resultBindPhones) > 0 {
		for _, bp := range resultBindPhones {
			pP = fmt.Sprintf(privateParam, bp.WaitTime, GetGatewayByAreaCode(bp.AreaCode), bp.BindPhone)
			pPs = append(pPs, pP)
		}

		pP = strings.Join(pPs, "|")
		gP := fmt.Sprintf(globalParam, diversion)
		self.CallString = gP + pP
	}

}

func filterBindPhoneArea(item *models.BindPhone) <-chan models.BindPhone {
	packages := make(chan models.BindPhone)
	go func() {
		if len(item.AreaDistrictNo) == 0 {
			packages <- *item
		} else {
			name, value := PhoneAreaCode(strings.TrimSpace(item.BindPhone))
			if count := models.IsExistAreaSetting(name, value); count > 0 {
				packages <- *item
			}
		}
		close(packages)
	}()
	return packages
}

func areaMerge(channels ...<-chan models.BindPhone) <-chan models.BindPhone {
	var wg sync.WaitGroup

	wg.Add(len(channels))
	outgoingPackages := make(chan models.BindPhone)
	multiplex := func(c <-chan models.BindPhone) {
		defer wg.Done()
		for i := range c {
			outgoingPackages <- i
		}
	}
	for _, c := range channels {
		go multiplex(c)
	}
	go func() {
		wg.Wait()
		close(outgoingPackages)
	}()
	return outgoingPackages
}

func filterBindPhoneTimeplan(item *models.BindPhone) <-chan models.BindPhone {
	packages := make(chan models.BindPhone)
	go func() {
		if item.IsTimeplan {
			valids := models.GetBindPhoneTimesetById(item.Id, item.IsRest)
			for _, v := range valids {
				if v.Id > 0 {
					packages <- *item
					break
				}
			}
		} else {
			packages <- *item
		}
		close(packages)
	}()
	return packages
}

func timeplanMerge(channels ...<-chan models.BindPhone) <-chan models.BindPhone {
	var wg sync.WaitGroup

	wg.Add(len(channels))
	outgoingPackages := make(chan models.BindPhone)
	multiplex := func(c <-chan models.BindPhone) {
		defer wg.Done()
		for i := range c {
			outgoingPackages <- i
		}
	}
	for _, c := range channels {
		go multiplex(c)
	}
	go func() {
		wg.Wait()
		close(outgoingPackages)
	}()
	return outgoingPackages
}

func PhoneAreaCode(number string) (name, value string) {
	if strings.Index(number, "1") == 0 && len(number) == 11 {
		// 手机截取前7位
		name = "no"
		value = number[:7]
	} else if strings.Index(number, "010") == 0 || strings.Index(number, "02") == 0 {
		// 电话截取
		name = "district_no"
		value = number[:3]
	} else if strings.Index(number, "0") == 0 {
		if strings.Index(number, "1") == 1 {
			// 认为是国字码的手机
			// 手机截取前7位
			name = "no"
			value = number[1:8]
			// 判断是否有重复的值
		} else {
			// 电话截取前4位
			name = "district_no"
			value = number[:4]
		}
	}
	return
}

func GetGatewayByAreaCode(code string) (gw string) {
	switch code {
	case "0591":
		gw = "zqzj"
	case "0592":
		gw = "xiamen"
	case "0593":
		gw = "ningde"
	case "0594":
		gw = "putian"
	case "0595":
		gw = "quanzhou"
	case "0596":
		gw = "zhangzhou"
	case "0597":
		gw = "longyan"
	case "0598":
		gw = "sanming"
	case "0599":
		gw = "nanping"
	default:
		gw = "zqzj"
	}
	return
}
