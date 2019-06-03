package controller

import (
	"fmt"
	"strings"
)

type Outcall struct {
	globalParam  string
	privateParam string
}

var OutcallInstance *Outcall

func init() {
	// {ignore_early_media=true,originate_continue_on_timeout=true}[leg_timeout=15]sofia/gateway/  zqzj/13675017141|[leg_timeout=25]sofia/gateway/zqzj/83127866"
	OutcallInstance = &Outcall{
		globalParam:  `{ignore_early_media=true,originate_continue_on_timeout=true,sip_h_Diversion=<sip:%s@ip>}`,
		privateParam: `[leg_timeout=%d]sofia/gateway/%s/%s`,
	}
}

func (self *Outcall) GetCallString(diversion string) string {
	// `{originate_timeout=%d,sip_h_Diversion=<sip:%s@ip>}sofia/gateway/zqzj/13675017141`
	timeout := []int{15, 15}
	gateway := "zqzj"
	bindPhone := []string{"13675017141", "17750409737"}
	pPs := []string{}

	var pP string
	for i := 0; i < len(bindPhone); i++ {
		pP = fmt.Sprintf(self.privateParam, timeout[i], gateway, bindPhone[i])
		pPs = append(pPs, pP)
	}
	pP = strings.Join(pPs, "|")
	gP := fmt.Sprintf(self.globalParam, diversion)
	return gP + pP
}
