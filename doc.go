package mres

import (
	"errors"
)

var (
	//ErrInvalidArgument ...
	ErrInvalidArgument = errors.New("invalid argument")

	errReceivedCancellation = errors.New("received cancellation")
)

type (
	FileMatchExp struct {
		ID  string
		Exp string
	}

	ContentMatchExp struct {
		ID                string
		FileFilterEnabled bool
		FileFilterExp     string
		Exp               string
	}

	Expressions struct {
		FileMatchExps    []FileMatchExp
		ContentMatchExps []ContentMatchExp
	}

	MatchResult struct {
		FileMatches    []FileMatchResult
		ContentMatches []ContentMatchResult
	}

	FileMatchResult struct {
		RegExpID string
		FilePath string
	}

	ContentMatchResult struct {
		RegExpID    string
		FilePath    string
		LineNumber  int
		MatchString string
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
