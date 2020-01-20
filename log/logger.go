package log

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	PanicLevel int = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
)

const (
	color_red     = uint8(iota + 91)
	color_green   //	绿
	color_yellow  //	黄
	color_blue    // 	蓝
	color_magenta //	洋红
)

const (
	fatalPrefix = "[FATAL] "
	errorPrefix = "[ERROR] "
	warnPrefix  = "[WARN] "
	infoPrefix  = "[INFO] "
	debugPrefix = "[DEBUG] "
)

const (
	ByDay int = iota
	ByWeek
	ByMonth
	BySize
)

type LogFile struct {
	level    int    // 日志等级
	saveMode int    // 保存模式
	saveDays int    // 日志保存天数
	logTime  int64  //
	fileName string // 日志文件名
	filesize int64  // 文件大小, 需要设置 saveMode 为 BySize 生效
	fileFd   *os.File
}

var lf *LogFile = nil

var logFile LogFile

func DLogger() *LogFile {
	if lf == nil {
		var logF = "./logFile/webCrawler.log"

		lf = &LogFile{
			level:    DebugLevel,
			saveMode: ByDay,
			saveDays: 2,
			fileName: logF,
			filesize: 1024 * 1024,
			logTime:  0,
		}

		// 日志初始化
		log.SetOutput(lf)
		//log.SetFlags(log.Lmicroseconds | log.Lshortfile)
		log.SetFlags(log.Llongfile | log.Ldate | log.Ltime)
	}
	return lf
}

func (logF *LogFile) Debugf(format string, args ...interface{}) {
	if logF.level >= DebugLevel {
		log.SetPrefix(blue(debugPrefix))
		log.Output(2, fmt.Sprintf(format, args...))
	}
}

func (logF *LogFile) Info(args ...interface{}) {
	if logF.level >= InfoLevel {
		log.SetPrefix(green(infoPrefix))
		log.Output(2, fmt.Sprintln(args...))
	}
}

func (logF *LogFile) Infof(format string, args ...interface{}) {
	if logF.level >= InfoLevel {
		log.SetPrefix(green(infoPrefix))
		log.Output(2, fmt.Sprintf(format, args...))
	}
}

func (logF *LogFile) Warnln(args ...interface{}) {
	if logF.level >= WarnLevel {
		log.SetPrefix(magenta(warnPrefix))
		log.Output(2, fmt.Sprintln(args...))
	}
}

func (logF *LogFile) Warnf(format string, args ...interface{}) {
	if logF.level >= WarnLevel {
		log.SetPrefix(magenta(warnPrefix))
		log.Output(2, fmt.Sprintf(format, args...))
	}
}

func (logF *LogFile) Error(args ...interface{}) {
	if logF.level >= ErrorLevel {
		log.SetPrefix(red(errorPrefix))
		log.Output(2, fmt.Sprintln(args...))
	}
}

func (logF *LogFile) Errorf(format string, args ...interface{}) {
	if logF.level >= ErrorLevel {
		log.SetPrefix(red(errorPrefix))
		log.Output(2, fmt.Sprintf(format, args...))
	}
}

func (logF *LogFile) Fatal(args ...interface{}) {
	if logF.level >= FatalLevel {
		log.SetPrefix(red(fatalPrefix))
		log.Output(2, fmt.Sprintln(args...))
	}
}

func (logF *LogFile) Fatalf(format string, args ...interface{}) {
	if logF.level >= FatalLevel {
		log.SetPrefix(red(fatalPrefix))
		log.Output(2, fmt.Sprintf(format, args...))
	}
}

func GetRedPrefix(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_red, s)
}

func red(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_red, s)
}

func green(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_green, s)
}

func yellow(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_yellow, s)
}

func blue(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_blue, s)
}

func magenta(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_magenta, s)
}

func (me LogFile) Write(buf []byte) (n int, err error) {
	fmt.Print(string(buf))
	if me.fileName == "" {
		return len(buf), nil
	}

	switch me.saveMode {
	case BySize:
		fileInfo, err := os.Stat(me.fileName)
		if err != nil {
			me.createLogFile()
			me.logTime = time.Now().Unix()
		} else {
			filesize := fileInfo.Size()
			if me.fileFd == nil ||
				filesize > me.filesize {
				me.createLogFile()
				me.logTime = time.Now().Unix()
			}
		}
	default: // 默认按天  ByDay
		if me.logTime+3600 < time.Now().Unix() {
			me.createLogFile()
			me.logTime = time.Now().Unix()
		}
	}

	if me.fileFd == nil {
		fmt.Printf("日志文件为空 !\n")
		return len(buf), nil
	}

	return me.fileFd.Write(buf)
}

func (me *LogFile) createLogFile() {
	if me.logTime != 0 {
		now := time.Now()
		filename := fmt.Sprintf("%s_%04d%02d%02d_%02d%02d",
			me.fileName, now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute())
		err := os.Rename(me.fileName, filename)
		if err != nil {
			fmt.Println("存档日志文件失败 : ", err.Error())
		}
	}

	_, err := os.Stat(me.fileName)
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(me.fileName), 0700)
	}

	if fd, err := os.OpenFile(me.fileName, os.O_CREATE|os.O_APPEND, 0755); nil == err {
		me.fileFd = fd
	} else {
		fmt.Println("打开日志文件失败: ", err.Error())
	}
}
