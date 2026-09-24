package util

import "testing"

func TestHostMatchesDomain(t *testing.T) {
	domains := []string{"imdb.com", "themoviedb.org", "letterboxd.com", "boxd.it", "rottentomatoes.com"}
	cases := []struct {
		name   string
		host   string
		domain string
		want   bool
	}{
		// Exact hosts.
		{"exact imdb", "imdb.com", "imdb.com", true},
		{"exact tmdb", "themoviedb.org", "themoviedb.org", true},
		{"exact letterboxd", "letterboxd.com", "letterboxd.com", true},
		{"exact boxd", "boxd.it", "boxd.it", true},
		{"exact rt", "rottentomatoes.com", "rottentomatoes.com", true},
		// Real subdomains.
		{"www", "www.imdb.com", "imdb.com", true},
		{"mobile", "m.imdb.com", "imdb.com", true},
		{"www tmdb", "www.themoviedb.org", "themoviedb.org", true},
		{"deep subdomain", "a.b.letterboxd.com", "letterboxd.com", true},
		// Lookalike prefixes.
		{"fake prefix", "fakeimdb.com", "imdb.com", false},
		{"not prefix", "notletterboxd.com", "letterboxd.com", false},
		{"dash prefix", "my-themoviedb.org", "themoviedb.org", false},
		{"prefix boxd", "xboxd.it", "boxd.it", false},
		// Suffixed hosts.
		{"suffixed", "imdb.com.evil.net", "imdb.com", false},
		{"suffixed tld", "letterboxd.com.co", "letterboxd.com", false},
		{"suffixed rt", "rottentomatoes.com-evil.com", "rottentomatoes.com", false},
		// Uppercase hosts.
		{"upper exact", "IMDB.COM", "imdb.com", true},
		{"upper sub", "WWW.ImDb.Com", "imdb.com", true},
		{"upper fake", "FAKEIMDB.COM", "imdb.com", false},
		// Hosts with ports.
		{"port", "imdb.com:443", "imdb.com", true},
		{"port sub", "www.imdb.com:8080", "imdb.com", true},
		{"port fake", "fakeimdb.com:443", "imdb.com", false},
		{"port suffixed", "imdb.com.evil.net:443", "imdb.com", false},
		// Trailing dot (fully qualified).
		{"trailing dot", "imdb.com.", "imdb.com", true},
		{"trailing dot port", "www.imdb.com.:443", "imdb.com", true},
		// Malformed / empty.
		{"empty", "", "imdb.com", false},
		{"leading dot", ".imdb.com", "imdb.com", false},
		{"empty label", "www..imdb.com", "imdb.com", false},
		{"only dot", ".", "imdb.com", false},
		{"ipv4", "93.184.216.34", "imdb.com", false},
		{"other domain", "letterboxd.com", "imdb.com", false},
	}
	for _, c := range cases {
		if got := HostMatchesDomain(c.host, c.domain); got != c.want {
			t.Errorf("%s: HostMatchesDomain(%q, %q) = %v, want %v", c.name, c.host, c.domain, got, c.want)
		}
	}
	// Every supported domain accepts itself and www., rejects lookalikes.
	for _, d := range domains {
		for host, want := range map[string]bool{
			d: true, "www." + d: true, "m." + d: true,
			"fake" + d: false, d + ".evil.net": false, "not" + d + ":443": false,
		} {
			if got := HostMatchesDomain(host, d); got != want {
				t.Errorf("HostMatchesDomain(%q, %q) = %v, want %v", host, d, got, want)
			}
		}
	}
}

func TestNormalizeHost(t *testing.T) {
	for in, want := range map[string]string{
		"WWW.IMDb.com.:443": "www.imdb.com",
		"imdb.com":          "imdb.com",
		" IMDB.com ":        "imdb.com",
		"imdb.com:":         "imdb.com",
	} {
		if got := NormalizeHost(in); got != want {
			t.Errorf("NormalizeHost(%q) = %q, want %q", in, got, want)
		}
	}
}
