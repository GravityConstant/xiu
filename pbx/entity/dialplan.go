package entity

type Extension struct {
	Name            string
	Field           string
	Expression      string
	Actions         []Action
	IsRecord        bool
	IsSayJobnum     bool
	IsSatisfySurvey bool
	ResponseType    int // 0(default):顺序接听, 1:随机接听
}

type Action struct {
	Order    int64
	App      string
	Data     string
	Sync     bool
	Executed bool
	Params   map[string]string
}

type ByOrder []Action

func (a ByOrder) Len() int           { return len(a) }
func (a ByOrder) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByOrder) Less(i, j int) bool { return a[i].Order < a[j].Order }

// [dialplanNumber]Extension
var MapExt = make(map[string]*Extension)
