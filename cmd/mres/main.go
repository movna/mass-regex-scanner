package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

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

func printRuntimeStats() {
	go func() {
		for {
			ms := runtime.MemStats{}
			runtime.ReadMemStats(&ms)
			stack := float64(ms.StackInuse) / float64(1024*1024)
			heap := float64(ms.HeapInuse) / float64(1024*1024)
			memObtained := float64(ms.Sys) / float64(1024*1024)
			log.Debug(fmt.Sprintf("Goroutines: %d Stack: %.2fmB Heap: %.2fmB MemObtained: %.2fmB", runtime.NumGoroutine(), stack, heap, memObtained))
			time.Sleep(time.Second * 1)
		}
	}()
}

func main() {
	printRuntimeStats()
	Run()
}

//Run ...
func Run() {
	cliOptions, err := parseCliOptions()
	if err != nil {
		log.Error(err, "Cannot continue further")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, os.Interrupt, os.Kill)
	go func() {
		sig := <-signalC
		log.Info(fmt.Sprintf("Received: %s. Stopping...", sig))
		cancel()
	}()
	scanner, errs := mres.NewScanner(cliOptions.mresExpressions)
	if len(errs) > 0 {
		for _, e := range errs {
			log.Error(e, "")
		}
		return
	}
	scanner.SetLogger(log)
	fmResultsCount := 0
	cmResultsCount := 0
	errorsCount := 0
	onFileMatchResult := func(r mres.FileMatchResult) {
		fmResultsCount++
		if !cliOptions.outputToFile {
			log.Info(fmt.Sprintf("File match - id: %s, filepath: %s", r.ExpID, r.FilePath))
		}
	}
	onContentMatchResult := func(r mres.ContentMatchResult) {
		cmResultsCount++
		if !cliOptions.outputToFile {
			log.Info(fmt.Sprintf("Content match - id: %s, filepath: %s, lineno: %d, match: %s", r.ExpID, r.FilePath, r.LineNumber, r.MatchString))
		}
	}
	onError := func(e error) {
		errorsCount++
		if !cliOptions.outputToFile {
			log.Error(e, "")
		}
	}
	start := time.Now()
	scanner.ScanWithCallback(ctx, cliOptions.foldersToScan, cliOptions.workerCount, onFileMatchResult, onContentMatchResult, onError)
	timeTaken := time.Now().Sub(start)
	log.Info(fmt.Sprintf("Timetaken: %s", timeTaken))
	log.Info(fmt.Sprintf("Total results: %d", fmResultsCount+cmResultsCount))
	log.Info(fmt.Sprintf("Total errors: %d", errorsCount))
	if cliOptions.outputToFile {
		log.Info(fmt.Sprintf("Output written to file: %s", cliOptions.outputFilePath))
	}
}

func parseCliOptions() (*cliOptions, error) {
	// flags
	//configPathPtr := flag.String("config", "", "Relative or absolute path to the config file")
	pathPtr := flag.String("path", "", "Relative or absolute path of the folder or a file to scan")
	fileRegexStrPtr := flag.String("file", "", "This is a regex supported flag which can be used to filter files with specific extensions or in specific subpath relative to the given path")
	contentRegexStrPtr := flag.String("content", "", "Regular Expression")
	resultDumpPathPtr := flag.String("out", "", "Relative or absolute path to the dump the results. The results will be written in JSON format. If a value is not specified, the results will be written to Stdout.")
	workerCountPtr := flag.Int("workers", 2, "Number of workers. Increase it if you are scanning through large number of files and complex regular expressions.")
	flag.Parse()
	if *pathPtr == "" {
		log.Info("Invalid path value specified. Check help by using -help option.")
		return nil, errInvalidCliOptions
	}
	fileFilterEnabled := false
	if *fileRegexStrPtr != "" {
		fileFilterEnabled = true
	}
	contentFilterEnabled := false
	if *contentRegexStrPtr != "" {
		contentFilterEnabled = true
	}
	if !fileFilterEnabled && !contentFilterEnabled {
		log.Info("Specify either -filefilter or -regex or both. Check help by using -help option.")
		return nil, errInvalidCliOptions
	}
	workerCount := *workerCountPtr
	if workerCount < 1 {
		workerCount = 1
	}
	mresExp := mres.Expressions{}
	switch {
	case contentFilterEnabled:
		mresExp.ContentMatchExps = []mres.ContentMatchExp{
			{
				ID:                "cli",
				Exp:               *contentRegexStrPtr,
				FileFilterEnabled: fileFilterEnabled,
				FileMatchExp: mres.FileMatchExp{
					Exp: *fileRegexStrPtr,
				},
			},
		}
	case fileFilterEnabled && !contentFilterEnabled:
		mresExp.FileMatchExps = []mres.FileMatchExp{
			{
				ID:  "cli",
				Exp: *fileRegexStrPtr,
			},
		}

	}
	cliOptions := &cliOptions{
		mresExpressions: mresExp,
		foldersToScan:   []string{*pathPtr},
		workerCount:     workerCount,
		outputToFile:    *resultDumpPathPtr != "",
		outputFilePath:  *resultDumpPathPtr,
	}
	return cliOptions, nil
}
