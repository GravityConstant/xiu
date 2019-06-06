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
