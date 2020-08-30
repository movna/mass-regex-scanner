package mres

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
)

type (
	ContentMatchExp struct {
		ID                string       `json:"id,omitempty"`
		FileFilterEnabled bool         `json:"file_filter_enabled,omitempty"`
		FileMatchExp      FileMatchExp `json:"file_match_exp,omitempty"`
		Exp               string       `json:"exp,omitempty"`
		//FlipMatch since Go doesn't support negative look ahead
		//FlipMatch bool `json:"flip_match,omitempty"`
	}

	contentMatcher struct {
		ID          string
		fileMatcher *fileMatcher
		Exp         *regexp.Regexp
		//FlipMatch   bool
	}

	contentMatchers []contentMatcher

	ContentMatchResult struct {
		ExpID       string `json:"exp_id,omitempty"`
		FilePath    string `json:"file_path,omitempty"`
		LineNumber  int    `json:"line_number,omitempty"`
		MatchString string `json:"match_string,omitempty"`
	}
)

func (matchers contentMatchers) matchAll(filePath string, bufPool []byte) ([]ContentMatchResult, error) {
	results := make([]ContentMatchResult, 0)
	if len(matchers) == 0 {
		return results, nil
	}
	applicableMatchers := matchers.filterApplicable(filePath)
	if len(applicableMatchers) == 0 {
		return results, nil
	}
	fp, err := os.OpenFile(filePath, os.O_RDONLY, os.ModePerm)
	defer fp.Close()
	if err != nil {
		return results, err
	}
	scanner := bufio.NewScanner(fp)
	scanner.Buffer(bufPool, cap(bufPool))
	lineNo := 0
	for scanner.Scan() {
		if scanner.Err() != nil {
			return results, err
		}
		lineNo++
		content := scanner.Bytes()
		for _, m := range applicableMatchers {
			match := m.Exp.Match(content)
			if match {
				matches := m.Exp.FindAll(content, -1)
				if len(matches) == 0 {
					continue
				}
				for _, match := range matches {
					results = append(results, ContentMatchResult{ExpID: m.ID, FilePath: filePath, LineNumber: lineNo, MatchString: string(match)})
				}
			}
		}
	}
	return results, nil
}

func (matchers contentMatchers) filterApplicable(filePath string) contentMatchers {
	applicableMatchers := make(contentMatchers, 0)
	pathBytes := []byte(filePath)
	for _, m := range matchers {
		if m.fileMatcher == nil {
			applicableMatchers = append(applicableMatchers, m)
			continue
		}
		if m.fileMatcher.match(pathBytes) {
			applicableMatchers = append(applicableMatchers, m)
		}
	}
	return applicableMatchers
}

func newContentMatcher(e ContentMatchExp) (contentMatcher, []error) {
	m := contentMatcher{}
	errs := make([]error, 0)
	if e.FileFilterEnabled {
		e.FileMatchExp.ID = e.ID
		fm, err := newFileMatcher(e.FileMatchExp)
		if err != nil {
			errs = append(errs, fmt.Errorf("error: %v while compiling content match *file filter* exp for id: %s", err, e.ID))
		} else {
			m.fileMatcher = &fm
		}
	}
	compiled, err := regexp.Compile(e.Exp)
	if err != nil {
		errs = append(errs, fmt.Errorf("error: %v while compiling content match exp for id: %s", err, e.ID))
		return m, errs
	}
	m.Exp = compiled
	m.ID = e.ID
	//m.FlipMatch = e.FlipMatch
	return m, errs
}

//buildContentMatchers is a helper function to compile content match expressions
func buildContentMatchers(exps []ContentMatchExp) (contentMatchers, []error) {
	matchers := make(contentMatchers, 0, len(exps))
	errs := make([]error, 0)
	if len(exps) == 0 {
		return matchers, errs
	}
	for _, e := range exps {
		matcher, errsin := newContentMatcher(e)
		if len(errsin) > 0 {
			errs = append(errs, errsin...)
			continue
		}
		matchers = append(matchers, matcher)
	}
	return matchers, errs
}
