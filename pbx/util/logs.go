package util

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/sirupsen/logrus"
)

var logpbx = logrus.New()

func InitLog() {
	runmode := PbxConfigInstance.Get("runmode")
	level, _ := strconv.ParseUint(PbxConfigInstance.Get("logs::level"), 10, 32)
	// level
	switch logrus.Level(level) {
	case logrus.PanicLevel:
		logpbx.Level = logrus.PanicLevel
	case logrus.FatalLevel:
		logpbx.Level = logrus.FatalLevel
	case logrus.ErrorLevel:
		logpbx.Level = logrus.ErrorLevel
	case logrus.WarnLevel:
		logpbx.Level = logrus.WarnLevel
	case logrus.InfoLevel:
		logpbx.Level = logrus.InfoLevel
	case logrus.DebugLevel:
		logpbx.Level = logrus.DebugLevel
	case logrus.TraceLevel:
		logpbx.Level = logrus.TraceLevel
	}
	// runmode
	if runmode == "dev" {
		dirname := filepath.Dir(".")
		logpath := filepath.Join(dirname, "logs")
		filename := filepath.Join(logpath, "pbx.log")

		// runmode
		if err := os.MkdirAll(logpath, 0666); err == nil {
			file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
			if err == nil {
				logpbx.Out = file
			} else {
				logpbx.Info("Failed to logpbx to file, using default stderr")
			}

		} else {
			logpbx.Info("Failed to create file, using default stderr")
		}

	}
}

func Trace(args ...interface{}) {
	logpbx.WithFields(logrus.Fields{
		"filename": args[0],
		"line":     args[1],
	}).Trace(args[2:]...)
}

func Debug(args ...interface{}) {
	logpbx.WithFields(logrus.Fields{
		"filename": args[0],
		"line":     args[1],
	}).Debug(args[2:]...)
}

func Print(args ...interface{}) {
	logpbx.WithFields(logrus.Fields{
		"filename": args[0],
		"line":     args[1],
	}).Print(args[2:]...)
}

func Info(args ...interface{}) {
	logpbx.WithFields(logrus.Fields{
		"filename": args[0],
		"line":     args[1],
	}).Info(args[2:]...)
}

func Warn(args ...interface{}) {
	logpbx.WithFields(logrus.Fields{
		"filename": args[0],
		"line":     args[1],
	}).Warn(args[2:]...)
}

func Warning(args ...interface{}) {
	logpbx.WithFields(logrus.Fields{
		"filename": args[0],
		"line":     args[1],
	}).Warning(args[2:]...)
}

func Error(args ...interface{}) {
	logpbx.WithFields(logrus.Fields{
		"filename": args[0],
		"line":     args[1],
	}).Error(args[2:]...)
}

func Fatal(args ...interface{}) {
	logpbx.WithFields(logrus.Fields{
		"filename": args[0],
		"line":     args[1],
	}).Fatal(args[2:]...)
}

func Panic(args ...interface{}) {
	logpbx.WithFields(logrus.Fields{
		"filename": args[0],
		"line":     args[1],
	}).Panic(args[2:]...)
}

func ExampleLogs() {
	Trace("call_in.go", "35", PbxConfigInstance.Get("appname"))
	Debug("call_in.go", "36", PbxConfigInstance.Get("runmode"))
	Print("call_in.go", "37", PbxConfigInstance.Get("postgres::alias"))
	Info("call_in.go", "38", PbxConfigInstance.Get("postgres::name"))
	Warn("call_in.go", "39", PbxConfigInstance.Get("postgres::pwd"))
	Warning("call_in.go", "40", PbxConfigInstance.Get("postgres::user"))
	Error("call_in.go", "41", PbxConfigInstance.Get("postgres::host"))
	Fatal("call_in.go", "42", PbxConfigInstance.Get("postgres::port"))
	Panic("call_in.go", "43", PbxConfigInstance.Get("postgres::sslmode"))
}

func getTimeFileAbsPath() string {
	dirname := filepath.Dir(".")
	logpath := filepath.Join(dirname, "logs")
	filename := filepath.Join(logpath, "pbx.log", time.Now().Format("2006-01-02-15-04-05"))

	return filename
}

type logFileWriter struct {
	file *os.File
	//write count
	size int64
}

func (p *logFileWriter) Write(data []byte) (n int, err error) {
	if p == nil {
		return 0, errors.New("logFileWriter is nil")
	}
	if p.file == nil {
		return 0, errors.New("file not opened")
	}
	n, e := p.file.Write(data)
	p.size += int64(n)
	//文件最大 64K byte
	if p.size > 1024*1024*11 {
		p.file.Close()
		fmt.Println("log file full")
		p.file, _ = os.OpenFile(getTimeFileAbsPath(), os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_SYNC, 0600)
		p.size = 0
	}
	return n, e
}
