package cmdlogger

import (
	"io"
	"os"

	"github.com/Cloud-Foundations/golib/pkg/log/debuglogger"
)

type Options struct {
	Datestamps bool
	DebugLevel int // Supported range: -1 to 65535.
	Subseconds bool
	Writer     io.Writer
}

var stdOptions = Options{Writer: os.Stderr}

// GetStandardOptions will return the standard options.
// The following command-line flags are registered and used:
//   -logDatestamps: if true, prefix logs with datestamps
//   -logDebugLevel: debug log level
//   -logSubseconds: if true, datestamps will have subsecond resolution
//  The standard error is used for the output.
func GetStandardOptions() Options { return stdOptions }

// New will create a debuglogger.Logger with the specified options.
func New(options Options) *debuglogger.Logger {
	return newLogger(options)
}

// SetDatestampsDefault will change the default for the -logDatestamps command
// line flag. This should be called before flag.Parse().
func SetDatestampsDefault(defaultValue bool) {
	stdOptions.Datestamps = defaultValue
}
