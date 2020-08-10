package scanner

import "errors"

var (
	errReceivedCancellation = errors.New("received cancellation")
)

type (
	RegExp struct {
		ID         string
		Expression string
	}

	Config struct {
		FoldersToScan []string
		Expressions   []RegExp
		WorkerCount   int
	}

	Result struct {
		RegExpID    string
		FilePath    string
		LineNumber  int
		MatchString string
	}

	ILogger interface {
		Debug(message string)
		Info(message string)
		Error(err error, message string)
	}

	noopLogger struct{}
)

func (l *noopLogger) Debug(message string) {
}

func (l *noopLogger) Info(message string) {
}

func (l *noopLogger) Error(err error, message string) {
}
