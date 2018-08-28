package trace

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/IBM-Cloud/ibm-cloud-cli-sdk/bluemix/terminal"
	. "github.com/IBM-Cloud/ibm-cloud-cli-sdk/i18n"
)

type Printer interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

type Closer interface {
	Close() error
}

type PrinterCloser interface {
	Printer
	Closer
}

type NullLogger struct{}

func (l *NullLogger) Print(v ...interface{})                 {}
func (l *NullLogger) Printf(format string, v ...interface{}) {}
func (l *NullLogger) Println(v ...interface{})               {}

type loggerImpl struct {
	*log.Logger
	c io.WriteCloser
}

func (loggerImpl *loggerImpl) Close() error {
	if loggerImpl.c != nil {
		return loggerImpl.c.Close()
	}
	return nil
}

func newLoggerImpl(out io.Writer, prefix string, flag int) *loggerImpl {
	l := log.New(out, prefix, flag)
	c, _ := out.(io.WriteCloser)
	return &loggerImpl{
		Logger: l,
		c:      c,
	}
}

var Logger Printer = NewLogger("")

// NewLogger returns a printer for the given trace setting.
func NewLogger(bluemix_trace string) Printer {
	switch strings.ToLower(bluemix_trace) {
	case "", "false":
		return new(NullLogger)
	case "true":
		return NewStdLogger()
	default:
		return NewFileLogger(bluemix_trace)
	}
}

// NewStdLogger creates a a printer that writes to StdOut.
func NewStdLogger() PrinterCloser {
	return newLoggerImpl(terminal.ErrOutput, "", 0)
}

// NewFileLogger creates a printer that writes to the given file path.
func NewFileLogger(path string) PrinterCloser {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		logger := NewStdLogger()
		logger.Printf(T("An error occurred when creating log file '{{.Path}}':\n{{.Error}}\n\n", map[string]interface{}{"Path": path, "Error": err.Error()}))
		return logger
	}
	return newLoggerImpl(file, "", 0)
}

var privateDataPlaceholder = "[PRIVATE DATA HIDDEN]"

// Sanitize returns a clean string with sensitive user data in the input
// replaced by PRIVATE_DATA_PLACEHOLDER.
func Sanitize(input string) string {
	re := regexp.MustCompile(`(?m)^(Authorization|X-Auth\S*): .*`)
	sanitized := re.ReplaceAllString(input, "$1: "+privateDataPlaceholder)

	re = regexp.MustCompile(`(?i)(password|token|apikey|passcode)=[^&]*(&|$)`)
	sanitized = re.ReplaceAllString(sanitized, "$1="+privateDataPlaceholder+"$2")

	re = regexp.MustCompile(`(?i)"([^"]*(password|token|apikey)[^"_]*)":\s*"[^\,]*"`)
	sanitized = re.ReplaceAllString(sanitized, fmt.Sprintf(`"$1":"%s"`, privateDataPlaceholder))

	return sanitized
}
