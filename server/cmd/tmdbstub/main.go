// tmdbstub runs a stub TMDB API server that serves JSON fixtures. It is used
// by the e2e tests so they never depend on the real TMDB API.
//
//	go run ./cmd/tmdbstub -addr 127.0.0.1:3099
//
// Point the Watcharr server at it with TMDB_API_BASE=http://127.0.0.1:3099
// and TMDB_IMAGE_BASE=http://127.0.0.1:3099/t/p.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/sbondCo/Watcharr/internal/testutil/tmdbstub"
)

func main() {
	defaultDir := os.Getenv("TMDB_FIXTURES_DIR")
	if defaultDir == "" {
		defaultDir = tmdbstub.FixturesDir()
	}
	addr := flag.String("addr", "127.0.0.1:3099", "address to listen on")
	dir := flag.String("dir", defaultDir, "fixtures directory (env TMDB_FIXTURES_DIR)")
	flag.Parse()

	if _, err := os.Stat(*dir); err != nil {
		log.Fatalf("tmdbstub: fixtures dir %q not usable: %v", *dir, err)
	}
	log.Printf("tmdbstub: serving %s on http://%s", *dir, *addr)
	log.Fatal(http.ListenAndServe(*addr, tmdbstub.Handler(*dir)))
}
