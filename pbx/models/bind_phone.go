package models

import (
	"fmt"
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
	ImplInstance.DB.Raw(sql).Scan(&dds)

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
	ImplInstance.DB.Raw(sql).Scan(&valids)

	return valids
}

func IsExistAreaSetting(name, value string) int {
	// SELECT CURRENT_TIME::time without time zone
	sql := `SELECT count(*) from %s where %s='%s'`
	sql = fmt.Sprintf(sql, BaseMobileLocationTBName(), name, value)

	var count int
	ImplInstance.DB.Raw(sql).Count(&count)

	return count
}

// 是否是黑名单
func IsCallBlacklist(dialplanNumber, caller string) int {
	sql := `SELECT count(*) from %s where dialplan_number=? and call_number=?`
	sql = fmt.Sprintf(sql, BlacklistTBName())

	var count int
	ImplInstance.DB.Raw(sql, dialplanNumber, caller).Count(&count)

	return count
}
