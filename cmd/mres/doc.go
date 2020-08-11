package main

import (
	"errors"

	"github.com/movna/mres"
	"github.com/movna/mres/internal"
)

var (
	log                  = internal.NewDefaultLogger()
	errInvalidCliOptions = errors.New("invalid cli options")
)

type cliOptions struct {
	mresExpressions mres.Expressions
	foldersToScan   []string
	workerCount     int
	outputToFile    bool
	outputFilePath  string
}
