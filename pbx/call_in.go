/*
freeswitch命令：
https://freeswitch.org/confluence/display/FREESWITCH/mod_commands
http://wiki.freeswitch.org.cn/wiki/Mod_Commands.html
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"

	"xiu/esl"
	// "xiu/pbx/controller"
	"xiu/pbx/dialplan"
	"xiu/pbx/entity"
	"xiu/pbx/models"
	"xiu/pbx/util"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

type Handler struct {
	Caller map[string]int
	Callee map[string]string
}

func init() {
	// log打印文件名和行号，说打印行号有损性能
	// log.SetFlags(log.Lshortfile | log.LstdFlags)
	// 初始化配置文件读取
	util.InitPbxConfig()
	// 初始化日志
	util.InitLog()
	// 初始化缓存
	util.InitCache()
	// 初始化数据库
	models.ImplInstance.InitDB()
	// 初始化map[destination_number]extension，全局变量
	dialplan.InitExtension()
	// 初始化map[destination_number]menu，全局变量
	dialplan.InitIvrMenu()
	// 测试是否写入到redis
	// controller.PrintCache()
}

func main() {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	// CPU性能监测
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)

	}

	// business code
	go func() {
		fmt.Println("////////////////////////////////////////////////////")
		fmt.Println("//                event                           //")
		fmt.Println("////////////////////////////////////////////////////")
		handler := &Handler{}
		handler.Caller = make(map[string]int)

		con, err := esl.NewConnection("127.0.0.1:8021", handler)
		if err != nil {
			log.Fatal("ERR connecting to freeswitch:", err)
		}
		if err := con.HandleEvents(); err != nil {
			util.Error("call_in.go", "handle events error", err)
		}
	}()

	// 优雅的退出CTRL+C
	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	// signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	// Block until we receive our signal.
	<-c
	// Create a deadline to wait for.
	_, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// stop profile
	fmt.Println("////////////////////////////////////////////////////")
	fmt.Println("//                end                             //")
	fmt.Println("////////////////////////////////////////////////////")
	pprof.StopCPUProfile()

	log.Println("shutting down")
	os.Exit(0)
}

func (h *Handler) OnConnect(con *esl.Connection) {
	con.SendRecv("event", "plain", "ALL")
}

func (h *Handler) OnDisconnect(con *esl.Connection, ev *esl.Event) {
	log.Println("esl disconnected:", ev)
}

func (h *Handler) OnClose(con *esl.Connection, ev *esl.Event) {
	log.Println("esl connection closed")
}

func (h *Handler) OnEvent(con *esl.Connection, ev *esl.Event) {
	log.Printf("%s - event %s %s %s\n", ev.UId, ev.Name, ev.App, ev.AppData)
	// fmt.Println(ev) // send to stderr as it is very verbose
	switch ev.Name {
	case esl.CHANNEL_CREATE:
		destinationNumber := ev.Get("Caller-Destination-Number")
		// 主叫号码
		callerNumber := ev.Get("Caller-Caller-ID-Number")
		direction := ev.Get("Call-Direction")

		if _, ok := h.Caller[ev.UId]; !ok && direction == "inbound" {
			log.Printf("channel create:%s // %s\n", destinationNumber, ev.UId)
			items := dialplan.PrepareExtension(destinationNumber)
			// bridge, ivr特殊处理
			extra := dialplan.ExecuteExtension(con, ev.UId, items)
			// 特殊处理准备
			for ext := range extra {
				switch t := ext.(type) {
				case entity.Action:
					switch t.App {
					case "bridge":
						items := dialplan.PrepareBridge(destinationNumber, callerNumber, t.Data)
						dialplan.ExecuteBridge(con, ev.UId, items)
					case "ivr":
						items := dialplan.PrepareIvrMenu(destinationNumber, callerNumber, t.Data, "")
						dialplan.ExecuteMenuEntry(con, ev.UId, items)
					}
				}
			}

			h.Caller[ev.UId] = 1
		}
	case esl.CHANNEL_EXECUTE_COMPLETE:
		// 主叫号码
		callerNumber := ev.Get("Caller-Caller-ID-Number")
		// 被叫号码
		calleeNumber := ev.Get("Caller-Callee-ID-Number")
		// ivr menu
		if ev.App == "play_and_get_digits" {
			// fmt.Println(ev)
			resultDTMF := ev.Get("variable_read_result")
			var ivrMenuExtension string
			if curMenu, ok := entity.UIdDtmfSyncMap.Load(ev.UId); ok {
				ivrMenuExtension = curMenu.(string)
			} else {
				util.Error("call_in.go", "ivr menu extension not found", ev.UId)
				con.Execute("hangup", ev.UId, "")
				return
			}
			if resultDTMF == "success" {
				dtmfDigits := ev.Get("variable_foo_dtmf_digits")
				destinationNumber := ev.Get("Caller-Destination-Number")
				// dialplan.MapMenu[entity.DtmfDigits[ev.UId]]
				// direction := ev.Get("Call-Direction")
				items := dialplan.PrepareIvrMenu(destinationNumber, callerNumber, ivrMenuExtension, dtmfDigits)
				dialplan.ExecuteMenuEntry(con, ev.UId, items)
			} else if resultDTMF == "timeout" {
				dtmfDigits := ev.Get("variable_foo_dtmf_digits")
				destinationNumber := ev.Get("Caller-Destination-Number")
				switch dtmfDigits {
				case "*": // 最大长度大于1，*后不按#会超时，处理返回上级ivr
					items := dialplan.PrepareIvrMenu(destinationNumber, callerNumber, ivrMenuExtension, dtmfDigits)
					dialplan.ExecuteMenuEntry(con, ev.UId, items)
				default:
					// 默认如果没有严格的正则表达式，不会播放输入错误的提示，输入按键不够的话，只能再次播放本层ivr
					items := dialplan.PrepareIvrMenu(destinationNumber, callerNumber, ivrMenuExtension, "digitTimeout")
					dialplan.ExecuteMenuEntry(con, ev.UId, items)
				}
			} else if resultDTMF == "failure" || resultDTMF == "" {
				// 彩铃不存在时，resultDTMF为空
				con.Execute("hangup", ev.UId, "")
			}
		} else if ev.App == "bridge" {
			// 挂机还是进行满意度调查
			// fmt.Println(ev)
			dialStatus := ev.Get("variable_DIALSTATUS")
			// 不为空，标识主叫先挂机，所以就不能进行满意度调查了
			hangupDisposition := ev.Get("variable_sip_hangup_disposition")
			if hangupDisposition == "" {
				switch dialStatus {
				case "SUCCESS":
					destinationNumber := ev.Get("Caller-Destination-Number")
					items := dialplan.PrepareSatisfySurvey(destinationNumber, calleeNumber)
					dialplan.ExecuteSatisfySurvey(con, ev.UId, items)
				case "ALLOTTED_TIMEOUT":
					// freeswitch自己挂断了。没有UUID，再次执行发生错误
				case "NO_USER_RESPONSE":
					if _, err := con.Execute("hangup", ev.UId, ""); err != nil {
						util.Error("call_in.go", "160", err)
					}
				}
			}

		} else if ev.App == "read" {
			// fmt.Println(ev)
			// 不为空，主叫不按满意度调查键直接挂机
			hangupDisposition := ev.Get("variable_sip_hangup_disposition")
			if hangupDisposition == "" {
				resultRead := ev.Get("variable_read_result")
				var satisfySurveyDigits string
				if resultRead == "success" {
					satisfySurveyDigits = ev.Get("variable_foo_satisfy_survey_digits")
				} else {
					satisfySurveyDigits = "unknown"
				}

				dialplan.HandleSatisfySurvey(con, ev.UId, satisfySurveyDigits)
			}

		}
	case esl.BACKGROUND_JOB:
	case esl.CHANNEL_ANSWER:
		// fmt.Println(ev)
		// 直转：先bleg answer，再aleg answer，然后进行bridge
		// ivr：先aleg answer，再bridge, 然后bleg answer
		// 通过指定ignore_early_media=ring_ready,解决了直转bleg先报工号，aleg再报的问题
		// 因为指定该参数，bleg接通，马上会通知aleg
		direction := ev.Get("Call-Direction")
		if direction == "outbound" {
			callUId := ev.Get("Channel-Call-UUID")
			dialplanNumber := ev.Get("Other-Leg-Destination-Number")
			// 录音
			caller := ev.Get("Caller-Caller-ID-Number")
			callee := ev.Get("Caller-Callee-ID-Number")
			record := dialplan.PrepareRecord(dialplanNumber, caller, callee)
			dialplan.ExecuteRecord(con, callUId, record)
			// 报工号
			blegJobnum := dialplan.PrepareSayJobnum(dialplanNumber, callee)
			dialplan.ExecuteSayJobnum(con, callUId, blegJobnum)
		}
	case esl.DTMF:
	case esl.CHANNEL_BRIDGE:
		// fmt.Println(ev)
	case esl.CHANNEL_DESTROY:
		// fmt.Println(ev)
	case esl.CHANNEL_HANGUP:
		entity.UIdDtmfSyncMap.Delete(ev.UId)
	}
}

// ivr：被叫挂机，主叫不挂机，通过hangup_after_bridge=true解决了
/*
ivr：但是被叫不接的话，就一直不挂机了。
解决：通过CHANNEL_EXECUTE_COMPLETE的bridge状态判断
*/
/*
直转拒接的时候回的是no_answer,而不是busy
可归类到下面那个，设置了ignore_early_media就不能发现运行商回的什么了~~~
*/
/*
[x] 解决不了啊
直转拒接的时候，leg_timeout设置的比较长，被叫人为提前挂，主叫不挂
monitor_early_media_fail=user_busy:2:450 并没有什么用
*/
