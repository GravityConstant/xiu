package models

import (
	"fmt"

	"xiu/pbx/util"
)

type BindPhone struct {
	Id                int64
	DialplanId        int64
	AreaCode          string
	BindPhone         string
	Privilege         int
	Jobnum            string
	GatewayAmount     int
	WaitTime          int
	AreaDistrictNo    string
	IsTimeplan        bool
	IsRest            bool
	OpenSatisfySurvey bool
}

func GetBindPhoneByDialplan(bpIds string) []*BindPhone {
	sql := `SELECT bp.id, bp.phone_id dialplan_id, bp.bind_phone, bp.privilege, bp.jobnum, bp.gateway_amount, bp.wait_time, bp.area_district_no, bp.is_timeplan, bp.is_rest, bp.open_satisfy_survey, d.area_code from %s d, %s bp where d.id=bp.phone_id and bp.id in (%s) ORDER BY privilege`
	sql = fmt.Sprintf(sql, DialplanTBName(), BindPhonerTBName(), bpIds)

	dds := make([]*BindPhone, 0)
	if err := ImplInstance.DB.Raw(sql).Scan(&dds).Error; err != nil {
		util.Error("models/bind_phone.go", "query bind phone error", err)
	}

	return dds
}

type BindPhoneTimeset struct {
	Id int64
}

func GetBindPhoneTimesetById(id int64, isRest bool) []BindPhoneTimeset {
	// SELECT CURRENT_TIME::time without time zone
	sql := `SELECT id from %s where bindphone_id=%d and weekday=extract(DOW FROM now())-1 and %s (CURRENT_TIME BETWEEN morning_start and morning_stop or CURRENT_TIME BETWEEN afternoon_start and afternoon_stop)`
	var not string
	if isRest {
		not = "not"
	}
	sql = fmt.Sprintf(sql, BindPhoneTimesetTBName(), id, not)

	var valids []BindPhoneTimeset
	if err := ImplInstance.DB.Raw(sql).Scan(&valids).Error; err != nil {
		util.Error("models/bind_phone.go", "query bind phone timeset error", err)
	}

	return valids
}

func IsExistAreaSetting(name, value string) int {
	// SELECT CURRENT_TIME::time without time zone
	sql := `SELECT count(*) from %s where %s='%s'`
	sql = fmt.Sprintf(sql, BaseMobileLocationTBName(), name, value)

	var count int
	if err := ImplInstance.DB.Raw(sql).Count(&count).Error; err != nil {
		util.Error("models/bind_phone.go", "query area count error", err)
	}

	return count
}

func GetMobileDistrictNo(no string) []string {
	// SELECT CURRENT_TIME::time without time zone
	var districtNos []string
	if err := ImplInstance.DB.Table(BaseMobileLocationTBName()).Where("no=?", no).Pluck("district_no", &districtNos).Error; err != nil {
		util.Error("models/bind_phone.go", "query bind phone district_no error", err)
	}

	return districtNos
}

// 是否是黑名单
func IsCallBlacklist(dialplanNumber, caller string) int {
	sql := `SELECT count(*) from %s where dialplan_number=? and call_number=?`
	sql = fmt.Sprintf(sql, BlacklistTBName())

	var count int
	if err := ImplInstance.DB.Raw(sql, dialplanNumber, caller).Count(&count).Error; err != nil {
		util.Error("models/bind_phone.go", "query blacklist error", err)
	}

	return count
}

// 获取工号
func GetJobnumString(dialplanNumber, callee string) string {
	sql := `SELECT bp.jobnum from %s d, %s bp where d.id=bp.phone_id and d.dialplan_number=? and bp.bind_phone=?`
	sql = fmt.Sprintf(sql, DialplanTBName(), BindPhonerTBName())

	var jobnum []string
	if err := ImplInstance.DB.Raw(sql, dialplanNumber, callee).Pluck("jobnum", &jobnum).Error; err != nil {
		util.Error("models/bind_phone.go", "query job num error", err)
		return ""
	} else {
		return jobnum[0]
	}
}

// 获取是否开启满意度状态
func GetOpenSatisfySurvey(dialplanNumber, callee string) bool {
	sql := `SELECT bp.open_satisfy_survey from %s d, %s bp where d.id=bp.phone_id and d.dialplan_number=? and bp.bind_phone=?`
	sql = fmt.Sprintf(sql, DialplanTBName(), BindPhonerTBName())

	var value []bool
	if err := ImplInstance.DB.Raw(sql, dialplanNumber, callee).Pluck("open_satisfy_survey", &value).Error; err != nil {
		util.Error("models/bind_phone.go", "query satisfy survey error", err)
		return false
	} else {
		return value[0]
	}
}
