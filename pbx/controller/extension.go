package controller

import (
	"fmt"
	"sort"

	"xiu/pbx/entity"
	"xiu/pbx/models"
	// "xiu/pbx/util"
)

func WriteExtensionToRedis() {
	extensionTpl := entity.Extension{
		Actions:         []entity.Action{},
		IsRecord:        true,
		IsSayJobnum:     true,
		IsSatisfySurvey: false,
	}
	exportDialplanIdTpl := entity.Action{
		Order: 0,
		App:   "export",
		Data:  "dialplan_id=%d",
	}
	action := entity.Action{}

	dialplanDetail := models.GetAllDialplanDetail()
	// 为了打印好看
	keys := make([]string, 0)

	for _, item := range dialplanDetail {
		action.Order = item.DialplanDetailOrder
		// ivr另外处理
		if item.DialplanDetailApp == "submenu" {
			action.App = "ivr"
		} else {
			action.App = item.DialplanDetailApp
		}
		action.Data = item.DialplanDetailData

		if _, ok := entity.MapExt[item.DialplanNumber]; ok {
			entity.MapExt[item.DialplanNumber].Actions = append(entity.MapExt[item.DialplanNumber].Actions, action)
		} else {
			// 赋值初始模板
			extension := extensionTpl

			extension.Name = item.DialplanName
			extension.Field = item.DialplanContext
			extension.Expression = item.DialplanNumber
			extension.Actions = append(extension.Actions, action)
			// export dialplan id
			exportDialplanId := exportDialplanIdTpl
			exportDialplanId.Data = fmt.Sprintf(exportDialplanId.Data, item.DialplanId)
			extension.Actions = append(extension.Actions, exportDialplanId)
			// 放入全局map
			entity.MapExt[extension.Expression] = &extension
			keys = append(keys, extension.Expression)
		}
		action = entity.Action{}
	}
	for _, key := range keys {
		sort.Sort(entity.ByOrder(entity.MapExt[key].Actions))
		fmt.Println(key, entity.MapExt[key])
	}
	// util.Info("controller/extension.go", "32", entity.MapExt)
}

func WriteExtensionToRedisV1() {
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
			entity.MapExt[extension.Expression] = &extension
		} else {
			if _, ok := entity.MapExt[item.DialplanNumber]; ok {
				// fmt.Println("ok=", item.DialplanNumber)
				extension.Actions = append(extension.Actions, action)
			} else {
				if len(extension.Expression) > 0 {
					// fmt.Println("new=", item.DialplanNumber, extension)
					keys = append(keys, item.DialplanNumber)
					// 这里才是完整的extension，因为action有可能多个，添加确实在上面
					entity.MapExt[extension.Expression] = &extension
					// 清空extension
					extension = entity.Extension{}

				}
				extension.Name = item.DialplanName
				extension.Field = item.DialplanContext
				extension.Expression = item.DialplanNumber
				extension.Actions = append(extension.Actions, action)
				// 必须要先赋一个extension的值，否则下面判断map就为false了
				entity.MapExt[extension.Expression] = &extension
			}
		}

		action = entity.Action{}
	}
	// 最后一个extension加入map
	if extension.Expression != "" {
		entity.MapExt[extension.Expression] = &extension
	}
	for _, key := range keys {
		sort.Sort(entity.ByOrder(entity.MapExt[key].Actions))
		// fmt.Println(key, entity.MapExt[key])
	}
	// util.Info("controller/extension.go", "32", entity.MapExt)
}

func WriteIvrMenuToRedis() {
	ivrMenu := models.GetAllIvrMenuDetail()

	// 跳到上级ivr
	entryTop := entity.Entry{
		Action: "menu-top",
		Digits: "*",
	}
	// 跳到下级ivr
	entrySub := entity.Entry{
		Action: "menu-sub",
	}
	// 执行app
	entryApp := entity.Entry{
		Action: "menu-exec-app",
		Param:  "%s %s",
	}
	// keys: 用于打印
	keys := make([]string, 0)
	// map[id]extension，为获取父级ivr的extension做准备
	idMap := make(map[int64]string)
	for _, item := range ivrMenu {
		idMap[item.Id] = item.Extension
	}
	// 初始化单个变量，供for使用
	entry := entity.Entry{}
	menuTpl := entity.Menu{
		Tries:        3,
		Timeout:      3000,
		Terminators:  "#",
		File:         `/home/voices/rings/uploads/%s`,
		InvalidFile:  `/home/voices/rings/common/input_error.wav`,
		VarName:      `foo_dtmf_digits`,
		Regexp:       `\d{%d}%s`,
		DigitTimeout: 3000,
		Entrys:       []entity.Entry{},
	}

	// 整理成map[extension]menu，注入全局变量entity.MapMenu
	for _, item := range ivrMenu {
		menu := menuTpl
		// ivr的处理动作
		switch item.App {
		case "submenu":
			entry = entrySub
			entry.Digits = item.Digits
			entry.Param = item.Param
		case "bridge":
			entry = entryApp
			entry.Digits = item.Digits
			entry.Param = fmt.Sprintf(entryApp.Param, "bridge", item.Param)
		default:
			entry = entity.Entry{}
		}

		if _, ok := entity.MapMenu[item.Extension]; ok {
			if len(entry.Action) > 0 {
				entity.MapMenu[item.Extension].Entrys = append(entity.MapMenu[item.Extension].Entrys, entry)
			}
		} else {
			menu.Name = item.Extension
			menu.Min = item.DigitLen
			menu.Max = item.DigitLen
			menu.File = fmt.Sprintf(menu.File, item.File)
			if len(entry.Action) > 0 {
				menu.Entrys = append(menu.Entrys, entry)
			}
			if item.ParentId == 0 {
				menu.Regexp = fmt.Sprintf(menu.Regexp, item.DigitLen, "")
			} else {
				menu.Regexp = fmt.Sprintf(menu.Regexp, item.DigitLen, `|\*`)
				entryTop.Param = idMap[item.ParentId]
				menu.Entrys = append(menu.Entrys, entryTop)
			}
			keys = append(keys, item.Extension)
			entity.MapMenu[item.Extension] = &menu
		}
	}
	// 打印
	// for _, key := range keys {
	// 	fmt.Printf("%s: %v\n", key, entity.MapMenu[key])
	// }
}
