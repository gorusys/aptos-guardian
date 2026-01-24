package version

import "runtime"

var (
	Version   = "0.1.0"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func Info() map[string]string {
	return map[string]string{
		"version":   Version,
		"commit":    Commit,
		"buildDate": BuildDate,
		"go":        runtime.Version(),
	}
}
