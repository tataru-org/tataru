package main

import "testing"

func Test_isGmailEmailAddress(t *testing.T) {
	type args struct {
		email string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "acceptable gmail",
			args: args{email: "hello_world@gmail.com"},
			want: true,
		},
		{
			name: "empty string before @ symbol",
			args: args{email: "@gmail.com"},
			want: false,
		},
		{
			name: "is not a gmail address",
			args: args{email: "hello_world@hotmail.com"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGmailEmailAddress(tt.args.email); got != tt.want {
				t.Errorf("isGmailEmailAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}
