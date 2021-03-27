package log

import (
	"os"

	"github.com/op/go-logging"
)

var Logger = logging.MustGetLogger("example")

var format = logging.MustStringFormatter(
	`%{color}%{time:2006-01-02 15:04:05} %{shortfile} -> [%{level:.4s}] %{color:reset} %{message}`,
)

// Password is just an example type implementing the Redactor interface. Any
// time this is logged, the Redacted() function will be called.
type Password string

func (p Password) Redacted() interface{} {
	return logging.Redact(string(p))
}

func init() {
	backend1 := logging.NewLogBackend(os.Stdout, "", 0)
	backend2Formatter := logging.NewBackendFormatter(backend1, format)
	logging.SetBackend(backend2Formatter)
}
