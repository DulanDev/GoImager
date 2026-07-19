package config

import "net/url"

// IsAllowed reports whether host is permitted by the allowlist.
// A wildcard entry "*" allows any host.
func (c Config) IsAllowed(host string) bool {
	for _, a := range c.Allowed {
		if a == "*" {
			return true
		}
		if a == host {
			return true
		}
	}
	return false
}

// IsAllowedURL checks the parsed URL's host against the allowlist.
func (c Config) IsAllowedURL(u *url.URL) bool {
	if u == nil {
		return false
	}
	return c.IsAllowed(u.Hostname())
}