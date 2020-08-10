package scanner

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

//Scanner ...
type Scanner struct {
	logger   ILogger
	config   Config
	wg       *sync.WaitGroup
	regexMap map[string](*regexp.Regexp)
}

//SetLogger helps set logger in scanner instance so logs of the scanner can be captured
func (s *Scanner) SetLogger(logger ILogger) {
	if logger == nil {
		return
	}
	s.logger = logger
	return
}

//Scan starts the scan
func (s *Scanner) Scan(ctx context.Context) ([]Result, []error) {
	results := make([]Result, 0)
	errs := make([]error, 0)
	mapBuildErrs := s.buildRegexMap()
	errs = append(errs, mapBuildErrs...)
	channelBuffer := s.config.WorkerCount * 2 //little bit extra buffer for not blocking the senders and workers
	jobC := make(chan string, channelBuffer)
	resultC := make(chan Result, channelBuffer)
	errorC := make(chan error, channelBuffer)
	for w := 1; w <= s.config.WorkerCount; w++ {
		s.wg.Add(1)
		go s.scan(ctx, w, jobC, resultC, errorC)
	}

	processPath := func(path string, f os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return errReceivedCancellation
		default:
			if err != nil {
				errorC <- err // normal errors send it to error channel
				return nil
			}
			if !f.IsDir() && (f.Mode()&os.ModeSymlink) != os.ModeSymlink { // skipping directory & symlink
				jobC <- path
			}
			return nil
		}
	}

	go func() {
		for _, f := range s.config.FoldersToScan {
			s.logger.Info(fmt.Sprintf("Walking directory: %s", f))
			err := filepath.Walk(f, processPath)
			if err != nil {
				if err == errReceivedCancellation {
					break
				} else {
					errorC <- err
				}
			}
		}
		s.logger.Info("Closing jobs channel")
		close(jobC)
	}()

	go func() {
		s.wg.Wait()
		s.logger.Info("Closing results and errors channel")
		close(resultC)
		close(errorC)
	}()

	for {
		select {
		case r, ok := <-resultC:
			if !ok {
				if len(errorC) > 0 {
					continue
				} else {
					return results, errs
				}
			}
			results = append(results, r)
		case e, ok := <-errorC:
			if !ok {
				if len(resultC) > 0 {
					continue
				} else {
					return results, errs
				}
			}
			errs = append(errs, e)
		}
	}
}

func (s *Scanner) buildRegexMap() []error {
	regexMap := make(map[string]*regexp.Regexp)
	errs := make([]error, 0)
	for _, c := range s.config.Expressions {
		re, err := regexp.Compile(c.Expression)
		if err != nil {
			errs = append(errs, fmt.Errorf("error in compiling regex id: %s | err: %v", c.ID, err))
		} else {
			regexMap[c.ID] = re
		}
	}
	s.regexMap = regexMap
	return errs
}

func (s *Scanner) scan(ctx context.Context, workerID int, jobC <-chan string, resultC chan<- Result, errorC chan<- error) {
	defer s.wg.Done()
	end := func() {
		s.logger.Info(fmt.Sprintf("Stopped worker: %d", workerID))
	}
	defer end()
	s.logger.Info(fmt.Sprintf("Starting worker: %d", workerID))
	for {
		select {
		case <-ctx.Done():
			s.logger.Info(fmt.Sprintf("Force stopping worker: %d", workerID))
			return
		case j, ok := <-jobC:
			if !ok {
				return
			}
			fp, err := os.Open(j)
			if err != nil {
				errorC <- err
				continue
			}
			content, err := ioutil.ReadAll(fp)
			fp.Close()
			if err != nil {
				errorC <- err
				continue
			}
			for id, exp := range s.regexMap {
				if exp.Match(content) {
					resultC <- Result{RegExpID: id, FilePath: j}
				}
			}
		}
	}
}

//NewScanner creates a new scanner
func NewScanner(config Config) (*Scanner, error) {
	scanner := &Scanner{
		logger: &noopLogger{},
		config: config,
		wg:     new(sync.WaitGroup),
	}
	return scanner, nil
}
