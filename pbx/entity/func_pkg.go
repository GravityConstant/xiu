package entity

// 报工号
type SayJobnum struct {
	PrefixFile string
	SuffixFile string
	Jobnum     []string
}

// 满意度调查
type SatisfySurvey struct {
	PrefixFile string
	SuffixFile string
}

// 录音
type Record struct {
	Name       string
	PrefixPath string
	File       string
}
