package util

import (
	"net"
	"strings"
)

// NormalizeHost lowercases a host and removes any port and trailing dot, so
// "WWW.IMDb.com.:443" becomes "www.imdb.com".
func NormalizeHost(host string) string {
	h := strings.ToLower(strings.TrimSpace(host))
	if hh, _, err := net.SplitHostPort(h); err == nil {
		h = hh
	}
	return strings.TrimSuffix(h, ".")
}

// HostMatchesDomain reports if host is exactly domain or a real subdomain of
// it (ends in "." + domain), after normalizing with NormalizeHost.
//
// Lookalikes are rejected: for domain "imdb.com", "fakeimdb.com" and
// "imdb.com.evil.net" don't match. Hosts with empty labels (".imdb.com",
// "www..imdb.com") don't match either.
//
// Use this for every host check on user supplied urls (url parsing and the
// outbound fetch allowlist), never strings.HasSuffix on the raw host.
func HostMatchesDomain(host string, domain string) bool {
	h := NormalizeHost(host)
	d := strings.ToLower(domain)
	if h == "" || d == "" || strings.HasPrefix(h, ".") || strings.Contains(h, "..") {
		return false
	}
	return h == d || strings.HasSuffix(h, "."+d)
}
