package service

import (
	"context"
	"errors"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/DulanDev/GoImager/internal/config"
)

// errBlockedIP is returned by the SSRF-aware dialer when the resolved
// destination IP falls into a private/reserved range. The message is
// intentionally generic so it never leaks the resolved address.
var errBlockedIP = errors.New("destination IP is in a blocked private/reserved range")

// blockedCIDRs enumerates networks /process must not reach even when the
// host appears in ALLOWED_DOMAINS. Covers RFC 1918 private space, the
// AWS/Azure/GCP link-local metadata endpoints (169.254.0.0/16), loopback,
// "this" host (0.0.0.0/8), IPv6 loopback/ULA/link-local and unspecified.
//
// Order does not matter; net.ParseCIDR is cheap.
var blockedCIDRs = func() []*net.IPNet {
	specs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"127.0.0.0/8",
		"0.0.0.0/8",
		"100.64.0.0/10", // CGNAT
		"::1/128",
		"fc00::/7", // IPv6 ULA
		"fe80::/10", // link-local
		"::/128",   // unspecified
	}
	out := make([]*net.IPNet, 0, len(specs))
	for _, s := range specs {
		if _, n, err := net.ParseCIDR(s); err == nil {
			out = append(out, n)
		}
	}
	return out
}()

// isBlockedIP reports whether ip falls in any blocked range. nil or
// unicast public IPs return false.
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, n := range blockedCIDRs {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// safeDialContext rejects destinations that resolve (in whole or in part)
// to a blocked private/reserved range. The check runs in the dialer's
// ControlContext hook, which fires *after* the OS resolves the hostname but
// *before* connect() — so an attacker-controlled DNS server that returns a
// public IP on the first lookup and a metadata IP (169.254.169.254) on the
// second cannot win (classic DNS-rebinding TOCTOU). Dialing the original
// host (not the IP literal) preserves TLS SNI for virtual-hosted origins.
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{
		Timeout: 10 * time.Second,
		ControlContext: func(_ context.Context, _ string, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				host = address
			}
			ip := net.ParseIP(host)
			if ip == nil || isBlockedIP(ip) {
				return errBlockedIP
			}
			return nil
		},
	}
	return d.DialContext(ctx, network, addr)
}

// safeTransport returns an *http.Transport whose DialContext blocks
// private/reserved destinations.
func safeTransport() *http.Transport {
	return &http.Transport{
		DialContext:           safeDialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
}

// NewSafeHTTPClient builds an *http.Client for outbound /process fetches.
// It rejects private/reserved destinations (SSRF mitigation) and
// re-applies the configured allowlist on every redirect so a 302 from an
// allowed host cannot hop to an internal address.
//
// timeout <= 0 falls back to 20s.
func NewSafeHTTPClient(cfg config.Config, timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: safeTransport(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			loc := req.URL
			if loc == nil {
				return errors.New("redirect without Location")
			}
			if !cfg.IsAllowedURL(loc) {
				return errors.New("redirect host not in allowlist")
			}
			if loc.Scheme != "http" && loc.Scheme != "https" {
				return errors.New("redirect scheme not http(s)")
			}
			return nil
		},
	}
}