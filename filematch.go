package mres

import (
	"fmt"
	"regexp"
)

type (
	FileMatchExp struct {
		ID  string `json:"id,omitempty"`
		Exp string `json:"exp,omitempty"`
		//FlipMatch since Go doesn't support negative look ahead
		FlipMatch bool `json:"flip_match,omitempty"`
	}

	fileMatcher struct {
		ID        string
		Exp       *regexp.Regexp
		FlipMatch bool
	}

	fileMatchers []fileMatcher

	FileMatchResult struct {
		ExpID    string `json:"exp_id,omitempty"`
		FilePath string `json:"file_path,omitempty"`
	}
)

func (m *fileMatcher) match(filePath []byte) bool {
	match := m.Exp.Match(filePath)
	//return match != m.FlipMatch
	if m.FlipMatch {
		match = !match
	}
	return match
}

func (matchers fileMatchers) matchAll(filePath string) []FileMatchResult {
	results := make([]FileMatchResult, 0)
	if len(matchers) == 0 {
		return results
	}
	pathBytes := []byte(filePath)
	for _, m := range matchers {
		if !m.match(pathBytes) {
			continue
		}
		results = append(results, FileMatchResult{ExpID: m.ID, FilePath: filePath})
	}
	return results
}

func newFileMatcher(e FileMatchExp) (fileMatcher, error) {
	compiledExp, err := regexp.Compile(e.Exp)
	if err != nil {
		return fileMatcher{}, fmt.Errorf("error: %v while compiling file match exp for id: %s", err, e.ID)
	}
	matcher := fileMatcher{
		ID:        e.ID,
		Exp:       compiledExp,
		FlipMatch: e.FlipMatch,
	}
	return matcher, nil
}

//buildFileMatchers is a helper function to compile file match expressions
func buildFileMatchers(exps []FileMatchExp) (fileMatchers, []error) {
	matchers := make(fileMatchers, 0, len(exps))
	errs := make([]error, 0)
	if len(exps) == 0 {
		return matchers, errs
	}
	for _, e := range exps {
		matcher, err := newFileMatcher(e)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		matchers = append(matchers, matcher)
	}
	return matchers, errs
}
