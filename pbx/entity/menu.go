package entity

import (
	"errors"
	"sync"
)

var (
	ErrIvrFileNotExist = errors.New("ivr file not exist")
	ErrNoEntry         = errors.New("ivr not entry")
)

type Menu struct {
	Name         string // 无关
	Min          int    // Minimum number of digits to fetch (minimum value of 0)
	Max          int    // Maximum number of digits to fetch (maximum value of 128)
	Tries        int    // number of tries for the sound to play
	Timeout      int    // Number of milliseconds to wait for a dialed response after the file playback ends and before PAGD does a retry.
	Terminators  string //  digits used to end input if less than <max> digits have been pressed. If it starts with '=', then a terminator must be present for the input to be accepted (Since FS 1.2). (Typically '#', can be empty). Add '+' in front of terminating digit to always append it to the result variable specified in var_name.
	File         string // Sound file to play to prompt for digits to be dialed by the caller; playback can be interrupted by the first dialed digit (can be empty or the special string "silence" to omit the message).
	InvalidFile  string // Sound file to play when digits don't match the regexp (can be empty to omit the message).
	VarName      string // Channel variable into which valid digits should be placed (optional, no variable is set by default. See also 'var_name_invalid' below).
	Regexp       string // Regular expression to match digits (optional, an empty string allows all input (default)).
	DigitTimeout int    // Inter-digit timeout; number of milliseconds allowed between digits in lieu of dialing a terminator digit; once this number is reached, PAGD assumes that the caller has no more digits to dial (optional, defaults to the value of <timeout>).
	Entrys       []Entry
	Err          error // 无关
}

type Entry struct {
	Action string
	Digits string
	Param  string
}

type MenuExecApp struct {
	App   string
	Data  string
	Extra []string
}

var MapMenu = make(map[string]*Menu)
var UIdDtmfSyncMap sync.Map
