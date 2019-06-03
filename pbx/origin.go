package main

import (
	"fmt"
	"log"

	"github.com/vma/esl"
)

type Handler struct {
	CallId  string
	BgJobId string
}

const (
	Caller = "059163368908"
	Callee = "13675017141"
)

// 102.165.36.185干扰sip的ip
func main() {
	handler := &Handler{}
	con, err := esl.NewConnection("127.0.0.1:8021", handler)
	if err != nil {
		log.Fatal("ERR connecting to freeswitch:", err)
	}
	con.HandleEvents()
}

func (h *Handler) OnConnect(con *esl.Connection) {
	con.SendRecv("event", "plain", "ALL")
	h.CallId, _ = con.Api("create_uuid")
	log.Println("call id:", h.CallId)
	h.BgJobId, _ = con.BgApi("originate", "{origination_uuid="+h.CallId+
		",origination_caller_id_number="+Caller+"}sofia/gateway/zq_tb/"+Callee, "&park()")
	log.Println("originate bg job id:", h.BgJobId)
}

func (h *Handler) OnDisconnect(con *esl.Connection, ev *esl.Event) {
	log.Println("esl disconnected:", ev)
}

func (h *Handler) OnClose(con *esl.Connection) {
	log.Println("esl connection closed")
}

func (h *Handler) OnEvent(con *esl.Connection, ev *esl.Event) {
	log.Printf("%s - event %s %s %s\n", ev.UId, ev.Name, ev.App, ev.AppData)
	fmt.Println(ev) // send to stderr as it is very verbose
	switch ev.Name {
	case esl.BACKGROUND_JOB:
		log.Printf("bg job result:%s\n", ev.GetTextBody())
	case esl.CHANNEL_ANSWER:
		log.Println("call answered, starting moh")
		con.Execute("playback", h.CallId, "local_stream://moh")
	case esl.CHANNEL_HANGUP:
		hupcause := ev.Get("Hangup-Cause")
		log.Printf("call terminated with cause %s", hupcause)
		con.Close()
	}
}
