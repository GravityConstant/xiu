/*
 */

package main

import (
	"fmt"
	"log"

	"github.com/vma/esl"
	"xiu/pbx/dialplan"
)

type Handler struct {
	Caller map[string]int
	Callee map[string]int
}

func init() {
	// 初始化map[destination_number]extension，全局变量
	dialplan.InitExtension()
	// 初始化map[destination_number]menu，全局变量
	dialplan.InitIvrMenu()
}

func main() {
	handler := &Handler{}
	handler.Caller = make(map[string]int)

	con, err := esl.NewConnection("127.0.0.1:8021", handler)
	if err != nil {
		log.Fatal("ERR connecting to freeswitch:", err)
	}
	fmt.Println("////////////////////////////////////////////////////")
	fmt.Println("//                event                           //")
	fmt.Println("////////////////////////////////////////////////////")
	con.HandleEvents()
}

func (h *Handler) OnConnect(con *esl.Connection) {
	con.SendRecv("event", "plain", "ALL")
}

func (h *Handler) OnDisconnect(con *esl.Connection, ev *esl.Event) {
	log.Println("esl disconnected:", ev)
}

func (h *Handler) OnClose(con *esl.Connection) {
	log.Println("esl connection closed")
}

func (h *Handler) OnEvent(con *esl.Connection, ev *esl.Event) {
	log.Printf("%s - event %s %s %s\n", ev.UId, ev.Name, ev.App, ev.AppData)
	// fmt.Println(ev) // send to stderr as it is very verbose
	switch ev.Name {
	case esl.CHANNEL_CREATE:
		destinationNumber := ev.Get("Caller-Destination-Number")
		direction := ev.Get("Call-Direction")

		if _, ok := h.Caller[ev.UId]; !ok && direction == "inbound" {
			log.Printf("channel create:%s // %s\n", destinationNumber, ev.UId)
			items := dialplan.PrepareExtension(destinationNumber)
			// 存在ivr的情况时使用
			hasIvr := make(chan string)
			dialplan.ExecuteExtension(con, ev.UId, items, hasIvr)
			// 不再阻塞，继续执行ivr的情况
			ivrMenuExtension := <-hasIvr
			if ivrMenuExtension != "" {
				ivrMenu := dialplan.PrepareIvrMenu(destinationNumber, ivrMenuExtension, "")
				dialplan.ExecuteMenuEntry(con, ev.UId, ivrMenu)
			}

			h.Caller[ev.UId] = 1
		}
	case esl.CHANNEL_EXECUTE_COMPLETE:
		if ev.App == "play_and_get_digits" {
			// fmt.Println(ev)
			resultDTMF := ev.Get("variable_read_result")
			if resultDTMF == "success" {
				dtmfDigits := ev.Get("variable_foo_dtmf_digits")
				destinationNumber := ev.Get("Caller-Destination-Number")
				// dialplan.MapMenu[dialplan.DtmfDigits[ev.UId]]
				// direction := ev.Get("Call-Direction")
				items := dialplan.PrepareIvrMenu(destinationNumber, dialplan.DtmfDigits[ev.UId], dtmfDigits)
				dialplan.ExecuteMenuEntry(con, ev.UId, items)
			} else if resultDTMF == "timeout" {
				dtmfDigits := ev.Get("variable_foo_dtmf_digits")
				destinationNumber := ev.Get("Caller-Destination-Number")
				switch dtmfDigits {
				case "*": // 最大长度大于1，*后不按#会超时，处理返回上级ivr
					items := dialplan.PrepareIvrMenu(destinationNumber, dialplan.DtmfDigits[ev.UId], dtmfDigits)
					dialplan.ExecuteMenuEntry(con, ev.UId, items)
				default:
					// 默认如果没有严格的正则表达式，不会播放输入错误的提示，输入按键不够的话，只能再次播放本层ivr
					items := dialplan.PrepareIvrMenu(destinationNumber, dialplan.DtmfDigits[ev.UId], "digitTimeout")
					dialplan.ExecuteMenuEntry(con, ev.UId, items)
				}
			} else if resultDTMF == "failure" {
				con.Execute("hangup", ev.UId, "")
			}
		} else if ev.App == "bridge" {
			// 满意度调查
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
			// callUId := ev.Get("Channel-Call-UUID")
			dialplanNumber := ev.Get("Other-Leg-Destination-Number")
			blegJobnum := dialplan.PrepareSayJobnum(dialplanNumber)
			dialplan.ExecuteSayJobnum(con, ev.UId, blegJobnum)
		}
	case esl.DTMF:
	case esl.CHANNEL_BRIDGE:
		// fmt.Println(ev)
	case esl.CHANNEL_DESTROY:
		// fmt.Println(ev)
	case esl.CHANNEL_HANGUP:
	}
}

// ivr：被叫挂机，主叫不挂机，通过hangup_after_bridge=true解决了
// ivr：但是被叫不接的话，就一直不挂机了。
/*
直转拒接的时候回的是no_answer,而不是busy
可归类到下面那个，设置了ignore_early_media就不能发现运行商回的什么了~~~
*/
/*
[x] 解决不了啊
直转拒接的时候，leg_timeout设置的比较长，被叫人为提前挂，主叫不挂
monitor_early_media_fail=user_busy:2:450 并没有什么用
*/
