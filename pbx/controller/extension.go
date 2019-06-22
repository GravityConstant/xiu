package controller

import (
	"fmt"
	"sort"

	"xiu/pbx/entity"
	"xiu/pbx/models"
	// "xiu/pbx/util"
)

func WriteExtensionToRedis() {
	extension := entity.Extension{}
	action := entity.Action{}

	dialplanDetail := models.GetAllDialplanDetail()
	keys := make([]string, 0)

	for key, item := range dialplanDetail {
		action.Order = item.DialplanDetailOrder
		// ivr另外处理
		if item.DialplanDetailApp == "submenu" {
			action.App = "ivr"
		} else {
			action.App = item.DialplanDetailApp
		}

		action.Data = item.DialplanDetailData

		// 第一次生成map要单独处理
		if key == 0 {
			keys = append(keys, item.DialplanNumber)
			extension.Name = item.DialplanName
			extension.Field = item.DialplanContext
			extension.Expression = item.DialplanNumber
			extension.Actions = append(extension.Actions, action)
			// 必须要先赋一个extension的值，否则下面判断map就为false了
			entity.MapExt[extension.Expression] = extension
		} else {
			if _, ok := entity.MapExt[item.DialplanNumber]; ok {
				// fmt.Println("ok=", item.DialplanNumber)
				extension.Actions = append(extension.Actions, action)
			} else {
				if len(extension.Expression) > 0 {
					// fmt.Println("new=", item.DialplanNumber, extension)
					keys = append(keys, item.DialplanNumber)
					// 这里才是完整的extension，因为action有可能多个，添加确实在上面
					entity.MapExt[extension.Expression] = extension
					// 清空extension
					extension = entity.Extension{}

				}
				extension.Name = item.DialplanName
				extension.Field = item.DialplanContext
				extension.Expression = item.DialplanNumber
				extension.Actions = append(extension.Actions, action)
				// 必须要先赋一个extension的值，否则下面判断map就为false了
				entity.MapExt[extension.Expression] = extension
			}
		}

		action = entity.Action{}
	}
	// 最后一个extension加入map
	if extension.Expression != "" {
		entity.MapExt[extension.Expression] = extension
	}
	for _, key := range keys {
		sort.Sort(entity.ByOrder(entity.MapExt[key].Actions))
		fmt.Println(key, entity.MapExt[key])
	}
	// util.Info("controller/extension.go", "32", entity.MapExt)
}

func WriteIvrMenuToRedis() {
	ivrMenu := models.GetAllIvrMenuDetail()

	for _, item := range ivrMenu {
		fmt.Println(item)
	}
}
