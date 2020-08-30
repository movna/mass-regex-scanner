package mres

import (
	"testing"
)

func Test_buildFileMatchers(t *testing.T) {
	type args struct {
		exps []FileMatchExp
	}
	tests := []struct {
		name        string
		args        args
		matchersLen int
		errorsLen   int
	}{
		{
			name: "with no error",
			args: args{
				exps: []FileMatchExp{
					newFileMatchExp("id1", ".go"),
					newFileMatchExp("id2", ".go|.txt"),
					newFileMatchExp("id3", ".db"),
					newFileMatchExp("id4", "(?i).go|.txt"),
				},
			},
			matchersLen: 4,
			errorsLen:   0,
		},
		{
			name: "with error",
			args: args{
				exps: []FileMatchExp{
					newFileMatchExp("id1", ".go"),
					newFileMatchExp("id2", ".go|.txt"),
					newFileMatchExp("id3", ".db"),
					newFileMatchExp("id4", "(?i).go|.txt"),
					newFileMatchExp("id5", "(?i)).go|.txt"),
				},
			},
			matchersLen: 4,
			errorsLen:   1,
		},
		{
			name: "nil exps",
			args: args{
				exps: nil,
			},
			matchersLen: 0,
			errorsLen:   0,
		},
		{
			name: "empty exps",
			args: args{
				exps: []FileMatchExp{},
			},
			matchersLen: 0,
			errorsLen:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matchers, errs := buildFileMatchers(tt.args.exps)
			if len(matchers) != tt.matchersLen {
				t.Errorf("buildFileMatchers() matchers = %v, want %v", len(matchers), tt.matchersLen)
			}
			if len(errs) != tt.errorsLen {
				t.Errorf("buildFileMatchers() errs = %v, want %v", len(errs), tt.errorsLen)
			}
		})
	}
}

func Test_buildContentMatchers(t *testing.T) {
	type args struct {
		exps []ContentMatchExp
	}
	tests := []struct {
		name        string
		args        args
		matchersLen int
		errorsLen   int
	}{
		{
			name: "with no error",
			args: args{
				exps: []ContentMatchExp{
					newContentMatchExp("id1", true, ".go", "todo"),
					newContentMatchExp("id2", true, ".go|.txt", "test"),
					newContentMatchExp("id3", false, ".db", "secret"),
					newContentMatchExp("id4", false, "(?i).go|.txt", "(?i)todo"),
				},
			},
			matchersLen: 4,
			errorsLen:   0,
		},
		{
			name: "with error",
			args: args{
				exps: []ContentMatchExp{
					newContentMatchExp("id1", true, ".go", "todo"),
					newContentMatchExp("id2", true, ".go|.txt", "test"),
					newContentMatchExp("id3", false, ".db", "secret"),
					newContentMatchExp("id4", false, "(?i).go|.txt", "(?i)todo"),
					newContentMatchExp("id5", false, "(?i)).go|.txt", "(?i)todo"),  // error in file filter but disabled
					newContentMatchExp("id6", true, "(?i)).go|.txt", "(?i)todo"),   // error in file filter but enabled - count
					newContentMatchExp("id7", true, "(?i).go|.txt", "(?i))todo"),   // error in content filter - count
					newContentMatchExp("id8", false, "(?i)).go|.txt", "(?i))todo"), // error in both but file filter disabled - count
					newContentMatchExp("id9", true, "(?i)).go|.txt", "(?i))todo"),  // error in both - count 2
				},
			},
			matchersLen: 5,
			errorsLen:   5,
		},
		{
			name: "nil exps",
			args: args{
				exps: nil,
			},
			matchersLen: 0,
			errorsLen:   0,
		},
		{
			name: "empty exps",
			args: args{
				exps: []ContentMatchExp{},
			},
			matchersLen: 0,
			errorsLen:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matchers, errs := buildContentMatchers(tt.args.exps)
			if len(matchers) != tt.matchersLen {
				t.Errorf("buildContentMatchers() matchers = %v, want %v", len(matchers), tt.matchersLen)
			}
			if len(errs) != tt.errorsLen {
				t.Errorf("buildContentMatchers() errs = %v, want %v", len(errs), tt.errorsLen)
			}
		})
	}
}

func newFileMatchExp(id string, exp string) FileMatchExp {
	return FileMatchExp{
		ID:  id,
		Exp: exp,
	}
}

func newContentMatchExp(id string, fileFilter bool, fileFilterExp string, exp string) ContentMatchExp {
	return ContentMatchExp{
		ID:                id,
		FileFilterEnabled: fileFilter,
		FileMatchExp: FileMatchExp{
			Exp: fileFilterExp,
		},
		Exp: exp,
	}
}
