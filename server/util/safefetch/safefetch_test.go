package safefetch

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

var allowed = []string{"letterboxd.com", "boxd.it", "rottentomatoes.com"}

func TestAllowed(t *testing.T) {
	f := New(allowed, nil)
	cases := map[string]bool{
		"https://letterboxd.com/film/x/":             true,
		"https://www.rottentomatoes.com/m/x":         true,
		"https://boxd.it/abc":                        true,
		"https://LETTERBOXD.COM/film/x/":             true,
		"https://letterboxd.com:443/film/x/":         true,
		"http://letterboxd.com/film/x/":              false, // not https
		"https://letterboxd.com:8443/film/x/":        false, // other port
		"https://user:pw@letterboxd.com/film/x/":     false, // userinfo
		"https://notletterboxd.com/film/x/":          false, // lookalike
		"https://letterboxd.com.evil.net/film/x/":    false, // suffixed
		"https://evil.net/?u=https://letterboxd.com": false,
		"https://127.0.0.1/":                         false,
		"https://169.254.169.254/latest/meta-data/":  false,
		"https://localhost/":                         false,
		"file:///etc/passwd":                         false,
		"ftp://letterboxd.com/":                      false,
	}
	for raw, want := range cases {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if got := f.Allowed(u); got != want {
			t.Errorf("Allowed(%s) = %v, want %v", raw, got, want)
		}
	}
}

func newFake(t *testing.T, h http.HandlerFunc) *Fetcher {
	t.Helper()
	s := NewFakeSites(h)
	t.Cleanup(s.Close)
	return New(allowed, s.Transport())
}

func TestGetOK(t *testing.T) {
	var gotUA, gotHost string
	f := newFake(t, func(w http.ResponseWriter, r *http.Request) {
		gotUA, gotHost = r.UserAgent(), r.Host
		w.Write([]byte("<html>ok</html>"))
	})
	body, final, err := f.Get(context.Background(), "https://letterboxd.com/film/fight-club/")
	if err != nil || string(body) != "<html>ok</html>" {
		t.Fatalf("Get: %q %v", body, err)
	}
	if final.Host != "letterboxd.com" || gotHost != "letterboxd.com" {
		t.Errorf("host: final=%s seen=%s", final.Host, gotHost)
	}
	if !strings.Contains(gotUA, "Mozilla") {
		t.Errorf("user agent not a browser one: %q", gotUA)
	}
}

func TestGetRejectsDisallowedWithoutRequest(t *testing.T) {
	called := false
	f := newFake(t, func(w http.ResponseWriter, r *http.Request) { called = true })
	for _, u := range []string{"https://evil.net/", "http://letterboxd.com/", "https://fakeletterboxd.com/", "not a url"} {
		if _, _, err := f.Get(context.Background(), u); !errors.Is(err, ErrNotAllowed) {
			t.Errorf("%s: err = %v, want ErrNotAllowed", u, err)
		}
	}
	if called {
		t.Fatal("a disallowed url reached the network")
	}
}

func TestRedirectsAreChecked(t *testing.T) {
	f := newFake(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Host + r.URL.Path {
		case "boxd.it/ok":
			http.Redirect(w, r, "https://letterboxd.com/film/fight-club/", http.StatusFound)
		case "boxd.it/evil":
			http.Redirect(w, r, "https://evil.net/steal", http.StatusFound)
		case "boxd.it/http":
			http.Redirect(w, r, "http://letterboxd.com/film/x/", http.StatusFound)
		case "boxd.it/loop":
			http.Redirect(w, r, "https://boxd.it/loop", http.StatusFound)
		case "letterboxd.com/film/fight-club/":
			w.Write([]byte("film page"))
		case "evil.net/steal":
			t.Error("redirect to a disallowed host was followed")
		default:
			http.NotFound(w, r)
		}
	})
	body, final, err := f.Get(context.Background(), "https://boxd.it/ok")
	if err != nil || string(body) != "film page" || final.Host != "letterboxd.com" {
		t.Fatalf("allowed redirect: %q %v %v", body, final, err)
	}
	for _, p := range []string{"evil", "http"} {
		if _, _, err := f.Get(context.Background(), "https://boxd.it/"+p); !errors.Is(err, ErrNotAllowed) {
			t.Errorf("redirect %s: err = %v, want ErrNotAllowed", p, err)
		}
	}
	if _, _, err := f.Get(context.Background(), "https://boxd.it/loop"); !errors.Is(err, ErrTooManyRedirects) {
		t.Errorf("redirect loop: err = %v, want ErrTooManyRedirects", err)
	}
}

func TestStatusesAndBodyCap(t *testing.T) {
	f := newFake(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/blocked":
			w.WriteHeader(http.StatusForbidden)
		case "/missing":
			w.WriteHeader(http.StatusNotFound)
		case "/huge":
			w.Write([]byte(strings.Repeat("a", DefaultMaxBodyBytes+10)))
		case "/limit":
			w.Write([]byte(strings.Repeat("a", DefaultMaxBodyBytes)))
		}
	})
	ctx := context.Background()
	if _, _, err := f.Get(ctx, "https://letterboxd.com/blocked"); !errors.Is(err, ErrBlocked) {
		t.Errorf("403: %v", err)
	}
	if _, _, err := f.Get(ctx, "https://letterboxd.com/missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("404: %v", err)
	}
	if _, _, err := f.Get(ctx, "https://letterboxd.com/huge"); !errors.Is(err, ErrTooLarge) {
		t.Errorf("huge: %v", err)
	}
	if b, _, err := f.Get(ctx, "https://letterboxd.com/limit"); err != nil || len(b) != DefaultMaxBodyBytes {
		t.Errorf("exactly the cap should be fine: %d %v", len(b), err)
	}
}
