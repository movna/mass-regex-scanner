package mres

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/movna/mres/internal"
)

func getTestScanner(t *testing.T, folders []string) *Scanner {
	exps := Expressions{}
	scanner, err := NewScanner(exps)
	if err != nil {
		t.Errorf("error while setting up scanner. err: %v", err)
		t.FailNow()
		return nil
	}
	scanner.SetLogger(internal.NewDefaultLogger())
	return scanner
}

func TestScanner_Scan(t *testing.T) {
	type args struct {
		ctx     context.Context
		folders []string
	}
	tests := []struct {
		name  string
		args  args
		want  []Result
		want1 []error
	}{
		{
			name:  "just try",
			args:  args{ctx: context.TODO(), folders: []string{"./testdatazz"}},
			want:  []Result{},
			want1: []error{errors.New("lstat ./testdatazz: no such file or directory")}, // TODO: better way to assert
		},
		{
			name:  "just try 2",
			args:  args{ctx: context.TODO(), folders: []string{"./testdata"}},
			want:  []Result{},
			want1: []error{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := getTestScanner(t, tt.args.folders)
			got, got1 := s.Scan(tt.args.ctx, []string{}, 1)
			if !reflect.DeepEqual(got, tt.want) { // TODO: better way to assert
				t.Errorf("Scanner.Scan() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Scanner.Scan() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
