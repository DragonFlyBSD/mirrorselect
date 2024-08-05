package common

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
)

var (
	outLogger *log.Logger
	errLogger *log.Logger
)

func init() {
	flag := log.Ldate | log.Ltime
	outLogger = log.New(os.Stdout, "", flag)
	errLogger = log.New(os.Stderr, "", flag)
}

// Get the file and function information of the logger caller.
// Result: "file:line:function"
func getOrigin() string {
	// calldepth is 2: caller -> xxxPrintf() -> getOrigin()
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return "???:?:???"
	}

	funcname := runtime.FuncForPC(pc).Name()
	fn := funcname[strings.LastIndex(funcname, ".")+1:]
	return file + ":" + strconv.Itoa(line) + ":" + fn
}

func DebugPrintf(format string, v ...interface{}) {
	if !AppConfig.Debug {
		return
	}

	format = fmt.Sprintf("[DEBUG] %s: %s", getOrigin(), format)
	errLogger.Printf(format, v...)
}

func InfoPrintf(format string, v ...interface{}) {
	format = fmt.Sprintf("[INFO] %s: %s", getOrigin(), format)
	outLogger.Printf(format, v...)
}

func WarnPrintf(format string, v ...interface{}) {
	format = fmt.Sprintf("[WARNING] %s: %s", getOrigin(), format)
	errLogger.Printf(format, v...)
}

func ErrorPrintf(format string, v ...interface{}) {
	format = fmt.Sprintf("[ERROR] %s: %s", getOrigin(), format)
	errLogger.Printf(format, v...)
}

func Fatalf(format string, v ...interface{}) {
	format = fmt.Sprintf("[FATAL] %s: %s", getOrigin(), format)
	errLogger.Fatalf(format, v...)
}
