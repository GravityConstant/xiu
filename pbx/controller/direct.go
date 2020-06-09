package controller

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"xiu/pbx/models"
	"xiu/pbx/util"
)

type Outcall struct {
	CallString string
}

// 绑定号码乱序排序，随机接听使用
type data []models.BindPhone

const (
	// {ignore_early_media=true,originate_continue_on_timeout=true}[leg_timeout=15]sofia/gateway/zqzj/13675017141|[leg_timeout=25]sofia/gateway/zqzj/83127866"
	// ignore_early_media=ring_ready,
	globalParam  = `{%soriginate_continue_on_timeout=true,sip_h_Diversion=<sip:%s@%s>}`
	privateParam = `[leg_timeout=%d]sofia/gateway/%s/%s`
)

func init() {
	// genome editing
}

func (self *Outcall) GetCallString(diversion, caller, bpIds string, responseType int) {
	// `{originate_timeout=%d,sip_h_Diversion=<sip:%s@ip>}sofia/gateway/zqzj/13675017141`

	// bpIds为空，直接返回
	if len(bpIds) == 0 {
		return
	}
	// 是否是黑名单
	if count := models.IsCallBlacklist(diversion, caller); count > 0 {
		util.Info("direct.go", "callblack worked", caller)
		return
	}
	util.Info("direct.go", "caller number", caller)
	// 获取bind_phoner表里的记录
	bps := models.GetBindPhoneByDialplan(bpIds)

	// 区域设置使用channel
	areaBindPhones := make([]models.BindPhone, 0)

	areaWorkers := make([]<-chan models.BindPhone, len(bps))
	for i, item := range bps {
		areaWorkers[i] = filterBindPhoneArea(item, caller)
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

	util.Info("direct.go", "area filtered", areaBindPhones)
	util.Info("direct.go", "time filtered", timeplanBindPhones)

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
	var pP, gP string

	if len(resultBindPhones) > 0 {
		gw, ip := GetGatewayByAreaCode(resultBindPhones[0].AreaCode)
		// 随机接听打乱顺序
		data(resultBindPhones).random()
		for _, bp := range resultBindPhones {

			pP = fmt.Sprintf(privateParam, bp.WaitTime, gw, bp.BindPhone)
			pPs = append(pPs, pP)
		}

		pP = strings.Join(pPs, "|")
		if util.PbxConfigInstance.Get("freeswitch::ignore_early_media") == "false" {
			// 网关都是设置caller-id-in-from=true,只要设置origination_caller_id_number就可以了
			gP = fmt.Sprintf(globalParam, "origination_caller_id_number=95795279,", diversion, ip)
		} else {
			gP = fmt.Sprintf(globalParam, "ignore_early_media=ring_ready,", diversion, ip)
		}

		self.CallString = gP + pP
	}

}

func filterBindPhoneArea(item *models.BindPhone, caller string) <-chan models.BindPhone {
	packages := make(chan models.BindPhone)
	go func() {
		if len(item.AreaDistrictNo) == 0 {
			packages <- *item
		} else {
			// 得到主叫可以判断区域的前缀
			name, value := PhoneAreaCode(strings.TrimSpace(caller))
			util.Info("direct.go", "caller area code", name, value)
			setAreas := strings.Split(item.AreaDistrictNo, ",")
			var districtNos []string

			var aTmp, dnTmp string
			if name == "no" {
				districtNos = models.GetMobileDistrictNo(value)
			END:
				for _, a := range setAreas {
					aTmp = strings.TrimSpace(a)
					for _, dn := range districtNos {
						dnTmp = strings.TrimSpace(dn)
						if aTmp == dnTmp {
							packages <- *item
							break END
						}
					}
				}
			} else if name == "district_no" {
				for _, a := range setAreas {
					aTmp = strings.TrimSpace(a)
					if aTmp == value {
						packages <- *item
						break
					}
				}
			} else {
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

func GetGatewayByAreaCode(code string) (gw, ip string) {
	switch code {
	case "0591": // 现无此网关
		gw = "zqzj"
		ip = "192.168.1.213"
	case "0592":
		gw = "xiamen"
		ip = "192.168.1.214"
	case "0593":
		gw = "ningde"
		ip = "192.168.1.218"
	case "0594":
		gw = "putian"
		ip = "192.168.1.216"
	case "0595":
		gw = "quanzhou"
		ip = "192.168.1.215"
	case "0596": // 现无此网关
		gw = "zhangzhou"
		ip = "192.168.1.220"
	case "0597":
		gw = "longyan"
		ip = "192.168.1.219"
	case "0598":
		gw = "sanming"
		ip = "192.168.1.217"
	case "0599":
		gw = "nanping"
		ip = "192.168.1.201"
	default: // 现无此网关
		gw = "zqzj"
		ip = "192.168.1.213"
	}
	return
}

func (d data) random() {
	rand.Seed(time.Now().Unix())
	strsLen := len(d)

	ns := make([]int, strsLen)
	for i := 0; i < strsLen; i++ {
		ns[i] = -1
	}
	// fmt.Println("init:", ns, strsLen)

	res := make(data, strsLen)

	for i := 0; i < strsLen; i++ {
		n := rand.Intn(strsLen)
		n = getPosition(n, strsLen, ns, true)
		// fmt.Println("real n: ", n)
		ns[i] = n
		res[i] = d[n]
	}
	copy(d, res)
}

// 还是要用结构体写，不然麻烦
func random(strs []string) []string {
	rand.Seed(time.Now().Unix())
	strsLen := len(strs)

	ns := make([]int, strsLen)
	for i := 0; i < strsLen; i++ {
		ns[i] = -1
	}
	// fmt.Println("init:", ns, strsLen)

	res := make([]string, strsLen)

	for i := 0; i < strsLen; i++ {
		n := rand.Intn(strsLen)
		n = getPosition(n, strsLen, ns, true)
		// fmt.Println("real n: ", n)
		ns[i] = n
		res[i] = strs[n]
	}
	return res
}

func getPosition(e, l int, ns []int, up bool) (v int) {
	fmt.Println("in position:", e, ns)
	for _, n := range ns {
		if n == e {
			if e == 0 {
				e = getPosition(e+1, l, ns, true)
			} else if e == l-1 {
				e = getPosition(e-1, l, ns, false)
			} else {
				if up {
					e = getPosition(e+1, l, ns, true)
				} else {
					e = getPosition(e-1, l, ns, false)
				}
			}
		}
	}
	return e
}
