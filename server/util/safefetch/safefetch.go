// Package safefetch fetches pages from a small allowlist of sites (for
// resolving pasted urls), without letting user input reach any other host.
//
//   - https only, default port only, no userinfo.
//   - The host must be exactly an allowed domain or a real subdomain of it
//     (util.HostMatchesDomain), checked again after every redirect.
//   - At most 3 redirects, a 5s timeout and a 2MB body cap.
package safefetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/sbondCo/Watcharr/util"
)

const (
	DefaultTimeout      = 5 * time.Second
	DefaultMaxBodyBytes = 2 << 20 // 2MB
	MaxRedirects        = 3
	// A normal browser user agent, some sites refuse obvious bots.
	UserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36"
)

var (
	// The url (or a redirect) points at a host/scheme we don't allow.
	ErrNotAllowed = errors.New("url is not allowed")
	// The site refused the request or answered with an error.
	ErrBlocked = errors.New("site blocked the request")
	// The page doesn't exist.
	ErrNotFound = errors.New("page not found")
	// Too many redirects.
	ErrTooManyRedirects = errors.New("too many redirects")
	// The body was bigger than the cap.
	ErrTooLarge = errors.New("response too large")
)

type Fetcher struct {
	client       *http.Client
	allowed      []string
	maxBodyBytes int64
}

// New creates a Fetcher allowing only the given domains (and subdomains).
// transport may be nil to use the default one (tests pass their own).
func New(allowed []string, transport http.RoundTripper) *Fetcher {
	f := &Fetcher{allowed: allowed, maxBodyBytes: DefaultMaxBodyBytes}
	if transport == nil {
		transport = http.DefaultTransport
	}
	f.client = &http.Client{
		Timeout:   DefaultTimeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > MaxRedirects {
				return ErrTooManyRedirects
			}
			if !f.Allowed(req.URL) {
				return fmt.Errorf("%w: redirect to %s", ErrNotAllowed, req.URL.Host)
			}
			return nil
		},
	}
	return f
}

// Allowed reports if u may be fetched.
func (f *Fetcher) Allowed(u *url.URL) bool {
	if u == nil || u.Scheme != "https" || u.User != nil {
		return false
	}
	if p := u.Port(); p != "" && p != "443" {
		return false
	}
	for _, d := range f.allowed {
		if util.HostMatchesDomain(u.Hostname(), d) {
			return true
		}
	}
	return false
}

// Get fetches rawURL and returns the body and the final url (after
// redirects).
func (f *Fetcher) Get(ctx context.Context, rawURL string) ([]byte, *url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil || !f.Allowed(u) {
		return nil, nil, ErrNotAllowed
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, ErrNotAllowed
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	res, err := f.client.Do(req)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotAllowed):
			return nil, nil, ErrNotAllowed
		case errors.Is(err, ErrTooManyRedirects):
			return nil, nil, ErrTooManyRedirects
		}
		return nil, nil, fmt.Errorf("%w: %v", ErrBlocked, err)
	}
	defer res.Body.Close()

	switch {
	case res.StatusCode == http.StatusNotFound:
		return nil, res.Request.URL, ErrNotFound
	case res.StatusCode < 200 || res.StatusCode > 299:
		return nil, res.Request.URL, fmt.Errorf("%w: status %d", ErrBlocked, res.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, f.maxBodyBytes+1))
	if err != nil {
		return nil, res.Request.URL, fmt.Errorf("%w: %v", ErrBlocked, err)
	}
	if int64(len(body)) > f.maxBodyBytes {
		return nil, res.Request.URL, ErrTooLarge
	}
	return body, res.Request.URL, nil
}
