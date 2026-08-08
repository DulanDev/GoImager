package service

import (
	"net"
	"testing"
)

func TestIsBlockedIP(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"nil", "<nil>", false},
		{"public v4", "8.8.8.8", false},
		{"public v6", "2001:4860:4860::8888", false},
		{"loopback v4", "127.0.0.1", true},
		{"loopback v6", "::1", true},
		{"private 10", "10.0.0.1", true},
		{"private 172.16", "172.16.0.1", true},
		{"private 192.168", "192.168.1.1", true},
		{"link-local 169.254", "169.254.169.254", true}, // cloud metadata endpoint
		{"this-host 0.0.0.0", "0.0.0.0", true},
		{"cgnat 100.64", "100.64.0.1", true},
		{"ipv6 ula", "fd00::1", true},
		{"ipv6 link-local", "fe80::1", true},
		{"ipv6 unspecified", "::", true},
		{"v4-mapped metadata", "::ffff:169.254.169.254", true},
		{"v4-mapped loopback", "::ffff:127.0.0.1", true},
		{"v4-mapped public", "::ffff:8.8.8.8", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var ip net.IP
			if c.in != "<nil>" {
				ip = net.ParseIP(c.in)
				if ip == nil {
					t.Fatalf("ParseIP(%q) = nil", c.in)
				}
			}
			if got := isBlockedIP(ip); got != c.want {
				t.Errorf("isBlockedIP(%s) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
