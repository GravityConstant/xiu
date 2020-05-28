package controller

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"xiu/pbx/entity"
	"xiu/pbx/models"
	"xiu/pbx/util"
	colorlog "xiu/util"
)

// 将Extension写入redis，如果失败，这放在全局变量entity.MapExt
func WriteExtensionToRedis() {
	// 定义局部的map[string]*Extension
	// 使用redis时，就被回收了，不使用redis，赋值给全局map
	me := make(map[string]*entity.Extension)
	// 添加一个没有路由时使用的Extension
	me["00000000"] = &entity.Extension{
		Name:       "4000000000",
		Field:      "destination_number",
		Expression: "00000000",
		Actions: []entity.Action{
			{
				Order: 1,
				App:   "hangup",
				Data:  "NO_ROUTE_DESTINATION",
			},
		},
	}
	// 设置extension模板
	extensionTpl := entity.Extension{
		Actions:         []entity.Action{},
		IsRecord:        true,
		IsSayJobnum:     true,
		IsSatisfySurvey: false,
	}
	// 默认export当前dialplan的id
	exportDialplanIdTpl := entity.Action{
		Order: 0,
		App:   "export",
		Data:  "dialplan_id=%d",
	}
	// 默认ivr呼叫铃音
	transferRingbackTpl := entity.Action{
		Order: 0,
		App:   "set",
		Data:  "transfer_ringback=/home/voices/rings/common/ivr_transfer.wav",
	}
	action := entity.Action{}

	// get db
	params := models.ExtensionQueryParam{}
	dialplanDetail := models.GetAllDialplanDetail(&params)
	// 用于写入redis
	keys := make([]string, 0)
	// 找不到extension
	keys = append(keys, "00000000")

	for _, item := range dialplanDetail {
		action.Order = item.DialplanDetailOrder
		// ivr另外处理
		if item.DialplanDetailApp == "submenu" {
			action.App = "ivr"
		} else {
			action.App = item.DialplanDetailApp
		}
		action.Data = item.DialplanDetailData

		if _, ok := me[item.DialplanNumber]; ok {
			me[item.DialplanNumber].Actions = append(me[item.DialplanNumber].Actions, action)
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
			// ivr外呼回铃音
			extension.Actions = append(extension.Actions, transferRingbackTpl)
			// 放入全局map
			me[extension.Expression] = &extension
			keys = append(keys, extension.Expression)
		}
		action = entity.Action{}
	}
	// 保存到redis
	if util.CheckRedis() {
		for _, key := range keys {
			sort.Sort(entity.ByOrder(me[key].Actions))
			if err := util.SetCache(key, me[key], 0); err == nil {
				colorlog.Success("save to redis success:", key)
			} else {
				colorlog.Success("save to redis fail:", key)
			}
		}
	} else {
		for _, key := range keys {
			colorlog.Success("save to redis fail:", key)
			sort.Sort(entity.ByOrder(me[key].Actions))
		}
		entity.MapExt = me
	}
}

func WriteExtensionToRedisV1() {
	extension := entity.Extension{}
	action := entity.Action{}

	// get data from db
	params := models.ExtensionQueryParam{}
	dialplanDetail := models.GetAllDialplanDetail(&params)
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

// 将IvrMenu写入redis，如果失败，这放在全局变量entity.MapMenu
func WriteIvrMenuToRedis() {
	// 局部的map[string]*Menu,使用redis时，不赋值给全局map
	mm := make(map[string]*entity.Menu)
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
	// 用于写入redis
	keys := make([]string, 0)
	// map[id]extension，为获取父级ivr的extension做准备
	idMap := make(map[int64]string)
	// 从数据库获取数据
	params := models.MenuQueryParam{}
	ivrMenu := models.GetAllIvrMenuDetail(&params)
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

		if _, ok := mm[item.Extension]; ok {
			if len(entry.Action) > 0 {
				mm[item.Extension].Entrys = append(mm[item.Extension].Entrys, entry)
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
			mm[item.Extension] = &menu
		}
	}
	// 保存到redis
	if util.CheckRedis() {
		for _, key := range keys {
			// fmt.Printf("%s: %v\n", key, mm[key])
			util.SetCache(key, mm[key], 0)
		}
	} else {
		entity.MapMenu = mm
	}
}

func PrintCache() {
	ext := entity.Extension{}
	if err := util.GetCache("28324284", &ext); err != nil {
		util.Error("controller/extension.go", "get error", err)
	}
	util.Debug("controller/extension.go", "28324284", ext)
	util.GetCache("28324285", &ext)
	util.Debug("controller/extension.go", "28324285", ext)
	util.GetCache("00000000", &ext)
	util.Debug("controller/extension.go", "00000000", ext)

	menu := entity.Menu{}
	util.GetCache("40004004261000", &menu)
	util.Debug("controller/extension.go", "40004004261000", menu)

	util.Debug("controller/extension.go", "is original MapExt?", entity.MapExt)
	util.Debug("controller/extension.go", "is original MapMenu?", entity.MapMenu)
}

// 得到dialplanNumber对应的Extension
func GetExtensionByDialplanNumber(dialplanNumber string) *entity.Extension {
	ext := &entity.Extension{}
	// first: get from redis
	// if err := util.GetCache(dialplanNumber, ext); err == nil {
	// 	return ext
	// } else {
	// 	util.Error("controller/extension.go", "get extension from redis error", err)
	// }
	// // second: get from entity.MapExt
	// if ext, ok := entity.MapExt[dialplanNumber]; ok {
	// 	return ext
	// } else {
	// 	util.Error("controller/extension.go", "get extension from map error")
	// }
	// third: get from database
	params := models.ExtensionQueryParam{DialplanNumber: dialplanNumber}
	dialplanDetail := models.GetAllDialplanDetail(&params)
	ext = GetExtensionByDialplanNumberResult(dialplanDetail)
	return ext
}

func GetExtensionByDialplanNumberResult(dialplanDetail []*models.Extension) *entity.Extension {
	// 添加一个没有路由时使用的Extension
	ext0 := entity.Extension{
		Name:       "4000000000",
		Field:      "destination_number",
		Expression: "00000000",
		Actions: []entity.Action{
			{
				Order: 1,
				App:   "hangup",
				Data:  "NO_ROUTE_DESTINATION",
			},
		},
	}
	if len(dialplanDetail) == 0 {
		return &ext0
	}
	// 设置extension模板
	extension := entity.Extension{
		Actions:         []entity.Action{},
		IsRecord:        true,
		IsSayJobnum:     true,
		IsSatisfySurvey: false,
	}
	// 默认export当前dialplan的id
	exportDialplanId := entity.Action{
		Order: 0,
		App:   "export",
		Data:  "dialplan_id=%d",
	}
	exportDialplanId.Data = fmt.Sprintf(exportDialplanId.Data, dialplanDetail[0].DialplanId)
	extension.Actions = append(extension.Actions, exportDialplanId)
	// 默认ivr呼叫铃音
	transferRingback := entity.Action{
		Order: 0,
		App:   "set",
		Data:  "transfer_ringback=/home/voices/rings/common/ivr_transfer.wav",
	}
	extension.Actions = append(extension.Actions, transferRingback)

	action := entity.Action{}
	for _, item := range dialplanDetail {
		if !item.DialplanEnabled {
			return &ext0
		}
		action.Order = item.DialplanDetailOrder
		// ivr另外处理
		if item.DialplanDetailApp == "submenu" {
			action.App = "ivr"
		} else {
			action.App = strings.TrimSpace(item.DialplanDetailApp)
		}
		action.Data = strings.TrimSpace(item.DialplanDetailData)

		if len(extension.Expression) > 0 {
			if strings.Compare(extension.Expression, strings.TrimSpace(item.DialplanNumber)) == 0 {
				extension.Actions = append(extension.Actions, action)
			} else {
				util.Error("controller/extension.go", "get one extension, but multi get", item.DialplanNumber)
			}

		} else {
			extension.Name = strings.TrimSpace(item.DialplanName)
			extension.Field = strings.TrimSpace(item.DialplanContext)
			extension.Expression = strings.TrimSpace(item.DialplanNumber)
			extension.Actions = append(extension.Actions, action)
		}
		action = entity.Action{}
	}
	// 保存到redis
	if util.CheckRedis() {
		sort.Sort(entity.ByOrder(extension.Actions))
		if err := util.SetCache(extension.Expression, extension, 0); err != nil {
			util.Info("controller/extension.go", "set redis cache fail", extension.Expression, err)
		} else {
			util.Info("controller/extension.go", "set redis cache success", extension.Expression)
		}
	} else {
		sort.Sort(entity.ByOrder(extension.Actions))
		entity.MapExt[extension.Expression] = &extension
	}
	colorlog.Success("current dialplan: %v\n", extension)
	return &extension
}

// 得到ivr_menu_extension对应的Menu
func GetMenuByExtension(extension string) *entity.Menu {
	menu := &entity.Menu{}
	defer func() {
		menu.Err = CheckMenuValid(menu)
		util.Info("controller/extension.go", "menu", menu)
	}()
	// // first: get from redis
	// if err := util.GetCache(extension, menu); err == nil {
	// 	return menu
	// } else {
	// 	util.Error("controller/extension.go", "get menu from redis error", err)
	// }
	// // second: get from entity.MapExt
	// if menu, ok := entity.MapMenu[extension]; ok {
	// 	return menu
	// } else {
	// 	util.Error("controller/extension.go", "get menu from map error")
	// }
	// third: get from database
	params := models.MenuQueryParam{Extension: extension}
	menuDetail := models.GetAllIvrMenuDetail(&params)
	menu = GetMenuByExtensionResult(menuDetail, extension)
	return menu
}

func CheckMenuValid(menu *entity.Menu) error {
	if fileInfo, err := os.Stat(menu.File); os.IsNotExist(err) {
		return entity.ErrIvrFileNotExist
	} else {
		if fileInfo.IsDir() {
			return entity.ErrIvrFileNotExist
		}
	}
	if len(menu.Entrys) == 0 {
		return entity.ErrNoEntry
	}
	return nil
}

func GetMenuByExtensionResult(menuDetail []*models.Menu, extension string) *entity.Menu {
	if len(menuDetail) == 0 {
		return &entity.Menu{}
	}
	// 局部的map[string]*Menu,使用redis时，不赋值给全局map
	mm := make(map[string]*entity.Menu)
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
	// 用于写入redis
	keys := make([]string, 0)
	// map[id]extension，为获取父级ivr的extension做准备
	idMap := make(map[int64]string)
	for _, item := range menuDetail {
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
	for _, item := range menuDetail {
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

		if _, ok := mm[item.Extension]; ok {
			if len(entry.Action) > 0 {
				mm[item.Extension].Entrys = append(mm[item.Extension].Entrys, entry)
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
			mm[item.Extension] = &menu
		}
	}
	// 保存到redis
	if util.CheckRedis() {
		for _, key := range keys {
			// fmt.Printf("%s: %v\n", key, mm[key])
			util.SetCache(key, mm[key], 0)
		}
	} else {
		entity.MapMenu = mm
	}
	return mm[extension]
}
