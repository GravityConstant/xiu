package models

import (
	"fmt"
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

func GetAllDialplanDetail() []*Extension {

	sql := `SELECT d.id dialplan_id, dialplan_name, dialplan_context, dialplan_number, dialplan_enabled, dd.id dialplan_detail_id, orderid dialplan_detail_order, (SELECT name from %s where id=dd.dialplan_detail_type_id) dialplan_detail_app, dialplan_detail_data FROM %s d, %s dd WHERE dd.dialplan_id=d.id order by d.id`
	sql = fmt.Sprintf(sql, CallOperationTBName(), DialplanTBName(), DialplanDetailTBName())

	dds := make([]*Extension, 0)
	ImplInstance.DB.Raw(sql).Scan(&dds)

	return dds
}

type Menu struct {
	Extension   string `gorm:"column:ivr_menu_extension"`
	GreetLongId int64  `gorm:"column:ivr_menu_greet_long_id"`
	DigitLen    int    `gorm:"column:ivr_menu_digit_len"`
	App         string
	Digits      string `gorm:"column:ivr_menu_option_digits"`
	Param       string `gorm:"column:ivr_menu_option_param"`
}

/*
WITH RECURSIVE t(id, ivr_menu_name, ivr_menu_extension, ivr_menu_greet_long_id, ivr_menu_digit_len) as (
	SELECT id, ivr_menu_name, ivr_menu_extension, ivr_menu_greet_long_id, ivr_menu_digit_len
	FROM call_ivr_menus
-- 	where ivr_menu_extension='40004004261000'

UNION ALL

	SELECT d.id, d.ivr_menu_name, d.ivr_menu_extension, d.ivr_menu_greet_long_id, d.ivr_menu_digit_len
	from call_ivr_menus d
	JOIN t on d.ivr_menu_parent_id = t.id
)
SELECT t.id, t.ivr_menu_name, t.ivr_menu_extension, t.ivr_menu_greet_long_id, t.ivr_menu_digit_len,
o.ivr_menu_option_digits, o.ivr_menu_option_param
from t
LEFT JOIN call_ivr_menu_options o on t.id=o.ivr_menu_id
ORDER BY t.ivr_menu_extension
*/
func GetAllIvrMenuDetail() []*Menu {

	sql := `WITH RECURSIVE t(id, ivr_menu_name, ivr_menu_extension, ivr_menu_greet_long_id, ivr_menu_digit_len) as (SELECT id, ivr_menu_name, ivr_menu_extension, ivr_menu_greet_long_id, ivr_menu_digit_len FROM %s UNION ALL SELECT d.id, d.ivr_menu_name, d.ivr_menu_extension, d.ivr_menu_greet_long_id, d.ivr_menu_digit_len from %s d	JOIN t on d.ivr_menu_parent_id = t.id) SELECT t.id, t.ivr_menu_name, t.ivr_menu_extension, t.ivr_menu_greet_long_id, t.ivr_menu_digit_len, o.ivr_menu_option_digits, (SELECT name from call_operation where id=o.ivr_menu_option_action_id) app, o.ivr_menu_option_param from t LEFT JOIN %s o on t.id=o.ivr_menu_id ORDER BY t.ivr_menu_extension`
	sql = fmt.Sprintf(sql, IvrMenuTBName(), IvrMenuTBName(), IvrMenuOptionTBName())

	menu := make([]*Menu, 0)
	ImplInstance.DB.Raw(sql).Scan(&menu)

	return menu
}
