package ssk

import "testing"

func TestEnvListenAddr(t *testing.T) {
	tests := []struct {
		name string
		env  Env
		want string
	}{
		{"default", Env{}, ephemeralServerAddr},
		{"server-only", Env{ServerOnly: true}, DefaultServerAddr},
		{"explicit", Env{ServerAddr: "127.0.0.1:8201"}, "127.0.0.1:8201"},
		{"explicit server-only", Env{ServerAddr: "127.0.0.1:8201", ServerOnly: true}, "127.0.0.1:8201"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.env.listenAddr(); got != tt.want {
				t.Errorf("listenAddr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClientHost(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"", "127.0.0.1"},
		{"0.0.0.0", "127.0.0.1"},
		{"::", "::1"},
		{"127.0.0.1", "127.0.0.1"},
		{"192.168.0.1", "192.168.0.1"},
		{"localhost", "localhost"},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := clientHost(tt.host); got != tt.want {
				t.Errorf("clientHost(%q) = %q, want %q", tt.host, got, tt.want)
			}
		})
	}
}
