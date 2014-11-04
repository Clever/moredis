package logger

import (
	"fmt"
	"log"

	"github.com/clever/kayvee-go"
)

type M map[string]interface{}

const (
	CRITICAL = "critical"
	ERROR    = "error"
	WARNING  = "warning"
	INFO     = "info"
	TRACE    = "trace"
)

func Info(title string, data M) {
	logWithLevel(title, INFO, data)
}

func Trace(title string, data M) {
	logWithLevel(title, TRACE, data)
}

func Warning(title string, data M) {
	logWithLevel(title, WARNING, data)
}

func Critical(title string, data M) {
	logWithLevel(title, CRITICAL, data)
}

func Error(title string, err error) {
	formatted := kayvee.FormatLog("moredis", ERROR, title, map[string]interface{}{"error": fmt.Sprint(err)})
	log.Fatal(formatted)
}

func ErrorDetailed(title string, err error, extras M) {
	extras["error"] = fmt.Sprint(err)
	formatted := kayvee.FormatLog("moredis", ERROR, title, extras)
	log.Fatal(formatted)
}

func logWithLevel(title string, level string, data M) {
	formatted := kayvee.FormatLog("moredis", level, title, data)
	log.Println(formatted)
}
