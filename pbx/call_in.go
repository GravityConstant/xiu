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
	"sync/atomic"
	"syscall"
	"time"

	"xiu/esl"
	// "xiu/pbx/controller"
	"xiu/pbx/dialplan"
	"xiu/pbx/entity"
	"xiu/pbx/models"
	"xiu/pbx/util"
	colorlog "xiu/util"
)

var done atomic.Value

type Handler struct {
	Caller map[string]int
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
	// dialplan.InitExtension()
	// 初始化map[destination_number]menu，全局变量
	// dialplan.InitIvrMenu()
	// 测试是否写入到redis
	// controller.PrintCache()
}

func main() {
	// set params
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*5, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	// parse params
	flag.Parse()
	// CPU性能监测
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		// stop profile
		pprof.StopCPUProfile()

	}
	// begin
	handler := new(Handler)
	handler.Caller = make(map[string]int)
	// Create a deadline to wait for.
	ctx, cancel := context.WithCancel(context.Background())
	// business code
	go func() {
	RESTART:
		fmt.Println("////////////////////////////////////////////////////")
		fmt.Println("//                event                           //")
		fmt.Println("////////////////////////////////////////////////////")

		con, err := esl.NewConnection("127.0.0.1:8021", handler)
		if err != nil {
			log.Fatal("ERR connecting to freeswitch:", err)
		}
		if err := con.HandleEvents(ctx); err != nil {
			util.Error("call_in.go", "handle events error", err)
			time.Sleep(time.Second)
			goto RESTART
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
	// on event no longer handle new UId
	done.Store(true)
	// statistics current calls
	log.Printf("quiting and remainning calls: %d", len(handler.Caller))
	// wait time and cancel
	// <-time.After(wait)
	// use time.Tick to wait handle remaing calls
	tickCount := 0
	tick := time.Tick(wait)
	for range tick {
		if len(handler.Caller) == 0 {
			break
		} else if tickCount == 36 {
			break
		}
		log.Printf("tickCount: %d, remainning calls: %d", tickCount, len(handler.Caller))
		tickCount++
	}
	// stop business work
	cancel()

	// exit
	log.Println("shutting down")
	os.Exit(0)
}

func (h *Handler) OnConnect(con *esl.Connection) {
	// 取消事件：nixevent
	// con.SendRecv("event", "plain", "ALL")
	con.SendRecv("event", "plain", "CHANNEL_CREATE CHANNEL_EXECUTE_COMPLETE CHANNEL_ANSWER CHANNEL_HANGUP")
}

func (h *Handler) OnDisconnect(con *esl.Connection, ev *esl.Event) {
	log.Println("esl disconnected:", ev)
}

func (h *Handler) OnClose(con *esl.Connection) {
	log.Println("esl connection closed")
}

func (h *Handler) OnEvent(con *esl.Connection, ev *esl.Event) {
	// 终止程序执行，不再处理新的呼入了。
	if t, ok := done.Load().(bool); ok && t == true {
		if _, ok := h.Caller[ev.UId]; !ok {
			// log.Printf("%s - reject event %s %s %s\n", ev.UId, ev.Name, ev.App, ev.AppData)
			return
		}
	}
	// log.Printf("%s - event %s %s %s\n", ev.UId, ev.Name, ev.App, ev.AppData)
	// fmt.Println(ev) // send to stderr as it is very verbose
	// 直接挂机了，不做任何处理
	// 看了CHANNEL_HANGUP没有这个变量，所以使用了。
	// 如果有的话，就不能清除entity.UIdDtmfSyncMap这个map的某些值
	if sipHangupDisposition := ev.Get("variable_sip_hangup_disposition"); len(sipHangupDisposition) > 0 {
		delete(h.Caller, ev.UId)
		entity.UIdDtmfSyncMap.Delete(ev.UId)
		// util.Info("call_in.go", "sip hangup disposition", ev.Name, ev.App, ev.AppData)
		return
	}
	// 事件处理
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
						// 这里进行修改。。。
						items := dialplan.PrepareBridge(destinationNumber, callerNumber, t.Data, t.Params)
						dialplan.ExecuteBridge(con, ev.UId, items)
						// viewDialplan(items)
						h.Caller[ev.UId] = 1
					case "ivr":
						items := dialplan.PrepareIvrMenu(destinationNumber, callerNumber, t.Data, "")
						dialplan.ExecuteMenuEntry(con, ev.UId, items)
						// viewDialplan(items)
						h.Caller[ev.UId] = 1
					case "hangup":
						util.Warning("call_in.go", "extension hangup", "NO_ROUTE_DESTINATION")
						// con.Execute("hangup", ev.UId, t.Data)
						return
					}
				}
			}
		}
	case esl.CHANNEL_EXECUTE_COMPLETE:

		log.Printf("%s - event %s %s %s\n", ev.UId, ev.Name, ev.App, ev.AppData)
		// 主叫号码
		callerNumber := ev.Get("Caller-Caller-ID-Number")
		// 被叫号码
		calleeNumber := ev.Get("Caller-Callee-ID-Number")
		des := ev.Get("Caller-Destination-Number")
		// 可能是belg进来，要查找另外一个脚
		otherLeg := ev.Get("Other-Leg-Unique-ID")
		// 只有在h.Caller的map中的呼叫才被处理
		if _, ok := h.Caller[ev.UId]; !ok {
			if _, ok := h.Caller[otherLeg]; !ok {
				util.Warning("call_in.go", "CHANNEL_EXECUTE_COMPLETE not need handle", map[string]string{"caller:": callerNumber, "callee:": calleeNumber, "des:": des})
				return
			}
		}
		// ivr menu
		if ev.App == "play_and_get_digits" {
			// fmt.Println(ev)
			resultDTMF := ev.Get("variable_read_result")
			var ivrMenuExtension string
			if curMenu, ok := entity.UIdDtmfSyncMap.Load(ev.UId); ok {
				ivrMenuExtension = curMenu.(string)
			} else {
				util.Error("call_in.go", "ivr menu extension not found", ev.UId)
				// 这里不能在挂机了，会进入这里，一般是主叫主动挂机了！！！
				// con.Execute("hangup", ev.UId, "")
				return
			}
			if resultDTMF == "success" {
				dtmfDigits := ev.Get("variable_foo_dtmf_digits")
				destinationNumber := ev.Get("Caller-Destination-Number")
				// dialplan.MapMenu[entity.DtmfDigits[ev.UId]]
				// direction := ev.Get("Call-Direction")
				items := dialplan.PrepareIvrMenu(destinationNumber, callerNumber, ivrMenuExtension, dtmfDigits)
				dialplan.ExecuteMenuEntry(con, ev.UId, items)
				// viewDialplan(items)
			} else if resultDTMF == "timeout" {
				dtmfDigits := ev.Get("variable_foo_dtmf_digits")
				destinationNumber := ev.Get("Caller-Destination-Number")
				switch dtmfDigits {
				case "*": // 最大长度大于1，*后不按#会超时，处理返回上级ivr
					items := dialplan.PrepareIvrMenu(destinationNumber, callerNumber, ivrMenuExtension, dtmfDigits)
					dialplan.ExecuteMenuEntry(con, ev.UId, items)
					// viewDialplan(items)
				default:
					// 默认如果没有严格的正则表达式，不会播放输入错误的提示，输入按键不够的话，只能再次播放本层ivr
					items := dialplan.PrepareIvrMenu(destinationNumber, callerNumber, ivrMenuExtension, "digitTimeout")
					dialplan.ExecuteMenuEntry(con, ev.UId, items)
					// viewDialplan(items)
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
					// viewDialplan(items)
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
				// fmt.Println("satisfySurveyDigits: ", satisfySurveyDigits)
			}

		}
	case esl.BACKGROUND_JOB:
	case esl.CHANNEL_ANSWER:
		// 可能是belg进来，要查找另外一个脚
		otherLeg := ev.Get("Other-Leg-Unique-ID")
		// 只有在h.Caller的map中的呼叫才被处理
		if _, ok := h.Caller[ev.UId]; !ok {
			if _, ok := h.Caller[otherLeg]; !ok {
				util.Warning("call_in.go", "CHANNEL_ANSWER not need handle", map[string]string{"des:": ev.Get("Other-Leg-Destination-Number")})
				return
			}
		}
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
			// viewDialplan(record)
			// 报工号
			blegJobnum := dialplan.PrepareSayJobnum(dialplanNumber, callee)
			dialplan.ExecuteSayJobnum(con, callUId, blegJobnum)
			// fmt.Println("callUId:", callUId, "blegJobnum", blegJobnum)
			// viewDialplan(record)
		}
	case esl.DTMF:
	case esl.CHANNEL_BRIDGE:
		// fmt.Println(ev)
	case esl.CHANNEL_DESTROY:
		// fmt.Println(ev)
	case esl.CHANNEL_HANGUP:
		// 可能是belg进来，要查找另外一个脚
		otherLeg := ev.Get("Other-Leg-Unique-ID")
		// 只有在h.Caller的map中的呼叫才被处理
		if _, ok := h.Caller[ev.UId]; !ok {
			if _, ok := h.Caller[otherLeg]; !ok {
				util.Warning("call_in.go", "CHANNEL_HANGUP not need handle", map[string]string{"des:": ev.Get("Caller-Destination-Number")})
				return
			}
		}
		delete(h.Caller, ev.UId)
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

// 先不处理呼入，打印查看数据
func viewDialplan(items <-chan interface{}) {
	for item := range items {
		switch t := item.(type) {
		case entity.Extension:
			colorlog.Info("extension: %v\n", t)
		case entity.Action:
			colorlog.Info("action: %v\n", t)
		case entity.Menu:
			colorlog.Info("menu: %v\n", t)
		case entity.Entry:
			colorlog.Info("entry: %v\n", t)
		case entity.MenuExecApp:
			colorlog.Info("menuexecapp: %v\n", t)
		default:
			colorlog.Warning("undefined struct: %v\n", t)
		}
	}
}
