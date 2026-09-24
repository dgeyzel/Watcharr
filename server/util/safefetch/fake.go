package safefetch

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
)

// FakeSites serves every host from one local TLS test server, so code using a
// Fetcher can be tested without the network. The handler sees the requested
// host in r.Host. Only for tests.
type FakeSites struct {
	Server *httptest.Server
}

func NewFakeSites(h http.Handler) *FakeSites {
	return &FakeSites{Server: httptest.NewTLSServer(h)}
}

// Transport routes every connection to the fake server (skipping cert
// verification, the fake cert isn't for the real host names).
func (s *FakeSites) Transport() http.RoundTripper {
	addr := s.Server.Listener.Addr().String()
	return &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
}

func (s *FakeSites) Close() { s.Server.Close() }
