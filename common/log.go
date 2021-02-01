package common

import (
	"log"
	"os"
	"strings"
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

	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	errLogger.Printf("[DEBUG] " + format, v...)
}

func InfoPrintf(format string, v ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	outLogger.Printf("[INFO] " + format, v...)
}

func WarnPrintf(format string, v ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	errLogger.Printf("[WARNING] " + format, v...)
}

func Fatalf(format string, v ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	errLogger.Fatalf("[FATAL] " + format, v...)
}
