package common

import (
	"log"
	"os"
)

var (
	outLogger *log.Logger
	errLogger *log.Logger
)

func init() {
	flag := log.Ldate|log.Ltime|log.Lshortfile
	outLogger = log.New(os.Stdout, "", flag)
	errLogger = log.New(os.Stderr, "", flag)
}

func DebugPrintf(format string, v ...interface{}) {
	if !AppConfig.Debug {
		return
	}
	errLogger.Printf("[DEBUG] " + format, v...)
}

func InfoPrintf(format string, v ...interface{}) {
	outLogger.Printf("[INFO] " + format, v...)
}

func WarnPrintf(format string, v ...interface{}) {
	errLogger.Printf("[WARNING] " + format, v...)
}

func ErrorPrintf(format string, v ...interface{}) {
	errLogger.Printf("[ERROR] " + format, v...)
}

func Fatalf(format string, v ...interface{}) {
	errLogger.Fatalf("[FATAL] " + format, v...)
}
