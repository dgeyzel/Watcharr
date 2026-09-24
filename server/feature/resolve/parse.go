package resolve

import (
	"encoding/json"
	"html"
	"regexp"
	"strconv"
	"strings"
)

// Page parsers. Sites change their markup, so these are small, isolated and
// covered by saved html fixtures (server/testdata/resolve).

var (
	lbTmdbID   = regexp.MustCompile(`data-tmdb-id="(\d+)"`)
	lbTmdbType = regexp.MustCompile(`data-tmdb-type="(movie|tv)"`)
	lbTmdbLink = regexp.MustCompile(`themoviedb\.org/(movie|tv)/(\d+)`)

	jsonLD  = regexp.MustCompile(`(?is)<script[^>]+type=["']application/ld\+json["'][^>]*>(.*?)</script>`)
	ogTitle = regexp.MustCompile(`(?is)<meta[^>]+property=["']og:title["'][^>]+content=["']([^"']*)["']`)
	// Title (1982) / Title (2008 - 2013)
	titleYear = regexp.MustCompile(`^(.*?)\s*\((\d{4})[^)]*\)\s*$`)
	slugYear  = regexp.MustCompile(`^(.*?)_(\d{4})$`)
)

// ParseLetterboxd finds the TMDB id (and type) on a Letterboxd film page.
func ParseLetterboxd(body string) (mediaType string, tmdbID int, ok bool) {
	if m := lbTmdbID.FindStringSubmatch(body); m != nil {
		id, _ := strconv.Atoi(m[1])
		mediaType = "movie"
		if t := lbTmdbType.FindStringSubmatch(body); t != nil {
			mediaType = t[1]
		}
		return mediaType, id, id > 0
	}
	if m := lbTmdbLink.FindStringSubmatch(body); m != nil {
		id, _ := strconv.Atoi(m[2])
		return m[1], id, id > 0
	}
	return "", 0, false
}

// TitleYear is a title and (optional, 0 = unknown) year.
type TitleYear struct {
	Title string
	Year  int
}

// ParseRottenTomatoes reads the title and year from a Rotten Tomatoes page
// (JSON-LD first, then og:title).
func ParseRottenTomatoes(body string) (TitleYear, bool) {
	for _, m := range jsonLD.FindAllStringSubmatch(body, -1) {
		if ty, ok := titleYearFromJSONLD(m[1]); ok {
			return ty, true
		}
	}
	if m := ogTitle.FindStringSubmatch(body); m != nil {
		t := html.UnescapeString(strings.TrimSpace(m[1]))
		// "Title | Rotten Tomatoes"
		if i := strings.Index(t, " | "); i > 0 {
			t = t[:i]
		}
		if tm := titleYear.FindStringSubmatch(t); tm != nil {
			y, _ := strconv.Atoi(tm[2])
			return TitleYear{Title: strings.TrimSpace(tm[1]), Year: y}, true
		}
		if t != "" {
			return TitleYear{Title: t}, true
		}
	}
	return TitleYear{}, false
}

func titleYearFromJSONLD(raw string) (TitleYear, bool) {
	var v any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &v); err != nil {
		return TitleYear{}, false
	}
	// A single object, a list, or {"@graph": [...]}.
	var objs []map[string]any
	switch t := v.(type) {
	case map[string]any:
		if g, ok := t["@graph"].([]any); ok {
			for _, o := range g {
				if m, ok := o.(map[string]any); ok {
					objs = append(objs, m)
				}
			}
		} else {
			objs = append(objs, t)
		}
	case []any:
		for _, o := range t {
			if m, ok := o.(map[string]any); ok {
				objs = append(objs, m)
			}
		}
	}
	for _, o := range objs {
		typ, _ := o["@type"].(string)
		if typ != "Movie" && typ != "TVSeries" {
			continue
		}
		name, _ := o["name"].(string)
		name = html.UnescapeString(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		ty := TitleYear{Title: name}
		for _, k := range []string{"dateCreated", "datePublished", "startDate", "releasedEvent"} {
			if s, ok := o[k].(string); ok && len(s) >= 4 {
				if y, err := strconv.Atoi(s[:4]); err == nil {
					ty.Year = y
					break
				}
			}
		}
		return ty, true
	}
	return TitleYear{}, false
}

// TitleYearFromRTSlug is the fallback when the page can't be read:
// "the_thing_1982" -> "the thing", 1982.
func TitleYearFromRTSlug(slug string) TitleYear {
	ty := TitleYear{Title: slug}
	if m := slugYear.FindStringSubmatch(slug); m != nil {
		ty.Title = m[1]
		ty.Year, _ = strconv.Atoi(m[2])
	}
	ty.Title = strings.TrimSpace(strings.NewReplacer("_", " ", "-", " ").Replace(ty.Title))
	return ty
}
