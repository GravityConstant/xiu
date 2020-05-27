package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "math/rand"
    "net/http"
    "strings"
    "sync/atomic"
    "time"

    "github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm/dialects/postgres"
)

// http://mobsec-dianhua.baidu.com/dianhua_api/open/location?tel=1568401,1463484
const (
    // request_url = `http://mobsec-dianhua.baidu.com/dianhua_api/open/location`
    
    // {"ret":"ok","mobile":"18050132566","data":["福建","福州","电信","0591",""]}
    request_url = `http://api.ip138.com/mobile/?mobile=18050132566&token=e7ad6e24e2b70d0e600be28d5e815874`
)

var (
    telTmpl   = `?tel=%s`
    provinces = []string{"北京", "天津", "河北", "山西", "内蒙古", "辽宁", "吉林", "黑龙江", "上海", "江苏",
        "浙江", "安徽", "福建", "江西", "山东", "河南", "湖北", "湖南", "广东", "广西",
        "海南", "重庆", "四川", "贵州", "云南", "西藏", "陕西", "甘肃", "青海", "宁夏",
        "新疆"}
)

type Impl struct {
    DB *gorm.DB
}

var ImplInstance = Impl{}

func (self *Impl) InitDB() {
    host := "127.0.0.1"
    port := "******"
    user := "******"
    password := "******"
    dbname := "******"
    sslmode := "disable"
    runmode := "debug"

    dsn := `host=%s port=%s user=%s password=%s dbname=%s sslmode=%s`
    dsn = fmt.Sprintf(dsn, host, port, user, password, dbname, sslmode)

    var err error
    self.DB, err = gorm.Open("postgres", dsn)
    if err != nil {
        log.Fatal("Got error when connect database, the error is ", err)
    }
    if runmode == "debug" {
        self.DB.LogMode(true)
    } else {
        self.DB.LogMode(false)
    }

    // self.DB.DB()获取到默认的*sql.DB
    self.DB.DB().SetMaxIdleConns(10)

}

//下面是统一的表名管理
func TableName(name string) string {
    prefix := ""
    return prefix + name
}

type Mobile struct {
    Phone string
}

// get data from db
func GetData() []*Mobile {
    sql := `select cdr.no as phone from (
        select DISTINCT substr(caller_id_number, 0, 8) as no from %s 
        where direction='inbound' 
        and char_length(caller_id_number)=11 and substr(caller_id_number, 0, 2) = '1'
    ) as cdr left JOIN base_mobile_location loc on cdr.no=loc.no where loc."location" is null order by cdr.no`
    sql = fmt.Sprintf(sql, CallPgCdrTBName())

    data := make([]*Mobile, 0)
    if err := ImplInstance.DB.Raw(sql).Scan(&data).Error; err != nil {
        log.Fatal("query cdr data error: ", err)
    }

    return data
}

type BaseMobileLocation struct {
    No         string
    Location   string
    DistrictNo string
}

func GetAreaCode(locations []string) []*BaseMobileLocation {
    // prepare
    tmpl := `'%s'`
    var locationsFormat []string
    for _, val := range locations {
        locationsFormat = append(locationsFormat, fmt.Sprintf(tmpl, val))
    }
    // query
    sql := `select DISTINCT district_no, location from %s where location in (%s) order by district_no`
    sql = fmt.Sprintf(sql, BaseMobileLocationTBName(), strings.Join(locationsFormat, ","))

    data := make([]*BaseMobileLocation, 0)
    if err := ImplInstance.DB.Raw(sql).Scan(&data).Error; err != nil {
        log.Fatal("query base mobile location data error: ", err)
    }

    return data
}

func CallPgCdrTBName() string {
    return TableName("call_pg_cdr")
}

func BaseMobileLocationTBName() string {
    return TableName("base_mobile_location")
}

// parse baidu json
type Result struct {
    Rsp       map[string]LabelPhone `json:"response"`
    RspHeader ResponseHeader        `json:"responseHeader"`
}

type LabelPhone struct {
    Detail   Detail
    Location string
}

type Detail struct {
    Area     []City
    Province string
    Type     string
    Operator string
}

type City struct {
    City string
}

type ResponseHeader struct {
    Status  int
    Time    int64
    Version string
}

func getRand() int {
    rand.Seed(time.Now().UnixNano())
    return 1000 + int(float64(rand.Intn(10))*100)
}

var count uint64

/*
 * {"response":{
 *     "1475051":{"detail":{"area":[{"city":"肇庆"}],"province":"广东","type":"domestic","operator":"移动"},"location":"广东肇庆移动"},
 *     "1787604":{"detail":{"area":[{"city":"湛江"}],"province":"广东","type":"domestic","operator":"移动"},"location":"广东湛江移动"},
 *     "1568428":{"detail":{"area":[{"city":"潍坊"}],"province":"山东","type":"domestic","operator":"联通"},"location":"山东潍坊联通"}
 * },"responseHeader":{
 *     "status":200,
 *     "time":1575938929021,
 *     "version":"1.1.0"
 * }}
 */
func GetLocations(url string, rsp chan map[string]LabelPhone) {

    go func() {
        // rest time and request again
        time.Sleep(time.Duration(getRand()) * time.Millisecond)
        newUInt := atomic.AddUint64(&count, 1)
        atomic.SwapUint64(&count, newUInt)
        resp, err := http.Get(url)
        if err != nil {
            log.Fatal(err)
        }
        defer resp.Body.Close()
        body, err := ioutil.ReadAll(resp.Body)

        // parse json
        var result Result
        err = json.Unmarshal(body, &result)
        if err != nil {
            fmt.Println("error:", err)
        }
        rsp <- result.Rsp
    }()

}

func main() {
    // init db
    ImplInstance.InitDB()
    defer ImplInstance.DB.Close()
    // test data
    // data := []string{
    //  "1461881",
    //  "1463484",
    //  "1470009",
    //  "1474369",
    //  "1475051",
    // }
    data := GetData()
    if len(data) == 0 {
        log.Println("no data need updated")
    }
    // query area
    var urls []string
    var tels []string
    for index, item := range data {
        if index > 0 && index%10 == 0 {
            urls = append(urls, request_url+fmt.Sprintf(telTmpl, strings.Join(tels, ",")))
            tels = []string{item.Phone}
        } else {
            tels = append(tels, item.Phone)
        }
    }
    // add ending
    if len(tels) > 0 {
        urls = append(urls, request_url+fmt.Sprintf(telTmpl, strings.Join(tels, ",")))
    }
    for _, url := range urls {
        fmt.Println(url)
    }
    rsp := make(chan map[string]LabelPhone, len(urls))

    for _, url := range urls {
        GetLocations(url, rsp)
    }

    // get response from baidu api
    for response := range rsp {
        // get area code
        var locations []string
        for _, item := range response {
            if len(item.Location) > 0 {
                tmp := formatLocation1(&item)

                if !strInArray(tmp, locations) {
                    locations = append(locations, tmp)
                }
            }
        }
        fmt.Println(locations)
        if len(locations) == 0 {
            log.Println("no locations!!!")
            return
        }
        locationData := GetAreaCode(locations)
        // prepare insert data structure
        multiInsertM := make([]string, 0)
        for key, item := range response {
            if len(item.Location) > 0 {
                tmp := formatLocation1(&item)
                for _, loc := range locationData {
                    if strings.Compare(tmp, loc.Location) == 0 {
                        multiInsertM = append(multiInsertM, fmt.Sprintf("('%s', '%s', '%s')", key, tmp, loc.DistrictNo))
                    }
                }
            }
        }
        // prepare sql
        if len(multiInsertM) == 0 {
            log.Println("sql no values!")
            return
        }
        sql := "insert into %s values " + strings.Join(multiInsertM, ",")
        sql = fmt.Sprintf(sql, BaseMobileLocationTBName())
        if err := ImplInstance.DB.Exec(sql).Error; err != nil {
            log.Fatal("insert base mobile location err. ", err)
        } else {
            log.Println("update base mobile location success!")
            fmt.Println(atomic.LoadUint64(&count))
        }
    }
    close(rsp)
}

func strInArray(s string, ss []string) bool {
    for _, val := range ss {
        if strings.Compare(s, val) == 0 {
            return true
            break
        }
    }
    return false
}

func formatLocation(s string) string {
    var tmp string
    // 直辖市没有省份
    if strings.Compare(s[:6], s[6:12]) == 0 {
        tmp = s[:6]
    } else {
        tmp = fmt.Sprintf("%s %s", s[:6], s[6:12])
    }

    return tmp
}

func formatLocation1(item *LabelPhone) string {
    if strings.Compare(item.Detail.Province, item.Detail.Area[0].City) == 0 {
        return item.Detail.Province
    }
    return fmt.Sprintf("%s %s", item.Detail.Province, item.Detail.Area[0].City)
}
