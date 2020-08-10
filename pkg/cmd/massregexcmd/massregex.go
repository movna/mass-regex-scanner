package massregexcmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/movna/mass-regex-scanner/internal"
	"github.com/movna/mass-regex-scanner/pkg/scanner"
)

var (
	log                  = internal.NewDefaultLogger()
	errInvalidCliOptions = errors.New("invalid cli options")
)

type cliOptions struct {
	scannerConfig  scanner.Config
	outputToFile   bool
	outputFilePath string
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
	maxWorkerCount := runtime.NumCPU() * 2
	if workerCount <= 0 {
		workerCount = 2
	} else if workerCount > maxWorkerCount {
		workerCount = maxWorkerCount
	}
	cliOptions := &cliOptions{
		scannerConfig: scanner.Config{
			FoldersToScan: []string{*pathPtr},
			Expressions: []scanner.RegExp{
				{ID: "cli", Expression: *regexStrPtr},
			},
			WorkerCount: workerCount,
		},
		outputToFile:   *resultDumpPathPtr != "",
		outputFilePath: *resultDumpPathPtr,
	}
	return cliOptions, nil
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
	scanner, err := scanner.NewScanner(cliOptions.scannerConfig)
	if err != nil {
		log.Error(err, "Cannot continue further")
		return
	}
	scanner.SetLogger(log)
	start := time.Now()
	results, errs := scanner.Scan(ctx)
	timeTaken := time.Now().Sub(start)
	log.Info(fmt.Sprintf("Timetaken: %s", timeTaken))
	if !cliOptions.outputToFile {
		log.Info(fmt.Sprintf("Results: %d", len(results)))
		if results != nil {
			for _, r := range results {
				log.Info(fmt.Sprintf("finding: %v", r))
			}
		}
		log.Info(fmt.Sprintf("Errors: %d", len(errs)))
		if errs != nil {
			for _, e := range errs {
				log.Error(e, "Error from scan")
			}
		}
	}
}
