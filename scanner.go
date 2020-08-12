package mres

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

type (
	//Scanner use mres.NewScanner for creating new scanner
	Scanner struct {
		fileMatchers    []fileMatcher
		contentMatchers []contentMatcher
		logger          ILogger
	}

	fileMatcher struct {
		ID  string
		Exp *regexp.Regexp
	}

	contentMatcher struct {
		ID                string
		FileFilterEnabled bool
		FileFilterExp     *regexp.Regexp
		Exp               *regexp.Regexp
	}
)

//SetLogger helps set logger in scanner instance so logs of the scanner can be captured
func (s *Scanner) SetLogger(logger ILogger) {
	if logger == nil {
		return
	}
	s.logger = logger
	return
}

// ScanWithCallback starts the scan and calls the passed functions when there is any result or errors
// Incase if you want to stop the execution on error or anytime, call cancel on the context passed.
func (s *Scanner) ScanWithCallback(
	ctx context.Context,
	foldersToScan []string,
	workerCount int,
	onFileMatchResult func(r FileMatchResult),
	onContentMatchResult func(r ContentMatchResult),
	onError func(e error)) {
	if len(foldersToScan) == 0 {
		onError(ErrInvalidArgument)
		return
	}
	if workerCount < 1 {
		workerCount = 1
	}
	jobsC := make(chan string, workerCount*10)
	fmResultC := make(chan FileMatchResult, workerCount)
	cmResultsC := make(chan ContentMatchResult, workerCount)
	errorsC := make(chan error, workerCount)
	doneC := make(chan bool)
	wg := new(sync.WaitGroup)
	for w := 1; w <= workerCount; w++ {
		wg.Add(1)
		go s.worker(ctx, w, wg, jobsC, fmResultC, cmResultsC, errorsC)
	}
	go s.produceJobs(ctx, foldersToScan, jobsC, errorsC)
	go func() {
		wg.Wait()
		s.logger.Debug("Closing result and error channels")
		close(fmResultC)
		close(cmResultsC)
		close(errorsC)
		doneC <- true
		s.logger.Debug("Closing done channel")
		close(doneC)
	}()
	for {
		select {
		case <-doneC:
			// drain incase any leftovers - ASK: is this necessary?
			for r := range fmResultC {
				onFileMatchResult(r)
			}
			for r := range cmResultsC {
				onContentMatchResult(r)
			}
			for e := range errorsC {
				onError(e)
			}
			return
		case r, ok := <-fmResultC:
			if !ok {
				continue
			}
			onFileMatchResult(r)
		case r, ok := <-cmResultsC:
			if !ok {
				continue
			}
			onContentMatchResult(r)
		case e, ok := <-errorsC:
			if !ok {
				continue
			}
			onError(e)
		}
	}
}

// Scan starts the scan, the results and errors are returned in one go in the end.
// If you want to stop the scan, call cancel on the context passed.
// For continuous callback, check ScanWithCallback method.
func (s *Scanner) Scan(ctx context.Context, foldersToScan []string, workerCount int) (MatchResult, []error) {
	fmResults := make([]FileMatchResult, 0)
	cmResults := make([]ContentMatchResult, 0)
	errors := make([]error, 0)
	onFileMatchResult := func(r FileMatchResult) {
		fmResults = append(fmResults, r)
	}
	onContentMatchResult := func(r ContentMatchResult) {
		cmResults = append(cmResults, r)
	}
	onError := func(e error) {
		errors = append(errors, e)
	}
	s.ScanWithCallback(ctx, foldersToScan, workerCount, onFileMatchResult, onContentMatchResult, onError)
	result := MatchResult{
		FileMatches:    fmResults,
		ContentMatches: cmResults,
	}
	return result, errors
}

//produceJobs walks the folders and produces jobs for the workers
func (s *Scanner) produceJobs(
	ctx context.Context,
	foldersToScan []string,
	jobsC chan<- string,
	errorsC chan<- error) {
	processPath := func(path string, f os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return errReceivedCancellation
		default:
			if err != nil {
				errorsC <- err // normal errors send it to error channel
				return nil
			}
			if !f.IsDir() && (f.Mode()&os.ModeSymlink) != os.ModeSymlink { // skipping directory & symlink
				jobsC <- path
			}
			return nil
		}
	}
	for _, f := range foldersToScan {
		s.logger.Debug(fmt.Sprintf("Walking directory: %s", f))
		err := filepath.Walk(f, processPath)
		if err != nil {
			if err == errReceivedCancellation {
				s.logger.Debug("Received cancellation. Not walking the directory further")
				break
			} else {
				errorsC <- err
			}
		}
	}
	s.logger.Debug("Closing jobs channel")
	close(jobsC)
}

func (s *Scanner) worker(
	ctx context.Context,
	workerID int,
	wg *sync.WaitGroup,
	jobsC <-chan string,
	fmResultC chan<- FileMatchResult,
	cmResultsC chan<- ContentMatchResult,
	errorsC chan<- error) {
	defer wg.Done()
	end := func() {
		s.logger.Debug(fmt.Sprintf("Stopped worker: %d", workerID))
	}
	defer end()
	s.logger.Debug(fmt.Sprintf("Starting worker: %d", workerID))
	for {
		select {
		case <-ctx.Done():
			s.logger.Debug(fmt.Sprintf("Force stopping worker: %d", workerID))
			return
		case filePath, ok := <-jobsC:
			if !ok {
				return
			}
			// TODO: move this logic to an another file
			// TODO: line number, content sample, multiple matches on same file
			if len(s.fileMatchers) > 0 {
				for _, exp := range s.fileMatchers {
					if exp.Exp.Match([]byte(filePath)) {
						fmResultC <- FileMatchResult{RegExpID: exp.ID, FilePath: filePath}
					}
				}
			}
			if len(s.contentMatchers) == 0 {
				continue
			}
			fp, err := os.Open(filePath)
			if err != nil {
				errorsC <- err
				continue
			}
			content, err := ioutil.ReadAll(fp)
			// close it as soon as you are done with it. defer will keep files open until the function exits and in this case until the worker stops
			fp.Close()
			if err != nil {
				errorsC <- err
				continue
			}
			for _, exp := range s.contentMatchers {
				if exp.FileFilterEnabled && !exp.FileFilterExp.Match([]byte(filePath)) {
					continue
				}
				if exp.Exp.Match(content) {
					cmResultsC <- ContentMatchResult{RegExpID: exp.ID, FilePath: filePath, LineNumber: -1, MatchString: "TODO"}
				}
			}
		}
	}
}

//NewScanner creates a new scanner
func NewScanner(exps Expressions) (*Scanner, []error) {
	fileMatchers, contentMatchers, errs := buildMatchers(exps.FileMatchExps, exps.ContentMatchExps)
	if len(errs) > 0 {
		return nil, errs
	}
	scanner := &Scanner{
		fileMatchers:    fileMatchers,
		contentMatchers: contentMatchers,
		logger:          &noopLogger{},
	}
	return scanner, nil
}
