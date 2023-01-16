package main

import (
	"reflect"
	"testing"
)

func Test_isHex(t *testing.T) {
	type args struct {
		hex string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "6 hex chars",
			args: args{hex: "#FFFFFF"},
			want: true,
		},
		{
			name: "8 hex chars with mixed case",
			args: args{hex: "#ffFfffff"},
			want: true,
		},
		{
			name: "8 hex chars no #",
			args: args{hex: "ffffffff"},
			want: false,
		},
		{
			name: "7 hex chars",
			args: args{hex: "#FFFFFFF"},
			want: false,
		},
		{
			name: "6 hex chars no #",
			args: args{hex: "FFFFFF"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHex(tt.args.hex); got != tt.want {
				t.Errorf("isHex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hex2rgba(t *testing.T) {
	type args struct {
		hex string
	}
	tests := []struct {
		name    string
		args    args
		want    *RGBA
		wantErr bool
	}{
		{
			name: "solid white with hex",
			args: args{hex: "#ffffffff"},
			want: &RGBA{
				Red:   255,
				Green: 255,
				Blue:  255,
				Alpha: 1,
			},
			wantErr: false,
		},
		{
			name: "solid white without hex",
			args: args{hex: "#ffffff"},
			want: &RGBA{
				Red:   255,
				Green: 255,
				Blue:  255,
				Alpha: 1,
			},
			wantErr: false,
		},
		{
			name: "transparent white",
			args: args{hex: "#ffffff00"},
			want: &RGBA{
				Red:   255,
				Green: 255,
				Blue:  255,
				Alpha: 0,
			},
			wantErr: false,
		},
		{
			name: "random color with easily divisible hex",
			args: args{hex: "#43ff6480"},
			want: &RGBA{
				Red:   67,
				Green: 255,
				Blue:  100,
				Alpha: float64(128) / float64(255),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hex2rgba(tt.args.hex)
			if (err != nil) != tt.wantErr {
				t.Errorf("hex2rgba() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("hex2rgba() = %v, want %v", got, tt.want)
			}
		})
	}
}
