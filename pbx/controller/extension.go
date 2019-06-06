package controller

import (
	"xiu/pbx/entity"
	"xiu/pbx/models"
	"xiu/pbx/util"
)

func WriteExtensionToRedis() {
	extension := entity.Extension{}
	action := entity.Action{}

	dialplanDetail := models.GetAllDialplanDetail()
	for _, item := range dialplanDetail {
		action.Order = item.DialplanDetailOrder
		action.App = item.DialplanDetailApp
		action.Data = item.DialplanDetailData

		if ext, ok := entity.MapExt[item.DialplanNumber]; ok {
			entity.MapExt[item.DialplanNumber].Actions = append(entity.MapExt[item.DialplanNumber].Actions, action)
		} else {
			extension.Name = item.DialplanName
			extension.Field = item.DialplanContext
			extension.Expression = item.DialplanNumber
			extension.Actions = append(extension.Actions, action)

			entity.MapExt[item.DialplanNumber] = extension
		}
		extension = entity.Extension{}
		action = entity.Action{}
	}
	util.Info("controller/extension.go", "32", entity.MapExt)
}
