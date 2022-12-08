package main

import (
	"reflect"
	"testing"
)

func TestIsInSlice(t *testing.T) {
	type args struct {
		itm   string
		slice []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"empty", args{"", []string{}}, false},
		{"yes", args{"a", []string{"b", "a", "p"}}, true},
		{"no", args{"a", []string{"b", "p"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInSlice(tt.args.itm, tt.args.slice); got != tt.want {
				t.Errorf("IsInSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSlicesEqual(t *testing.T) {
	type args struct {
		a []string
		b []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"empty", args{[]string{}, []string{}}, true},
		{"yes", args{[]string{"f", "w", "t"}, []string{"t", "f", "w"}}, true},
		{"no", args{[]string{"f", "w", "t"}, []string{"ph", "w", "t"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSlicesEqual(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("IsSlicesEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseSpeed(t *testing.T) {
	type args struct {
		tcomment string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"empty", args{""}, 0},
		{"ok15", args{"патриот города - 40 [15/200]"}, 15},
		{"ok50", args{"патриот города - 40 [ 50/200]"}, 50},
		{"ok4", args{"патриот города - 40 [ 4 / 44]"}, 4},
		{"err", args{" 4 / 44]"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSpeed(tt.args.tcomment)
			if got != tt.want {
				t.Errorf("parseSpeed() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseIps(t *testing.T) {
	type args struct {
		ips string
	}
	tests := []struct {
		name    string
		args    args
		wantRes []string
	}{
		{"empty", args{""}, nil},
		{"ok", args{"10.10.0.1, 192.168.0.10 and 172.16.28.244"},
			[]string{"10.10.0.1", "172.16.28.244", "192.168.0.10"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotRes := parseIps(tt.args.ips); !reflect.DeepEqual(gotRes, tt.wantRes) {
				t.Errorf("parseIps() = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

func Test_parseSpeedFromQueueLimit(t *testing.T) {
	type args struct {
		queueValue string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"empty", args{""}, 0},
		{"ok40", args{"40000000/"}, 40},
		{"ok50", args{"50000000/200]"}, 50},
		{"err0", args{"[ 4000000 / 44]"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSpeedFromQueueLimit(tt.args.queueValue)
			if got != tt.want {
				t.Errorf("parseSpeedFromQueueLimit() got = %v, want %v", got, tt.want)
			}
		})
	}
}
