package main

import (
	"testing"
)

func Test_formatDsn(t *testing.T) {
	type args struct {
		hostname string
		username string
		password string
		dbname   string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 string
	}{
		{"empty", args{"", "", "", ""},
			"mysql", ":@tcp(:3306)/"},
		{"localhost", args{"localhost", "user", "passwd", "db"},
			"mysql", "user:passwd@/db"},
		{"remote", args{"example.com", "user", "passwd", "db"},
			"mysql", "user:passwd@tcp(example.com:3306)/db"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := formatDsn(tt.args.hostname, tt.args.username, tt.args.password, tt.args.dbname)
			if got != tt.want {
				t.Errorf("formatDsn() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("formatDsn() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

//goland:noinspection SpellCheckingInspection
func TestDbRec_calcSum(t *testing.T) {
	tests := []struct {
		name     string
		rec      *DbRec
		wantHash string
	}{
		{"empty", &DbRec{}, "d228cb69101a8caf78912b704e4a144f"},
		{"ok", &DbRec{Name: "User Name"}, "dffd2d35fb8c6daf1449c251e4963f27"},
		{"ok", &DbRec{Name: "User Name", Speed: 40}, "ee6b37aba8caf66df6c21ac444dfb021"},
		{"ok", &DbRec{Name: "User Name", Speed: 40, Comment: "some comment"},
			"67882cfad286235b78677a36cf49a9b2"},
		{"ok", &DbRec{Name: "User Name", Speed: 40, Comment: "some comment",
			Ips: []string{"10.0.0.10"}}, "c08e8ea93d3073e7ad50e864959ae18e"},
		{"ok", &DbRec{Name: "User Name", Speed: 40, Comment: "some comment",
			Ips: []string{"10.0.0.10", "172.16.25.24"}}, "1759fa6c3bb559a87c21e3fca3c9c138"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rec.calcSum()
			if tt.rec.Hash != tt.wantHash {
				t.Errorf("calcSum() got hash = %v, want %v", tt.rec.Hash, tt.wantHash)
			}
		})
	}
}
