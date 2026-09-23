# End-to-end tests

Playwright tests that run the built UI against the real Go server. TMDB is
served by a fixture stub (`server/cmd/tmdbstub`, fixtures in
`server/testdata/tmdb`), and the browser aborts every request that leaves
`127.0.0.1`, so no test needs the network.

`playwright.config.ts` starts three processes:

1. the TMDB stub on `127.0.0.1:3099`
2. the built UI (`node build`) on `127.0.0.1:3000`
3. `seed` (fresh data dir in `e2e/.data`, admin `admin` / `e2e-admin-password`,
   plus the fixture list), then the server on `127.0.0.1:3080`, which proxies
   UI requests to port 3000

## Running natively (Linux, CI)

```bash
npm ci
npx playwright install --with-deps chromium
npm run build
sh scripts/e2e-build-go.sh
npm run test:e2e
```

## Running with Docker only

The Go binaries are built statically on Alpine, so they run inside the
glibc-based Playwright image. `node_modules` lives in named volumes, separately
for the Alpine and Playwright images, so nothing platform-specific lands in the
checkout.

```bash
# Go binaries (server, seed, tmdb stub)
docker run --rm -v "$PWD:/app" -w /app golang:1.26-alpine sh -c \
  'apk add --no-cache gcc musl-dev build-base && E2E_STATIC=1 sh scripts/e2e-build-go.sh'

# UI build
docker run --rm -v "$PWD:/app" -v watcharr_nm:/app/node_modules -w /app \
  node:24-alpine sh -c 'npm ci && npm run build'

# Tests (the image tag must match the @playwright/test version)
docker run --rm --ipc=host -v "$PWD:/app" -v watcharr_nm_pw:/app/node_modules -w /app \
  mcr.microsoft.com/playwright:v1.63.0-noble sh -c 'npm ci && npm run test:e2e'
```
