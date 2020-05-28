package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"xiu/pbx/util"
)

type Impl struct {
	DB *gorm.DB
}

var ImplInstance = Impl{}

func (self *Impl) InitDB() {
	host := util.PbxConfigInstance.Get("postgres::host")
	port := util.PbxConfigInstance.Get("postgres::port")
	user := util.PbxConfigInstance.Get("postgres::user")
	password := util.PbxConfigInstance.Get("postgres::password")
	dbname := util.PbxConfigInstance.Get("postgres::dbname")
	sslmode := util.PbxConfigInstance.Get("postgres::sslmode")
	runmode := util.PbxConfigInstance.Get("runmode")

	dsn := `host=%s port=%s user=%s password=%s dbname=%s sslmode=%s`
	dsn = fmt.Sprintf(dsn, host, port, user, password, dbname, sslmode)

	var err error
	self.DB, err = gorm.Open("postgres", dsn)
	if err != nil {
		util.Fatal("models/init.go", "29", "Got error when connect database, the error is", err)
	}
	if runmode == "debug" {
		self.DB.LogMode(true)
	} else {
		self.DB.LogMode(false)
	}

	// self.DB.DB()获取到默认的*sql.DB
	self.DB.DB().SetMaxIdleConns(60)

}

//下面是统一的表名管理
func TableName(name string) string {
	prefix := util.PbxConfigInstance.Get("postgres::prefix")
	return prefix + name
}

func BaseMobileLocationTBName() string {
	return TableName("base_mobile_location")
}

func BlacklistTBName() string {
	return TableName("call_blacklist")
}

func DialplanTBName() string {
	return TableName("call_dialplans")
}

func DialplanDetailTBName() string {
	return TableName("call_dialplan_details")
}

func IvrMenuTBName() string {
	return TableName("call_ivr_menus")
}

func IvrMenuOptionTBName() string {
	return TableName("call_ivr_menu_options")
}

func RingsTBName() string {
	return TableName("call_rings")
}

func CallOperationTBName() string {
	return TableName("call_operation")
}

func BindPhonerTBName() string {
	return TableName("foo_bind_phoner")
}

func BindPhoneTimesetTBName() string {
	return TableName("foo_bindphone_timeset")
}

func SatisfySurveyDetailTBName() string {
	return TableName("foo_satisfy_survey_detail")
}
