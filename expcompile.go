package mres

import (
	"fmt"
	"regexp"
)

//compileFileMatchExps is a helper function to compile file match expressions
func compileFileMatchExps(exps []FileMatchExp) ([]fileMatcher, []error) {
	matchers := make([]fileMatcher, 0)
	errs := make([]error, 0)
	if len(exps) == 0 {
		return matchers, errs
	}
	for _, e := range exps {
		compiledExp, err := regexp.Compile(e.Exp)
		if err != nil {
			errs = append(errs, fmt.Errorf("error: %v while compiling file match exp for id: %s", err, e.ID))
			continue
		}
		matcher := fileMatcher{}
		matcher.Exp = compiledExp
		matcher.ID = e.ID
		matchers = append(matchers, matcher)
	}
	return matchers, errs
}

//compileFileMatchExps is a helper function to compile content match expressions
func compileContentMatchExps(exps []ContentMatchExp) ([]contentMatcher, []error) {
	matchers := make([]contentMatcher, 0)
	errs := make([]error, 0)
	if len(exps) == 0 {
		return matchers, errs
	}
	for _, e := range exps {
		matcher := contentMatcher{}
		if e.FileFilterEnabled {
			matcher.FileFilterEnabled = true
			compiled, err := regexp.Compile(e.FileFilterExp)
			if err != nil {
				errs = append(errs, fmt.Errorf("error: %v while compiling content match file filter exp for id: %s", err, e.ID))
				continue
			}
			matcher.FileFilterExp = compiled
		}
		compiled, err := regexp.Compile(e.Exp)
		if err != nil {
			errs = append(errs, fmt.Errorf("error: %v while compiling content match exp for id: %s", err, e.ID))
			continue
		}
		matcher.Exp = compiled
		matcher.ID = e.ID
		matchers = append(matchers, matcher)
	}
	return matchers, errs
}

func buildMatchers(fileExps []FileMatchExp, contentExps []ContentMatchExp) ([]fileMatcher, []contentMatcher, []error) {
	fileMatchers, errs1 := compileFileMatchExps(fileExps)
	contentMatchers, errs2 := compileContentMatchExps(contentExps)
	errs := make([]error, 0) //TODO: calculate length and build errs
	errs = append(errs, errs1...)
	errs = append(errs, errs2...)
	return fileMatchers, contentMatchers, errs
}
