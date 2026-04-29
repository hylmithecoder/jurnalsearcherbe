package handler

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// getCallerInfo returns full/relative path, line, and function name
func getCallerInfo(skip int) (string, int, string) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "???", 0, "unknown"
	}

	fn := runtime.FuncForPC(pc)
	funcName := "unknown"
	if fn != nil {
		funcName = fn.Name()
	}

	// Try to make path relative to working directory (cleaner than full absolute)
	wd, err := os.Getwd()
	if err == nil {
		if rel, err := filepath.Rel(wd, file); err == nil {
			file = rel
		}
	}

	return file, line, funcName
}

// Debug mencetak log dengan informasi lengkap
func Debug(format string, a ...interface{}) {
	file, line, funcName := getCallerInfo(2)

	timestamp := time.Now().Add(7 * time.Hour).Format("15:04:05") // UTC+7 WIB
	msg := fmt.Sprintf(format, a...)

	fmt.Printf("\x1b[36m[%s] [DEBUG]\x1b[0m \x1b[90m%s:%d (%s)\x1b[0m | %s\n",
		timestamp,
		file,
		line,
		funcName,
		msg,
	)
}

// LogErr mencetak log error dengan informasi lengkap
func LogErr(format string, a ...interface{}) {
	file, line, funcName := getCallerInfo(2)

	timestamp := time.Now().Add(7 * time.Hour).Format("15:04:05") // UTC+7 WIB
	msg := fmt.Sprintf(format, a...)

	fmt.Printf("\x1b[31m[%s] [ERROR]\x1b[0m \x1b[90m%s:%d (%s)\x1b[0m | %s\n",
		timestamp,
		file,
		line,
		funcName,
		msg,
	)
}

// LogInfo mencetak log informasi umum
func LogInfo(format string, a ...interface{}) {
	timestamp := time.Now().Add(7 * time.Hour).Format("15:04:05")
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("\x1b[32m[%s] [INFO]\x1b[0m | %s\n", timestamp, msg)
}