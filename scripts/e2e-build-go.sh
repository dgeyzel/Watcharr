#!/bin/sh
# Builds the Go binaries used by the e2e tests: the server, the seed command
# and the TMDB fixture stub. Output goes to $E2E_BIN (default server/.e2e-bin).
#
# Set E2E_STATIC=1 to link statically (needed when building on Alpine/musl
# and running the tests in a glibc image, e.g. the Playwright docker image).
set -eu

cd "$(dirname "$0")/.."
out="${E2E_BIN:-server/.e2e-bin}"
mkdir -p "$out"
out="$(cd "$out" && pwd)"

export CGO_ENABLED=1
export CGO_CFLAGS="${CGO_CFLAGS:--D_LARGEFILE64_SOURCE}"

if [ "${E2E_STATIC:-}" = "1" ]; then
	set -- -ldflags '-linkmode external -extldflags "-static"'
else
	set --
fi

go -C server build "$@" -o "$out/watcharr" .
go -C server build "$@" -o "$out/seed" ./cmd/seed
go -C server build "$@" -o "$out/tmdbstub" ./cmd/tmdbstub
echo "e2e binaries built in $out"
