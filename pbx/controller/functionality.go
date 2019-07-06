package controller

import (
	"fmt"
	"strings"

	"xiu/pbx/models"
	"xiu/pbx/util"
)

const (
	pathParam = `/home/voices/rings/common/%s.wav`
)

type Jobnum struct {
	RingPaths []string
}

func (self *Jobnum) GetJobnumRingPath(dialplanNumber, callee string) {
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

// 结果保存
func (self *SatisfySurvey) SaveSatisfySurveyResult(uid, key string) {
	// db insert
	sql := `insert into %s values ($1, $2)`
	sql = fmt.Sprintf(sql, models.SatisfySurveyDetailTBName())

	if err := models.ImplInstance.DB.Exec(sql, uid, key).Error; err != nil {
		util.Error("controller/functionality.go", "insert satisfy survey detail", err)
	}
}
