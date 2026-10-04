package main

import (
	"reflect"
	"testing"
)

func TestTuiArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    []string
		wantErr bool
	}{
		{name: "no command", args: nil, want: nil},
		{name: "tui command", args: []string{"tui", "-no-sync"}, want: []string{"-no-sync"}},
		{name: "legacy flags", args: []string{"-no-sync"}, want: []string{"-no-sync"}},
		{name: "unknown command", args: []string{"sync"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tuiArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("tuiArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("tuiArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
