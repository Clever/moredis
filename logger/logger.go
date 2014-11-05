package logger

import (
	"fmt"
	"log"

	"github.com/clever/kayvee-go"
)

// M is an alias for map[string]interface{} to make log lines less painful to write.
type M map[string]interface{}

// Logging levels
const (
	CRITICAL = "critical"
	ERROR    = "error"
	WARNING  = "warning"
	INFO     = "info"
	TRACE    = "trace"
)

// Info logs at the info level
func Info(title string, data M) {
	logWithLevel(title, INFO, data)
}

// Trace logs at the trace level
func Trace(title string, data M) {
	logWithLevel(title, TRACE, data)
}

// Warning logs at the warning level
func Warning(title string, data M) {
	logWithLevel(title, WARNING, data)
}

// Critical logs at teh critical level
func Critical(title string, data M) {
	logWithLevel(title, CRITICAL, data)
}

// Error logs an error at the error level
func Error(title string, err error) {
	logWithLevel(title, ERROR, M{"error": fmt.Sprint(err)})
}

// ErrorDetailed logs an error at the error level, along with an extra info you can provide
func ErrorDetailed(title string, err error, extras M) {
	extras["error"] = fmt.Sprint(err)
	logWithLevel(title, ERROR, M{"error": fmt.Sprint(err)})
}

func logWithLevel(title string, level string, data M) {
	formatted := kayvee.FormatLog("moredis", level, title, data)
	log.Println(formatted)
}
