package dialplan

import (
	"fmt"
	"sort"
	"strings"
	"xiu/util"

	"github.com/vma/esl"
	// "xiu/pbx/controller"
)

type Extension struct {
	Field       string
	Expression  string
	Actions     []Action
	IsSayJobnum bool
}

type Action struct {
	Order    int64
	App      string
	Data     string
	Sync     bool
	Executed bool
}

type ByOrder []Action

func (a ByOrder) Len() int           { return len(a) }
func (a ByOrder) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByOrder) Less(i, j int) bool { return a[i].Order < a[j].Order }

// [dialplanNumber]Extension
var MapExt = make(map[string]Extension)

func InitExtension() {
	ext0 := Extension{
		Field:      "destination_number",
		Expression: "00000000",
		Actions: []Action{
			{
				Order: 4,
				App:   "hangup",
				Data:  "NO_ROUTE_DESTINATION",
			},
		},
	}
	sort.Sort(ByOrder(ext0.Actions))
	ext1 := Extension{
		Field:      "destination_number",
		Expression: "28324284",
		Actions: []Action{
			{
				Order: 1,
				App:   "set",
				Data:  "hangup_after_bridge=true",
			},
			{
				Order: 2,
				App:   "set", // 回铃太慢了
				Data:  "ringback=/home/voices/rings/pbx/testRing.wav",
			},
			{
				Order: 4,
				App:   "bridge",
				Data:  "{ignore_early_media=ring_ready,originate_continue_on_timeout=true,sip_h_Diversion=<sip:28324284@ip>}[leg_timeout=15]sofia/gateway/zqzj/13675017141", // |[leg_timeout=15]sofia/gateway/zqzj/83127866"
			},
		},
		IsSayJobnum: true,
	}
	sort.Sort(ByOrder(ext1.Actions))
	ext2 := Extension{
		Field:      "destination_number",
		Expression: "28324285",
		Actions: []Action{
			{
				Order: 1,
				App:   "set",
				Data:  "hangup_after_bridge=true",
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
		IsSayJobnum: true,
	}
	sort.Sort(ByOrder(ext2.Actions))
	MapExt["00000000"] = ext0
	MapExt["28324284"] = ext1
	MapExt["28324285"] = ext2
}

func PrepareExtension(dialplanNumber string) <-chan Extension {
	items := make(chan Extension)
	go func() {
		var extension Extension
		if ext, ok := MapExt[dialplanNumber]; ok {
			extension = ext
		} else {
			extension = MapExt["00000000"]
		}
		items <- extension
		close(items)
	}()
	return items
}

func ExecuteExtension(con *esl.Connection, UId string, items <-chan Extension, hasIvr chan string) {
	go func() {
		for item := range items {
		END:
			for _, action := range item.Actions {
				switch action.App {
				case "ivr":
					hasIvr <- action.Data
					break END
				}
				if action.Sync == true {
					con.ExecuteSync(action.App, UId, action.Data)
				} else {
					con.Execute(action.App, UId, action.Data)
				}
			}
		}
		hasIvr <- ""
	}()
}

type Menu struct {
	Name         string //
	Min          int    // Minimum number of digits to fetch (minimum value of 0)
	Max          int    // Maximum number of digits to fetch (maximum value of 128)
	Tries        int    // number of tries for the sound to play
	Timeout      int    // Number of milliseconds to wait for a dialed response after the file playback ends and before PAGD does a retry.
	Terminators  string //  digits used to end input if less than <max> digits have been pressed. If it starts with '=', then a terminator must be present for the input to be accepted (Since FS 1.2). (Typically '#', can be empty). Add '+' in front of terminating digit to always append it to the result variable specified in var_name.
	File         string // Sound file to play to prompt for digits to be dialed by the caller; playback can be interrupted by the first dialed digit (can be empty or the special string "silence" to omit the message).
	InvalidFile  string // Sound file to play when digits don't match the regexp (can be empty to omit the message).
	VarName      string // Channel variable into which valid digits should be placed (optional, no variable is set by default. See also 'var_name_invalid' below).
	Regexp       string // Regular expression to match digits (optional, an empty string allows all input (default)).
	DigitTimeout int    // Inter-digit timeout; number of milliseconds allowed between digits in lieu of dialing a terminator digit; once this number is reached, PAGD assumes that the caller has no more digits to dial (optional, defaults to the value of <timeout>).
	Entrys       []Entry
}

type Entry struct {
	Action string
	Digits string
	Param  string
}

var MapMenu = make(map[string]Menu)
var DtmfDigits = make(map[string]string)

func InitIvrMenu() {
	menu0 := Menu{
		Name:         "283242851000",
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
		Entrys: []Entry{
			{
				Action: "menu-sub",
				Digits: "1",
				Param:  "283242851002",
			},
		},
	}
	menu1 := Menu{
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
		Entrys: []Entry{
			{
				Action: "menu-sub",
				Digits: "1",
				Param:  "283242851001",
			},
			{
				Action: "menu-top",
				Digits: "*",
				Param:  "283242851000",
			},
		},
	}
	menu2 := Menu{
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
		Entrys: []Entry{
			{
				Action: "menu-exec-app",
				Digits: "801",
				Param:  "bridge {ignore_early_media=ring_ready,sip_h_Diversion=<sip:28324285@ip>}sofia/gateway/zqzj/13675017141",
			},
			{
				Action: "menu-exec-app",
				Digits: "802",
				Param:  "bridge {ignore_early_media=ring_ready,sip_h_Diversion=<sip:28324285@ip>}sofia/gateway/zqzj/17750409737",
			},
			{
				Action: "menu-top",
				Digits: "*",
				Param:  "283242851002",
			},
		},
	}
	MapMenu["283242851000"] = menu0
	MapMenu["283242851002"] = menu1
	MapMenu["283242851001"] = menu2
}

func PrepareIvrMenu(dialplanNumber, ivrMenuExtension, dtmfDigits string) <-chan interface{} {
	items := make(chan interface{})
	go func() {
		if dtmfDigits == "" { // 首层ivr处理
			items <- MapMenu[ivrMenuExtension]
		} else if dtmfDigits == "digitTimeout" {
			// 按键输入不完整，重新播放本层
			items <- MapMenu[ivrMenuExtension]
		} else {
			util.Info("dtmfDigits: %s", dtmfDigits)
			digitNotFound := true
			for _, entry := range MapMenu[ivrMenuExtension].Entrys {
				if dtmfDigits == entry.Digits {
					switch entry.Action {
					case "menu-exec-app": // 执行app
						params := strings.Split(entry.Param, " ")
						action := Action{
							App:  params[0],
							Data: params[1],
						}
						items <- action
					case "menu-sub": // 跳到下层ivr
						items <- MapMenu[entry.Param]
					case "menu-top": // 跳到上层ivr
						items <- MapMenu[entry.Param]
					default:
						items <- "action_not_found"
					}
					digitNotFound = false
					break
				}
			}
			if digitNotFound { // 按键输入错误，重新播放本层
				items <- MapMenu[ivrMenuExtension]
			}
		}
		close(items)
	}()
	return items
}

func RepairMenuTimeout(con *esl.Connection, UId string, entrys <-chan interface{}) {
	go func() {
		for entry := range entrys {
			switch item := entry.(type) {
			case Menu:
				if len(item.File) == 0 {
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
			switch item := entry.(type) {
			case Menu:
				if len(item.File) == 0 {
					return
				}
				app := "play_and_get_digits"
				// <min> <max> <tries> <timeout> <terminators> <file> <invalid_file> [<var_name> [<regexp> [<digit_timeout> [<transfer_on_failure>]]]]
				data := `%d %d %d %d %s %s %s %s %s %d`
				data = fmt.Sprintf(data, item.Min, item.Max, item.Tries, item.Timeout, item.Terminators, item.File, item.InvalidFile, item.VarName, item.Regexp, item.DigitTimeout)
				con.Execute(app, UId, data)
				DtmfDigits[UId] = item.Name
			case Action:
				con.Execute(item.App, UId, item.Data)
				con.Execute("playback", UId, "/home/voices/rings/common/ivr_transfer.wav")
			case string:
				con.ExecuteSync("playback", UId, "/home/voices/rings/pbx/no_number.wav")
				con.Execute("hangup", UId, "")
			default:
				con.Execute("hangup", UId, "")
			}
		}
	}()
}

func ExecuteIvrMenu(con *esl.Connection, UId string, items <-chan Menu) {
	go func() {
		for item := range items {
			if len(item.File) == 0 {
				return
			}
			app := "play_and_get_digits"
			// <min> <max> <tries> <timeout> <terminators> <file> <invalid_file> [<var_name> [<regexp> [<digit_timeout> [<transfer_on_failure>]]]]
			data := `%d %d %d %d %s %s %s %s %s %d`
			data = fmt.Sprintf(data, item.Min, item.Max, item.Tries, item.Timeout, item.Terminators, item.File, item.InvalidFile, item.VarName, item.Regexp, item.DigitTimeout)
			con.Execute(app, UId, data)
			DtmfDigits[UId] = item.Name
		}
	}()
}

// 报工号
type SayJobnum struct {
	PrefixFile string
	SuffixFile string
	Jobnum     []string
}

func PrepareSayJobnum(dialplanNumber string) <-chan interface{} {
	items := make(chan interface{})
	go func() {
		if MapExt[dialplanNumber].IsSayJobnum == true {
			sayJobnum := SayJobnum{
				PrefixFile: "/home/voices/rings/common/job_number_prefix.wav",
				SuffixFile: "/home/voices/rings/common/job_number_suffix.wav",
				Jobnum: []string{
					"/home/voices/rings/common/1.wav",
					"/home/voices/rings/common/0.wav",
					"/home/voices/rings/common/0.wav",
				},
			}
			items <- sayJobnum
		} else {
			items <- "no_need_say_jobnum"
		}
		close(items)
	}()
	return items
}
func ExecuteSayJobnum(con *esl.Connection, UId string, items <-chan interface{}) {
	go func() {
		for item := range items {
			switch t := item.(type) {
			case SayJobnum:
				// con.ExecuteSync("playback", UId, t.PrefixFile)
				// for _, val := range t.Jobnum {
				// 	con.ExecuteSync("playback", UId, val)
				// }
				// con.ExecuteSync("playback", UId, t.SuffixFile)
				// 直转的话，被叫先answer，所以先听到报工号，被叫说话，但主叫报工号还没完成，可以被打断。

				con.Api("uuid_broadcast", UId+" "+t.PrefixFile+" both")
				for _, val := range t.Jobnum {
					con.Api("uuid_broadcast", UId+" "+val+" both")
				}
				con.Api("uuid_broadcast", UId+" "+t.SuffixFile+" both")
			}
		}
	}()
}
