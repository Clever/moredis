package logger

import (
	"fmt"
	"log"

	"gopkg.in/clever/kayvee-go.v2"
)

// M is an alias for map[string]interface{} to make log lines less painful to write.
type M map[string]interface{}

// Info logs at the info level
func Info(title string, data M) {
	logWithLevel(title, kayvee.Info, data)
}

// Trace logs at the trace level
func Trace(title string, data M) {
	logWithLevel(title, kayvee.Trace, data)
}

// Warning logs at the warning level
func Warning(title string, data M) {
	logWithLevel(title, kayvee.Warning, data)
}

// Critical logs at teh critical level
func Critical(title string, data M) {
	logWithLevel(title, kayvee.Critical, data)
}

// Error logs an error at the error level
func Error(title string, err error) {
	logWithLevel(title, kayvee.Error, M{"error": fmt.Sprint(err)})
}

// ErrorDetailed logs an error at the error level, along with an extra info you can provide
func ErrorDetailed(title string, err error, extras M) {
	extras["error"] = fmt.Sprint(err)
	logWithLevel(title, kayvee.Error, extras)
}

func logWithLevel(title string, level kayvee.LogLevel, data M) {
	formatted := kayvee.FormatLog("moredis", level, title, data)
	log.Println(formatted)
}
