package main

import (
	"testing"
)

func TestQRec_CalcSum(t *testing.T) {
	tests := []struct {
		name     string
		rec      *QRec
		wantHash string
	}{
		{"empty", &QRec{}, "d228cb69101a8caf78912b704e4a144f"},
		{"ok", &QRec{Name: "User Name"}, "dffd2d35fb8c6daf1449c251e4963f27"},
		{"ok", &QRec{Name: "User Name", Speed: 40}, "ee6b37aba8caf66df6c21ac444dfb021"},
		{"ok", &QRec{Name: "User Name", Speed: 40, Comment: "some comment"},
			"67882cfad286235b78677a36cf49a9b2"},
		{"ok", &QRec{Name: "User Name", Speed: 40, Comment: "some comment",
			Target: []string{"10.0.0.10"}}, "c08e8ea93d3073e7ad50e864959ae18e"},
		{"ok", &QRec{Name: "User Name", Speed: 40, Comment: "some comment",
			Target: []string{"10.0.0.10", "172.16.25.24"}}, "1759fa6c3bb559a87c21e3fca3c9c138"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rec.CalcSum()
			if tt.rec.Hash != tt.wantHash {
				t.Errorf("CalcSum() got hash = %v, want %v", tt.rec.Hash, tt.wantHash)
			}
		})
	}
}
