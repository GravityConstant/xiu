package models

import (
	"fmt"
	"strings"

	"xiu/pbx/util"
)

type Extension struct {
	DialplanId          int64
	DialplanName        string
	DialplanContext     string
	DialplanNumber      string
	DialplanEnabled     bool
	DialplanDetailId    string
	DialplanDetailOrder int64
	DialplanDetailApp   string
	DialplanDetailData  string
}

type ExtensionQueryParam struct {
	BaseQueryParam
	DialplanNumber string
}

func GetAllDialplanDetail(params *ExtensionQueryParam) []*Extension {

	sql := `SELECT d.id dialplan_id, dialplan_name, dialplan_context, dialplan_number, dialplan_enabled, dd.id dialplan_detail_id, orderid dialplan_detail_order, (SELECT name from %s where id=dd.dialplan_detail_type_id) dialplan_detail_app, dialplan_detail_data FROM %s d, %s dd WHERE dd.dialplan_id=d.id %s order by d.id`
	// condition
	var where string
	mw := make(map[string]string)
	if len(params.DialplanNumber) > 0 {
		mw["dialplan_number"] = `d.dialplan_number='%s'`
		mw["dialplan_number"] = fmt.Sprintf(mw["dialplan_number"], params.DialplanNumber)

	}
	for _, val := range mw {
		where += val + " and"
	}
	if len(where) > 0 {
		where = "and " + strings.TrimSuffix(where, "and")
	}
	// sql combine
	sql = fmt.Sprintf(sql, CallOperationTBName(), DialplanTBName(), DialplanDetailTBName(), where)

	dds := make([]*Extension, 0)
	ImplInstance.DB.Raw(sql).Scan(&dds)
	if ImplInstance.DB.Error != nil {
		util.Error("db GetAllDialplanDetail", " error occur: ", ImplInstance.DB.Error)
	}

	return dds
}

type Menu struct {
	Id        int64
	ParentId  int64  `gorm:"column:ivr_menu_parent_id"`
	Extension string `gorm:"column:ivr_menu_extension"`
	File      string //
	DigitLen  int    `gorm:"column:ivr_menu_digit_len"`
	App       string //
	Digits    string `gorm:"column:ivr_menu_option_digits"`
	Param     string `gorm:"column:ivr_menu_option_param"`
}

type MenuQueryParam struct {
	BaseQueryParam
	Extension string
}

/*
WITH RECURSIVE t(id, ivr_menu_name, ivr_menu_extension, ivr_menu_greet_long_id, ivr_menu_digit_len, ivr_menu_parent_id) as (
	SELECT id, ivr_menu_name, ivr_menu_extension, ivr_menu_greet_long_id, ivr_menu_digit_len, ivr_menu_parent_id
	FROM call_ivr_menus
 	where ivr_menu_parent_id=0

UNION ALL

	SELECT d.id, d.ivr_menu_name, d.ivr_menu_extension, d.ivr_menu_greet_long_id, d.ivr_menu_digit_len, d.ivr_menu_parent_id
	from call_ivr_menus d
	JOIN t on d.ivr_menu_parent_id = t.id
)
SELECT t.id, t.ivr_menu_name, t.ivr_menu_extension, t.ivr_menu_greet_long_id, t.ivr_menu_digit_len, t.ivr_menu_parent_id,
o.ivr_menu_option_digits, o.ivr_menu_option_param
from t
LEFT JOIN call_ivr_menu_options o on t.id=o.ivr_menu_id
ORDER BY t.ivr_menu_extension
*/
func GetAllIvrMenuDetail(params *MenuQueryParam) []*Menu {

	sql := `WITH RECURSIVE t(id, ivr_menu_name, ivr_menu_extension, ivr_menu_greet_long_id, ivr_menu_digit_len, ivr_menu_parent_id) as (SELECT id, ivr_menu_name, ivr_menu_extension, ivr_menu_greet_long_id, ivr_menu_digit_len, ivr_menu_parent_id FROM %s where ivr_menu_parent_id=0 %s UNION ALL SELECT d.id, d.ivr_menu_name, d.ivr_menu_extension, d.ivr_menu_greet_long_id, d.ivr_menu_digit_len, d.ivr_menu_parent_id from %s d JOIN t on d.ivr_menu_parent_id = t.id) SELECT t.id, t.ivr_menu_name, t.ivr_menu_extension, (SELECT ring_path from %s where id=t.ivr_menu_greet_long_id) file, t.ivr_menu_digit_len, t.ivr_menu_parent_id,  o.ivr_menu_option_digits, (SELECT name from call_operation where id=o.ivr_menu_option_action_id) app, o.ivr_menu_option_param from t LEFT JOIN %s o on t.id=o.ivr_menu_id ORDER BY t.ivr_menu_extension`
	// condition
	var where string
	mw := make(map[string]string)
	if len(params.Extension) > 0 {
		mw["extension"] = `ivr_menu_extension='%s'`
		mw["extension"] = fmt.Sprintf(mw["extension"], params.Extension)

	}
	for _, val := range mw {
		where += "and " + val + " and"
	}
	where = strings.TrimSuffix(where, "and")
	// sql combine
	sql = fmt.Sprintf(sql, IvrMenuTBName(), where, IvrMenuTBName(), RingsTBName(), IvrMenuOptionTBName())

	menu := make([]*Menu, 0)
	ImplInstance.DB.Raw(sql).Scan(&menu)
	if ImplInstance.DB.Error != nil {
		util.Error("db GetAllIvrMenuDetail", " error occur: ", ImplInstance.DB.Error)
	}

	return menu
}
