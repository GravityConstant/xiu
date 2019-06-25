package controller

import (
	"fmt"
	"strings"

	"xiu/pbx/models"
)

const (
	pathParam = `/home/voices/rings/common/%s.wav`
)

type Jobnum struct {
	RingPaths []string
}

func (self *Jobnum) GetCallString(dialplanNumber, callee string) {
	// db query
	jobnumStr := models.GetJobnumString(dialplanNumber, callee)
	// split
	if len(jobnumStr) > 0 {
		jobnums := strings.Split(strings.TrimSpace(jobnumStr), "")
		for _, val := range jobnums {
			self.RingPaths = append(self.RingPaths, fmt.Sprintf(pathParam, val))
		}
	}
}

type SatisfySurvey struct {
	IsOpen bool
}

func (self *SatisfySurvey) IsOpenSatisfySurvey(dialplanNumber, callee string) {
	// db query
	self.IsOpen = models.GetOpenSatisfySurvey(dialplanNumber, callee)
}
