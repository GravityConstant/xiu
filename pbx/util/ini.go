package util

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/go-ini/ini"
)

type PbxConfig struct {
	cfg *ini.File
}

var PbxConfigInstance *PbxConfig

func InitPbxConfig() {
	dir := filepath.Dir(".")
	file := filepath.Join(dir, "conf", "conf.ini")
	var err error
	PbxConfigInstance, err = NewPbxConfig(file)
	if err != nil {
		log.Fatal(err)
	}
}

func NewPbxConfig(filename string) (*PbxConfig, error) {
	cfg, err := ini.Load(filename)
	pbxcfg := &PbxConfig{cfg: cfg}
	return pbxcfg, err
}

func (self *PbxConfig) Get(key string) string {
	keys := strings.Split(key, "::")
	if len(keys) == 1 {
		return self.cfg.Section("").Key(key).String()
	} else if len(keys) == 2 {
		return self.cfg.Section(keys[0]).Key(keys[1]).String()
	} else {
		return ""
	}
}
