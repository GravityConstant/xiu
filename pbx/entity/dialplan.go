package entity

type Extension struct {
	Name            string
	Field           string
	Expression      string
	Actions         []Action
	IsRecord        bool
	IsSayJobnum     bool
	IsSatisfySurvey bool
}

type Action struct {
	Order    int64
	App      string
	Data     string
	Sync     bool
	Executed bool
}

type ByOrder []Action

func (a ByOrder) Len() int           { return len(a) }
func (a ByOrder) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByOrder) Less(i, j int) bool { return a[i].Order < a[j].Order }

// [dialplanNumber]Extension
var MapExt = make(map[string]*Extension)
