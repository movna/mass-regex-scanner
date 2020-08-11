package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/movna/mres"
)

func main() {
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
			log.Info(fmt.Sprintf("File match - id: %s filepath: %s", r.RegExpID, r.FilePath))
		}
	}
	onContentMatchResult := func(r mres.ContentMatchResult) {
		cmResultsCount++
		if !cliOptions.outputToFile {
			log.Info(fmt.Sprintf("Content match - id: %s filepath: %s", r.RegExpID, r.FilePath))
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
	regexStrPtr := flag.String("regex", "", "Regular Expression")
	resultDumpPathPtr := flag.String("out", "", "Relative or absolute path to the dump the results. The results will be written in JSON format. If a value is not specified, the results will be written to Stdout.")
	workerCountPtr := flag.Int("workers", 2, "Number of workers. Increase it if you are scanning through large number of files and complex regular expressions.")
	flag.Parse()
	if *pathPtr == "" {
		log.Info("Invalid path value specified. Check help by using -help option.")
		return nil, errInvalidCliOptions
	}
	if *regexStrPtr == "" {
		log.Info("Invalid regex value specified. Check help by using -help option.")
		return nil, errInvalidCliOptions
	}
	/*
		there is not much use setting beyond this. file regex scanning is both disk io and cpu bound. if there are
		large volume of files but a simple regex - the operation is highly bounded by disk io and if the files are of
		low volume but the regexes are complex - the operation is highly bounded by cpu
	*/
	workerCount := *workerCountPtr
	if workerCount < 1 {
		workerCount = 1
	}
	cliOptions := &cliOptions{
		mresExpressions: mres.Expressions{
			ContentMatchExps: []mres.ContentMatchExp{
				{ID: "cli", Exp: *regexStrPtr, FileFilterEnabled: false, FileFilterExp: ""},
			},
		},
		foldersToScan:  []string{*pathPtr},
		workerCount:    workerCount,
		outputToFile:   *resultDumpPathPtr != "",
		outputFilePath: *resultDumpPathPtr,
	}
	return cliOptions, nil
}
