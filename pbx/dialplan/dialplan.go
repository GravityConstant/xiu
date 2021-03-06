package dialplan

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"xiu/esl"
	"xiu/pbx/controller"
	"xiu/pbx/entity"
	"xiu/pbx/util"
	colorlog "xiu/util"
)

func InitExtension() {
	controller.WriteExtensionToRedis()

	return

	ext0 := entity.Extension{
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
	entity.MapExt["00000000"] = &ext0

	ext1 := entity.Extension{
		Field:      "destination_number",
		Expression: "28324284",
		Actions: []entity.Action{
			{
				Order: 1,
				App:   "set",
				Data:  "hangup_after_bridge=false",
			},
			{
				Order: 2,
				App:   "set", // 回铃太慢了
				Data:  "ringback=/home/voices/rings/pbx/testRing.wav",
			},
			{
				Order: 3,
				App:   "bridge",
				Data:  "{ignore_early_media=ring_ready,originate_continue_on_timeout=true,sip_h_Diversion=<sip:28324284@ip>}[leg_timeout=15]sofia/gateway/zqzj/13675017141|[leg_timeout=15]sofia/gateway/zqzj/83127866", // |[leg_timeout=15]sofia/gateway/zqzj/83127866"
			},
		},
		IsRecord:    true,
		IsSayJobnum: true,
		// IsSatisfySurvey: true,
	}
	sort.Sort(entity.ByOrder(ext1.Actions))
	ext2 := entity.Extension{
		Field:      "destination_number",
		Expression: "28324285",
		Actions: []entity.Action{
			{
				Order: 1,
				App:   "set",
				Data:  "hangup_after_bridge=false",
			},
			{
				Order: 1,
				App:   "answer",
			},
			{
				Order: 2,
				App:   "ivr", // play_and_get_digits
				Data:  "283242851000",
				// <min> <max> <tries> <timeout> <terminators> <file> <invalid_file>
				// Data: `3 3 3 6000 # /home/voices/rings/pbx/fu_xia_zhang_quan.wav /home/voices/rings/common/input_error.wav foo_dtmf_digits \d+ 3000`,
			},
		},
		IsRecord:        true,
		IsSayJobnum:     true,
		IsSatisfySurvey: true,
	}
	sort.Sort(entity.ByOrder(ext2.Actions))

	entity.MapExt["28324284"] = &ext1
	entity.MapExt["28324285"] = &ext2
}

func PrepareExtension(dialplanNumber string) <-chan entity.Extension {
	items := make(chan entity.Extension)
	go func() {
		extension := controller.GetExtensionByDialplanNumber(dialplanNumber)
		items <- *extension
		close(items)
	}()
	return items
}

func ExecuteExtension(con *esl.Connection, UId string, items <-chan entity.Extension) <-chan interface{} {
	extraHandle := make(chan interface{})
	go func() {
		util.Info("execute", "extension")
		for item := range items {
			for _, action := range item.Actions {
				switch action.App {
				case "ivr", "bridge":
					extraHandle <- action
					continue
				}
				if action.Sync == true {
					con.ExecuteSync(action.App, UId, action.Data)
				} else {
					con.Execute(action.App, UId, action.Data)
				}
			}
		}
		close(extraHandle)
	}()
	return extraHandle
}

func InitIvrMenu() {
	controller.WriteIvrMenuToRedis()
	return
	menu0 := entity.Menu{
		Name:         "40004004261000",
		Min:          1,
		Max:          1,
		Tries:        3,
		Timeout:      3000,
		Terminators:  "#",
		File:         `/home/voices/rings/pbx/ivr_first.wav`,
		InvalidFile:  `/home/voices/rings/common/input_error.wav`,
		VarName:      `foo_dtmf_digits`,
		Regexp:       `\d`,
		DigitTimeout: 3000,
		Entrys: []entity.Entry{
			{
				Action: "menu-sub",
				Digits: "1",
				Param:  "283242851002",
			},
		},
	}
	menu1 := entity.Menu{
		Name:         "283242851002",
		Min:          1,
		Max:          1,
		Tries:        3,
		Timeout:      3000,
		Terminators:  "#",
		File:         `/home/voices/rings/pbx/fujian_jiangsu.wav`,
		InvalidFile:  `/home/voices/rings/common/input_error.wav`,
		VarName:      `foo_dtmf_digits`,
		Regexp:       `\d|\*`,
		DigitTimeout: 3000,
		Entrys: []entity.Entry{
			{
				Action: "menu-sub",
				Digits: "1",
				Param:  "283242851001",
			},
			{
				Action: "menu-top",
				Digits: "*",
				Param:  "40004004261000",
			},
		},
	}
	menu2 := entity.Menu{
		Name:         "283242851001",
		Min:          1,
		Max:          3,
		Tries:        3,
		Timeout:      3000,
		Terminators:  "#",
		File:         `/home/voices/rings/pbx/fu_xia_zhang_quan.wav`,
		InvalidFile:  `/home/voices/rings/common/input_error.wav`,
		VarName:      `foo_dtmf_digits`,
		Regexp:       `\d{3}|\*`,
		DigitTimeout: 3000,
		Entrys: []entity.Entry{
			{
				Action: "menu-exec-app",
				Digits: "801",
				Param:  "bridge {ignore_early_media=ring_ready,sip_h_Diversion=<sip:28324285@ip>}[leg_timeout=15]sofia/gateway/zqzj/13675017141|[leg_timeout=15]sofia/gateway/zqzj/83127866",
			},
			{
				Action: "menu-exec-app",
				Digits: "802",
				Param:  "bridge {ignore_early_media=ring_ready,sip_h_Diversion=<sip:28324285@ip>}[leg_timeout=15]sofia/gateway/zqzj/83127866",
			},
			{
				Action: "menu-top",
				Digits: "*",
				Param:  "283242851002",
			},
		},
	}
	entity.MapMenu["40004004261000"] = &menu0
	entity.MapMenu["283242851002"] = &menu1
	entity.MapMenu["283242851001"] = &menu2
}

func PrepareIvrMenu(dialplanNumber, callerNumber, ivrMenuExtension, dtmfDigits string) <-chan interface{} {
	items := make(chan interface{})
	go func() {
		// 返回时，关闭通道
		defer close(items)
		// 获取menu
		menu := controller.GetMenuByExtension(ivrMenuExtension)
		// 错误处理
		switch menu.Err {
		case entity.ErrIvrFileNotExist:
			items <- entity.MenuExecApp{
				App:  "playback",
				Data: "/home/voices/rings/common/ivr-invalid_sound_prompt.wav",
			}
			return
		case entity.ErrNoEntry:
			items <- entity.MenuExecApp{
				App:  "playback",
				Data: menu.File,
			}
			return
		}
		// 正常流程
		if dtmfDigits == "" { // 首层ivr处理
			items <- *menu
		} else if dtmfDigits == "digitTimeout" {
			// 按键输入不完整，重新播放本层
			items <- *menu
		} else {
			util.Info("dialplan/dialplan.go", "dtmf digit", dtmfDigits)
			// colorlog.Info("dtmfDigits: %s", dtmfDigits)
			digitNotFound := true
			for _, entry := range menu.Entrys {
				if dtmfDigits == entry.Digits {
					switch entry.Action {
					case "menu-exec-app": // 执行app
						params := strings.Split(entry.Param, " ")
						action := entity.MenuExecApp{
							App:   params[0],
							Data:  params[1],
							Extra: []string{dialplanNumber, callerNumber, entry.Extra["ResponseType"]},
						}
						items <- action
					case "menu-sub": // 跳到下层ivr
						menu = controller.GetMenuByExtension(entry.Param)
						items <- *menu
					case "menu-top": // 跳到上层ivr
						menu = controller.GetMenuByExtension(entry.Param)
						items <- *menu
					default:
						items <- "action_not_found"
					}
					digitNotFound = false
					break
				}
			}
			if digitNotFound { // 按键输入错误，重新播放本层
				items <- *menu
			}
		}
	}()
	return items
}

// 已弃用，原本用来修复按键超时的问题。
func RepairMenuTimeout(con *esl.Connection, UId string, entrys <-chan interface{}) {
	go func() {
		util.Info("repair", "menu timeout")
		for entry := range entrys {
			switch item := entry.(type) {
			case entity.Menu:
				if len(item.File) == 0 {
					con.Execute("hangup", UId, "NO_ROUTE_DESTINATION")
					return
				}
				app := "play_and_get_digits"
				// <min> <max> <tries> <timeout> <terminators> <file> <invalid_file> [<var_name> [<regexp> [<digit_timeout> [<transfer_on_failure>]]]]
				data := `%d %d %d %d %s %s %s %s %s %d`
				data = fmt.Sprintf(data, item.Min, item.Max, item.Tries, item.Timeout, item.Terminators, item.File, item.InvalidFile, item.VarName, item.Regexp, item.DigitTimeout)
				con.ExecuteSync("playback", UId, "/home/voices/rings/common/input_error.wav")
				con.Execute(app, UId, data)
			default:
				con.Execute("hangup", UId, "")
			}
		}
	}()
}

func ExecuteMenuEntry(con *esl.Connection, UId string, entrys <-chan interface{}) {
	go func() {
		for entry := range entrys {
			util.Info("execute", "menu entry", entry)
			switch item := entry.(type) {
			case entity.Menu:
				if len(item.File) == 0 {
					con.Execute("hangup", UId, "NO_ROUTE_DESTINATION")
					return
				}
				app := "play_and_get_digits"
				// <min> <max> <tries> <timeout> <terminators> <file> <invalid_file> [<var_name> [<regexp> [<digit_timeout> [<transfer_on_failure>]]]]
				data := `%d %d %d %d %s %s %s %s %s %d`
				data = fmt.Sprintf(data, item.Min, item.Max, item.Tries, item.Timeout, item.Terminators, item.File, item.InvalidFile, item.VarName, item.Regexp, item.DigitTimeout)
				con.Execute(app, UId, data)
				entity.UIdDtmfSyncMap.Store(UId, item.Name)
			case entity.MenuExecApp:
				switch item.App {
				case "bridge":
					bridgeItem := PrepareBridge(item.Extra[0], item.Extra[1], item.Data, map[string]string{"ResponseType": item.Extra[2]})
					ExecuteBridge(con, UId, bridgeItem)
					// 使用playback,外呼的时候还是没有回铃音
					// con.Execute("playback", UId, "/home/voices/rings/common/ivr_transfer.wav")
					// 李工使用uuid_broadcast, 我使用了transfer_ringback
				case "playback":
					con.ExecuteSync(item.App, UId, item.Data)
					con.Execute("hangup", UId, "")
				default:
					con.Execute(item.App, UId, item.Data)
				}
			case string:
				con.ExecuteSync("playback", UId, "/home/voices/rings/pbx/no_number.wav")
				con.Execute("hangup", UId, "")
			default:
				con.Execute("hangup", UId, "")
			}
		}
	}()
}

// 已弃用，原来只有简单的回铃
func ExecuteIvrMenu(con *esl.Connection, UId string, items <-chan entity.Menu) {
	go func() {
		util.Info("execute", "ivr menu")
		for item := range items {
			if len(item.File) == 0 {
				return
			}
			app := "play_and_get_digits"
			// <min> <max> <tries> <timeout> <terminators> <file> <invalid_file> [<var_name> [<regexp> [<digit_timeout> [<transfer_on_failure>]]]]
			data := `%d %d %d %d %s %s %s %s %s %d`
			data = fmt.Sprintf(data, item.Min, item.Max, item.Tries, item.Timeout, item.Terminators, item.File, item.InvalidFile, item.VarName, item.Regexp, item.DigitTimeout)
			con.Execute(app, UId, data)
			entity.UIdDtmfSyncMap.Store(UId, item.Name)
		}
	}()
}

func PrepareSayJobnum(dialplanNumber, callee string) <-chan interface{} {
	items := make(chan interface{})
	go func() {
		extension := controller.GetExtensionByDialplanNumber(dialplanNumber)
		if extension.IsSayJobnum {
			jn := &controller.Jobnum{}
			jn.GetJobnumRingPath(dialplanNumber, callee)
			if len(jn.RingPaths) > 0 {
				sayJobnum := entity.SayJobnum{
					PrefixFile: "/home/voices/rings/common/job_number_prefix.wav",
					SuffixFile: "/home/voices/rings/common/job_number_suffix.wav",
					Jobnum:     jn.RingPaths,
				}
				items <- sayJobnum
			} else {
				items <- "none_jobnum"
			}
		} else {
			items <- "no_need_say_jobnum"
		}

		close(items)
	}()
	return items
}

// 直转的：playback(file_string:///home/voices/default/job_number_prefix.wav!/home/voices/default/0.wav!/home/voices/default/1.wav!/home/voices/default/0.wav!/home/voices/default/2.wav!/home/voices/default/job_number_suffix.wav)
// ivr的：uuid_broadcast李浩好像是这么用的
func ExecuteSayJobnum(con *esl.Connection, UId string, items <-chan interface{}) {
	go func() {
		util.Info("execute", "say jobnum")
		var err error
		for item := range items {
			switch t := item.(type) {
			case entity.SayJobnum:
				// con.ExecuteSync("playback", UId, t.PrefixFile)
				// for _, val := range t.Jobnum {
				// 	con.ExecuteSync("playback", UId, val)
				// }
				// _, err = con.ExecuteSync("playback", UId, t.SuffixFile)
				// if err != nil {
				// 	util.Error("dialplan/dialplan.go", "359", err)
				// }
				// 直转的话，被叫先answer，所以先听到报工号，被叫说话，但主叫报工号还没完成，可以被打断。

				// 使用file_string播放音乐
				path := `file_string://%s!%s!%s`
				var jobnumMusic string
				for _, val := range t.Jobnum {
					if len(jobnumMusic) == 0 {
						jobnumMusic += val
					} else {
						jobnumMusic += `!` + val
					}

				}
				path = fmt.Sprintf(path, t.PrefixFile, jobnumMusic, t.SuffixFile)
				// uuid_broadcast <uuid> <path> [aleg|bleg|both]
				paramsFormat := `%s %s both`
				_, err = con.Api("uuid_broadcast", fmt.Sprintf(paramsFormat, UId, path))
				if err != nil {
					util.Error("dialplan/dialplan.go", "370", err)
				}
			}
		}
	}()
}

func PrepareSatisfySurvey(dialplanNumber, callee string) <-chan interface{} {
	items := make(chan interface{})
	go func() {
		// if entity.MapExt[dialplanNumber].IsSatisfySurvey == true {
		// 不在dialplan中取值了，而在绑定号中
		ss := &controller.SatisfySurvey{}
		ss.IsOpenSatisfySurvey(dialplanNumber, callee)
		if ss.IsOpen {
			satisfySurvey := entity.SatisfySurvey{
				PrefixFile: "/home/voices/rings/common/satisfy_survey.wav",
				SuffixFile: "/home/voices/rings/common/satisfy_survey_end.wav",
			}
			items <- satisfySurvey
		} else {
			items <- "no_need_satisfy_survey"
		}
		close(items)
	}()
	return items
}
func ExecuteSatisfySurvey(con *esl.Connection, UId string, items <-chan interface{}) {
	go func() {
		util.Info("execute", "satisfy survey.")
		for item := range items {
			switch t := item.(type) {
			case entity.SatisfySurvey:
				params := `1 1 %s foo_satisfy_survey_digits 3000 #`
				params = fmt.Sprintf(params, t.PrefixFile)
				con.Execute("read", UId, params)
			case string:
				// no need handle
			default:
				con.Execute("hangup", UId, "")
			}
		}
	}()
}
func HandleSatisfySurvey(con *esl.Connection, UId string, satisfySurveyDigits string) {
	go func() {
		util.Info("handle", "satisfy survey.", "satisfySurveyDigits"+satisfySurveyDigits)
		if satisfySurveyDigits == "unknown" {
			con.ExecuteSync("speak", UId, `tts_commandline|Mandarin|未收到按键，但还是谢谢您的评价！`)
			con.Execute("hangup", UId, "")
		} else {
			switch satisfySurveyDigits {
			case "1", "2", "3":
				con.ExecuteSync("playback", UId, "/home/voices/rings/common/satisfy_survey_end.wav")
			default:
				con.ExecuteSync("speak", UId, `tts_commandline|Mandarin|按键按错了，但还是谢谢您的评价！`)
			}
			con.Execute("hangup", UId, "")
			ss := new(controller.SatisfySurvey)
			ss.SaveSatisfySurveyResult(UId, satisfySurveyDigits)
			colorlog.Success("%s %s insert db success", UId, satisfySurveyDigits)
		}
	}()
}

func PrepareRecord(dialplanNumber, caller, callee string) <-chan interface{} {
	items := make(chan interface{})
	go func() {
		extension := controller.GetExtensionByDialplanNumber(dialplanNumber)
		if extension.IsRecord {
			// 路径前缀
			commonPath := `/home/voices/records`

			now := time.Now()
			year, month, day := now.Date()
			// 年文件夹
			yearStr := strconv.Itoa(year)
			// 月文件夹
			monthStr := strconv.Itoa(int(month))
			// 日文件夹
			dayStr := strconv.Itoa(day)
			// dialplanNumber文件夹
			// 文件名: 主叫-被叫-时间
			nowStr := now.Format("20060102150405000")
			filename := `%s-%s-%s.wav`
			filename = fmt.Sprintf(filename, caller, callee, nowStr)

			filePath := filepath.Join(yearStr, monthStr, dayStr, dialplanNumber, filename)
			record := entity.Record{
				Name:       "record_file",
				PrefixPath: commonPath,
				File:       filePath,
			}
			items <- record
		} else {
			items <- "no_need_record"
		}
		close(items)
	}()
	return items
}

func ExecuteRecord(con *esl.Connection, UId string, items <-chan interface{}) {
	go func() {
		util.Info("execute", "record")
		for item := range items {
			switch t := item.(type) {
			case entity.Record:
				// uuid_setvar <uuid> <varname> [value]
				paramsFormat := `%s %s %s`
				con.BgApi("uuid_setvar", fmt.Sprintf(paramsFormat, UId, t.Name, t.File))
				// uuid_record <uuid> [start|stop|mask|unmask] <path> [<limit>]
				con.BgApi("uuid_record", fmt.Sprintf(paramsFormat, UId, "start", filepath.Join(t.PrefixPath, t.File)))
			case string:

			}
		}
	}()
}

func PrepareBridge(dialplanNumber, callerNumber, bpIds string, params map[string]string) <-chan interface{} {
	items := make(chan interface{})
	go func() {
		var rs int
		if v, ok := params["ResponseType"]; ok {
			rs, _ = strconv.Atoi(v)
		}
		outcall := &controller.Outcall{}
		outcall.GetCallString(dialplanNumber, callerNumber, bpIds, rs)

		var bridge entity.Action
		if len(outcall.CallString) == 0 {
			bridge = entity.Action{
				App:  "playback",
				Data: "/home/voices/rings/common/busy.wav",
			}
		} else {
			bridge = entity.Action{
				App:  "bridge",
				Data: outcall.CallString,
			}
		}
		util.Info("dialplan/dialplan.go", "call string", outcall.CallString)
		// colorlog.Info("outbound call: %s", outcall.CallString)
		items <- bridge
		close(items)
	}()
	return items
}
func ExecuteBridge(con *esl.Connection, UId string, items <-chan interface{}) {
	go func() {
		util.Info("execute", "bridge")
		for item := range items {
			switch t := item.(type) {
			case entity.Action:
				switch t.App {
				case "bridge":
					if _, err := con.Execute(t.App, UId, t.Data); err != nil {
						util.Error("dialplan/dialplan.go", "execute bridge", err)
					}
				case "playback":
					if _, err := con.ExecuteSync(t.App, UId, t.Data); err != nil {
						util.Error("dialplan/dialplan.go", "execute playback", err)
					}
					if _, err := con.Execute("hangup", UId, ""); err != nil {
						util.Error("dialplan/dialplan.go", "execute hangup", err)
					}
				}
			}
		}
	}()
}
