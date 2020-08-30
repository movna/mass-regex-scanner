package mres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type (
	Expressions struct {
		FileMatchExps    []FileMatchExp    `json:"file_match_exps,omitempty"`
		ContentMatchExps []ContentMatchExp `json:"content_match_exps,omitempty"`
	}

	MatchResult struct {
		FileMatches    []FileMatchResult    `json:"file_matches,omitempty"`
		ContentMatches []ContentMatchResult `json:"content_matches,omitempty"`
	}

	ILogger interface {
		Debug(message string)
		Info(message string)
		Error(err error, message string)
	}

	//Scanner use mres.NewScanner for creating new scanner
	Scanner struct {
		fileMatchers    fileMatchers
		contentMatchers contentMatchers
		logger          ILogger
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
	ctx context.Context, pathsToScan []string, workerCount int,
	onFileMatchResult func(r FileMatchResult),
	onContentMatchResult func(r ContentMatchResult),
	onError func(e error)) {
	if len(pathsToScan) == 0 {
		onError(ErrInvalidArgument)
		return
	}
	if workerCount < 1 {
		workerCount = 1
	}
	jobC := make(chan string, workerCount*10)
	fmResultC := make(chan FileMatchResult, workerCount)
	cmResultC := make(chan ContentMatchResult, workerCount)
	errorC := make(chan error, workerCount)
	doneC := make(chan struct{})
	wg := new(sync.WaitGroup)
	for w := 1; w <= workerCount; w++ {
		wg.Add(1)
		go s.scanWorker(ctx, w, wg, jobC, fmResultC, cmResultC, errorC)
	}
	go s.walkPaths(ctx, pathsToScan, jobC, errorC)
	go func() {
		wg.Wait()
		s.logger.Debug("Closing result and error channels")
		close(fmResultC)
		close(cmResultC)
		close(errorC)
		doneC <- struct{}{}
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
			for r := range cmResultC {
				onContentMatchResult(r)
			}
			for e := range errorC {
				onError(e)
			}
			return
		case r, ok := <-fmResultC:
			if !ok {
				continue
			}
			onFileMatchResult(r)
		case r, ok := <-cmResultC:
			if !ok {
				continue
			}
			onContentMatchResult(r)
		case e, ok := <-errorC:
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

//walkPaths walks the folders and produces jobs for the workers
func (s *Scanner) walkPaths(ctx context.Context, pathsToScan []string, jobsC chan<- string, errorsC chan<- error) {
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
	for _, f := range pathsToScan {
		s.logger.Debug(fmt.Sprintf("Walking path: %s", f))
		err := filepath.Walk(f, processPath)
		if err != nil {
			if err == errReceivedCancellation {
				s.logger.Debug("Received cancellation. Not walking the paths further")
				break
			} else {
				errorsC <- err
			}
		}
	}
	s.logger.Debug("Closing jobs channel")
	close(jobsC)
}

func (s *Scanner) scanWorker(
	ctx context.Context, workerID int, wg *sync.WaitGroup,
	jobC <-chan string, fmResultC chan<- FileMatchResult, cmResultC chan<- ContentMatchResult, errorC chan<- error) {
	defer func() {
		wg.Done()
		s.logger.Debug(fmt.Sprintf("Stopped worker: %d", workerID))
	}()
	s.logger.Debug(fmt.Sprintf("Starting worker: %d", workerID))
	bufPool := make([]byte, 0, 10*1024*1024)
	for {
		select {
		case <-ctx.Done():
			s.logger.Debug(fmt.Sprintf("Force stopping worker: %d", workerID))
			return
		case filePath, ok := <-jobC:
			if !ok {
				return
			}
			for _, r := range s.fileMatchers.matchAll(filePath) {
				fmResultC <- r
			}
			contentResults, err := s.contentMatchers.matchAll(filePath, bufPool)
			if err != nil {
				errorC <- err
				continue
			}
			for _, r := range contentResults {
				cmResultC <- r
			}
		}
	}
}

//NewScanner creates a new scanner
func NewScanner(exps Expressions) (*Scanner, []error) {
	fileMatchers, errs1 := buildFileMatchers(exps.FileMatchExps)
	contentMatchers, errs2 := buildContentMatchers(exps.ContentMatchExps)
	errs := make([]error, 0, len(errs1)+len(errs2))
	errs = append(errs, errs1...)
	errs = append(errs, errs2...)
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

type noopLogger struct{}

func (l *noopLogger) Debug(message string) {
}

func (l *noopLogger) Info(message string) {
}

func (l *noopLogger) Error(err error, message string) {
}
