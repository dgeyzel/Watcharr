# Watcharr Review Site

A personal, read-only review site for the movies and TV shows one owner has watched. Visitors browse the owner's list, S to F tiers and written reviews without signing in. The owner signs in at `/admin` and keeps every management feature: adding titles, rating, reviewing, importing and server settings.

## About This Project

This project is a fork of [Watcharr](https://github.com/sbondCo/Watcharr), a self-hosted watched list for movies and TV shows, and is distributed under the same GPL-3.0-only license. Upstream Watcharr is a multi-user watch tracker. This fork turns it into a single-owner site that anyone can read and only the owner can change.

What this fork changes (a notice of modifications, as GPL-3.0 section 5(a) asks):

- **Single owner.** Registration, Plex, Jellyfin and trusted-header login, admin-token promotion and user management are removed. The only account is the admin created at first run, which is protected by a one-time setup token.
- **Public read-only access.** A new public API (`/api/public/*`) serves the owner's visible titles, tags, reviews and stats to visitors with no account. Every other API route requires the admin.
- **Admin login at `/admin`.** `/login` is removed and the public UI has no login link. Logins are rate limited and passwords must be at least 12 characters.
- **Tiers.** A new S, A, B, C, D, F score is stored alongside the existing numeric ratings. Visitors only ever see the tier. Numeric ratings are never converted into tiers, including on import.
- **Hidden titles.** Any title can be hidden from visitors, like a draft.
- **Public statuses.** Finished, Watching and Planned titles are public. On Hold and Dropped are admin-only.
- **Public stats page** at `/stats`.
- **Fixing failed imports by URL.** Paste a TMDB, IMDb, Letterboxd or Rotten Tomatoes link to match a row the import couldn't find. The same links work in search and in "Add by URL".
- **Removed:** following, public profiles and lists, games (IGDB), Sonarr and Radarr requests, season and episode ratings, the documentation site and upstream project files.
- **Added:** a footer credit on every page, and Go, Vitest and Playwright test suites.

## Features

### For visitors

- Browse the owner's list with posters, sorting and filters (status, type, tier)
- A tier badge on every poster, or "not rated yet" for watched titles with no tier
- Title pages with the tier, the written review and TMDB details (cast, overview, runtime)
- Tag pages, and search within the owner's list
- A stats page: totals, titles by status, the tier distribution, titles added per month, release decades, top genres, tags and hours of movies watched
- No account, no sign-in and nothing that can be changed

### For the admin

- Search TMDB and Discover to find titles, and add them with a status
- Set a tier (S to F), a private numeric rating and a written review
- Hide or unhide a title from visitors
- Tag titles and pin favourites
- Import from a text list, TMDB CSV, IMDb CSV, Movary, MyAnimeList, Ryot, TodoMovies, Trakt and Watcharr exports
- Fix failed import rows, or add any title, by pasting a TMDB, IMDb, Letterboxd or Rotten Tomatoes link
- Export the list (including tiers and hidden flags) as a Watcharr JSON file
- Server settings: TMDB key, default country, debug logging and scheduled tasks

## Tech Stack

- **Server:** Go with Gin (HTTP), GORM (ORM) and SQLite (via CGO, `mattn/go-sqlite3`)
- **Web UI:** SvelteKit 2 and Svelte 5, TypeScript and SCSS, served by `@sveltejs/adapter-node`
- **Data:** the TMDB API for title details and posters
- **Packaging:** Docker (multi-stage build) and Docker Compose
- **Testing:** Go `testing`, Vitest with Testing Library (jsdom) and Playwright

## Requirements

- Go 1.26 or newer (see `server/go.mod`)
- Node.js 24 (the version used by the `Dockerfile`)
- gcc, because SQLite is built with CGO
- A TMDB API key (recommended). A shared default key is built in, but your own key avoids its rate limits. Get one from [themoviedb.org](https://www.themoviedb.org/settings/api).

Docker is enough if you only want to run the site: the image builds the server and UI itself.

## Installation

Clone the repository:

```bash
git clone https://github.com/dgeyzel/Watcharr.git
cd Watcharr
```

Install the web dependencies and Go modules for local development:

```bash
npm ci
cd server && go mod download && cd ..
```

## Configuration

The server keeps everything in its data directory: `./data` by default, or the path in the `WATCHARR_DATA` environment variable. In the Docker image this is `/data`.

```
data/
├── watcharr.json    Server config (created on first start)
├── watcharr.db      SQLite database
├── watcharr.log     Log file
└── ...              Cached images and TMDB responses
```

### watcharr.json

On first start the server writes `data/watcharr.json` with a random `JWT_SECRET` and default settings. Edit it while the server is stopped. `TMDB_KEY`, `DEFAULT_COUNTRY` and `DEBUG` can also be changed from the admin's server settings page.

```json
{
	"JWT_SECRET": "generated on first start, keep it long and secret",
	"DEFAULT_COUNTRY": "US",
	"TMDB_KEY": "your TMDB API key",
	"TRUSTED_PROXIES": ["172.18.0.1"],
	"DEBUG": false
}
```

| Key               | Purpose                                                                                                                                 |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| `JWT_SECRET`      | Signs admin login tokens. Required. Changing it signs the admin out.                                                                    |
| `TMDB_KEY`        | Your TMDB API key. Optional; the built-in key is used when empty.                                                                       |
| `DEFAULT_COUNTRY` | Region used for streaming providers (ISO 3166-1 code).                                                                                  |
| `TMDB_API_BASE`   | TMDB API base url. Default `https://api.themoviedb.org/3`.                                                                              |
| `TMDB_IMAGE_BASE` | TMDB image base url. Default `https://image.tmdb.org/t/p`.                                                                              |
| `TRUSTED_PROXIES` | IPs or CIDRs of reverse proxies allowed to set `X-Forwarded-For`, which is used for rate limiting. Leave empty when not behind a proxy. |
| `TASK_SCHEDULE`   | Overrides for how often background tasks run.                                                                                           |
| `DEBUG`           | `true` enables debug logging.                                                                                                           |

Keys that upstream Watcharr used and this fork removed (`SIGNUP_ENABLED`, `JELLYFIN_HOST`, `USE_EMBY`, `PLEX_HOST`, `PLEX_MACHINE_ID`, `HEADER_AUTH`, `SONARR`, `RADARR`, `TWITCH`) are ignored when read, and dropped the next time the config is saved.

### Environment variables

| Variable           | Purpose                                                                                                                           |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------- |
| `WATCHARR_DATA`    | Data directory. Default `./data`.                                                                                                 |
| `TMDB_API_BASE`    | Overrides `TMDB_API_BASE` from the config (used by the tests).                                                                    |
| `TMDB_IMAGE_BASE`  | Overrides `TMDB_IMAGE_BASE` from the config.                                                                                      |
| `MODE`             | `DEV` runs the API only, for use with `npm run dev`.                                                                              |
| `WATCHARR_SKIP_UI` | In production mode, don't start the bundled UI server. The server still proxies to it on port 3000, so run `node build` yourself. |

Variables can also be put in a `.env` file in the directory the server is started from.

### The first-run setup token

While the database has no users, the server prints a one-time setup token to its log at start-up:

```
level=WARN msg="Server is in setup. Use this setup token to create the admin account." setup_token=...
```

The setup page asks for this token, so nobody who finds a fresh server first can make themselves the admin. The token changes on every restart until the admin exists.

### Upgrading from upstream Watcharr

This fork is meant to start from a fresh data directory. If you point it at an existing upstream `data/` directory, back it up first:

```bash
cp -r data data-backup-$(date +%Y%m%d)
```

## Building the Application

### Docker image

```bash
docker build -t watcharr-review .
```

The `Dockerfile` builds the Go server with CGO on Alpine, builds the UI, and produces an image that runs both on port 3080.

### Local build

Build the server (CGO must be enabled):

```bash
cd server
CGO_ENABLED=1 go build -o watcharr .
cd ..
```

On Alpine or other musl systems, add `CGO_CFLAGS="-D_LARGEFILE64_SOURCE"`.

Build the UI:

```bash
npm run build
```

The UI build goes to `build/` and is run with `node build`.

## Running the Application

### Docker Compose

`docker-compose.yml` runs the published image and keeps its data in `./data`:

```bash
docker compose up -d
docker compose logs watcharr | grep setup_token
```

Open `http://localhost:3080/setup`, paste the setup token and create the admin. To run your own build instead, use `docker-compose.dev.yml`, which builds the image from the checkout and serves it on port 3081:

```bash
docker compose -f docker-compose.dev.yml up --build
```

### Local development

Run the API in dev mode in one terminal:

```bash
cd server
MODE=DEV go run .
```

Run the UI dev server in another:

```bash
npm run dev
```

Open the address Vite prints (usually `http://localhost:5173`). In dev mode the UI calls the API on port 3080 of the same host.

### Local production build

With the server and UI built as above, run the UI on port 3000 and the server in front of it:

```bash
node build &
cd server
WATCHARR_SKIP_UI=1 ./watcharr
```

The site is then at `http://localhost:3080`.

## Usage Guide

### 1. First-run setup

Start the server and copy the `setup_token` from its log. Open `/setup`, enter the token, and choose a username and a password of at least 12 characters. This creates the only account.

### 2. Sign in at /admin

Go to `/admin` and sign in. There is no login link on the public site, so bookmark this page. Login attempts are limited to five per minute from one address.

### 3. Add a title

Search for a title with the search bar and open it, or use **Add by URL** on the search page and paste a TMDB, IMDb, Letterboxd or Rotten Tomatoes link. On the title's page, choose a status (Finished, Watching, Planned, On Hold or Dropped) to add it to your list.

### 4. Set a tier and write a review

On the title's page, pick a tier from S to F and write your review. You can also set a numeric rating; only you see it. Watched titles without a tier show "Watched but not yet rated" to visitors. Planned titles show no tier at all.

### 5. Hide or unhide a title

Tick **Hidden from visitors** on the title's page to keep it private while you work on it. Hidden titles are marked on your posters and are missing from every public page, the public API and the stats. Untick it to publish the title.

### 6. Import a list

Open your **Profile** from the account menu, click **Import** and choose a file (text list, TMDB CSV, IMDb CSV, Movary, MyAnimeList, Ryot, TodoMovies, Trakt or a Watcharr export). Review the rows, change any status, then start the import. Imported titles keep their numeric ratings but arrive without a tier; tier each one yourself.

### 7. Fix failed imports by URL

When some rows can't be matched, the import page stays open and those rows get a **Fix** button. Click it, paste a link to the title (TMDB, IMDb, Letterboxd or Rotten Tomatoes), check the match, and confirm. The row is imported with its original status, dates, review and rating.

### 8. The public stats page

Open `/stats` for totals and charts built from your visible titles only. Every chart also has a table version for screen readers.

## Project Structure

```
Watcharr/
├── server/                      Go API server
│   ├── watcharr.go              Entry point: config, database, UI process, HTTP server
│   ├── app/                     Builds the Gin engine and wires every feature
│   │   └── allowlist.go         Routes open to visitors; everything else is admin-only
│   ├── config/                  watcharr.json loading and saving
│   ├── database/
│   │   ├── entity/              GORM models (Watched, Content, Tier, ...)
│   │   ├── migrate/             Versioned migrations
│   │   └── query/               Shared queries (tier sort and filter)
│   ├── feature/
│   │   ├── auth/                Admin login, password change
│   │   ├── setup/               First-run admin creation with the setup token
│   │   ├── public/              Public read API, visibility rules, stats
│   │   ├── watched/             The admin's list: add, update, tier, hide
│   │   ├── content/             TMDB details and caching
│   │   ├── search/              TMDB search, including pasted urls
│   │   ├── resolve/             Pasted url to TMDB title
│   │   ├── imprt/               Imports (all formats) and the resolve endpoint
│   │   ├── discover/            TMDB discover (admin)
│   │   ├── tag/                 Tags
│   │   └── ...                  Activity, profile, images, jobs, tasks, server settings
│   ├── media/tmdb/              TMDB API client
│   ├── router/                  Middleware and the rate limiter
│   ├── util/safefetch/          Allowlisted, size and time limited page fetching
│   ├── cmd/seed/                Seeds a data dir for the e2e tests
│   ├── cmd/tmdbstub/            Serves TMDB fixtures for the e2e tests
│   ├── internal/testutil/       Test server, TMDB stub and helpers
│   └── testdata/                TMDB fixtures and saved Letterboxd and Rotten Tomatoes pages
├── src/                         SvelteKit UI
│   ├── routes/
│   │   ├── (app)/               Pages with the nav: home, movie, tv, tag, search, stats, import, ...
│   │   ├── (plain)/             Admin login (/admin) and first-run setup (/setup)
│   │   └── +layout.svelte       Root layout, including the footer
│   ├── lib/
│   │   ├── public/              Public API client for visitors
│   │   ├── tier/                Tier badge, picker and display rules
│   │   ├── stats/               Stats page charts
│   │   ├── import/              Fix by URL dialog
│   │   ├── poster/              Posters and poster lists
│   │   ├── util/                API client, route guards, helpers
│   │   └── ...                  Other components
│   ├── store.svelte.ts          App state
│   └── types.ts                 Shared types
├── e2e/                         Playwright tests
├── static/                      Icons, logos and other static files
├── scripts/                     Build helpers (e2e binaries)
├── Dockerfile
├── docker-compose.yml           Runs the published image
└── docker-compose.dev.yml       Builds and runs the checkout
```

## How It Works

### Request flow

In production, the Go server listens on port 3080. Requests under `/api` are handled by Gin. Every other path is proxied to the SvelteKit Node server on port 3000, which serves the single-page app. The browser app then calls `/api` on the same origin. Unknown `/api` paths return a JSON 404 rather than reaching the UI.

### Public and admin API

Every route under `/api` passes through an allowlist middleware (`server/app/allowlist.go`). Only these routes are open to visitors:

- `GET /api/public/*`: the owner's visible titles, tags, reviews and stats
- `POST /api/auth/`: admin login (rate limited)
- `GET /api/auth/available`
- `POST /api/setup/create_admin`: first-run setup (needs the setup token)
- `GET /api/img/*`: cached poster images

Any other route requires a valid admin token, and a test walks every registered route to check this. Public responses never include numeric ratings, user ids, activity or settings. Visitors can't reach TMDB search or Discover, so the TMDB key can't be used as an open proxy.

### What makes a title visible

A title is visible to visitors only when all of these hold (`server/feature/public/visible.go`):

- it belongs to the admin
- it is a movie or TV show
- its status is Finished, Watching or Planned
- it isn't hidden

Every public endpoint and the stats use this one rule.

### Tiers and numeric ratings

The tier is a separate nullable column (`watched.tier`) holding exactly `S`, `A`, `B`, `C`, `D` or `F`; anything else, including lowercase letters, is rejected. The numeric rating keeps its own column and the admin's chosen rating system. Nothing converts one into the other. Imports store numeric ratings as before and leave the tier empty. Sorting by tier goes S to F, then watched titles with no tier, then Planned titles.

### Resolving pasted urls

TMDB and IMDb links are looked up through the TMDB API. Letterboxd and Rotten Tomatoes pages are fetched to read the TMDB id, or the title and year. Page fetching only reaches `letterboxd.com`, `boxd.it` and `rottentomatoes.com` over HTTPS, re-checks the host on every redirect (at most 3), and stops after 5 seconds or 2 MB. Hosts must match a site exactly or be a subdomain of it, so lookalikes such as `fakeimdb.com` are refused.

## Code Quality

Go formatting and static checks:

```bash
cd server
gofmt -l .
go vet ./...
```

Web formatting, linting and type checks:

```bash
npm run lint
npm run check
```

Apply the formatting:

```bash
npm run format
cd server && gofmt -w .
```

## Testing

### Installing test dependencies

The Go tests need only Go and gcc. The web tests need the npm dependencies, and the end-to-end tests need a Playwright browser:

```bash
npm ci
npx playwright install --with-deps chromium
```

### Running tests

Go tests:

```bash
cd server
go test ./...
```

Unit tests (Vitest):

```bash
npm run test:unit
```

End-to-end tests (Playwright). These run the built UI against the real server, a seeded data directory and a TMDB stub, with no network access:

```bash
npm run build
sh scripts/e2e-build-go.sh
npm run test:e2e
```

To run everything in Docker without installing Go or Node, see `e2e/README.md`.

### Test structure

```
server/**/*_test.go        Go tests next to the code they cover
server/internal/testutil/  In-memory test server, admin and user tokens, TMDB stub
server/testdata/           TMDB fixtures and saved pages used by the tests
src/**/*.test.ts           Vitest component and unit tests
e2e/*.spec.ts              Playwright journeys: visitor, admin, tiers, import
```

## Development Status

The single-owner fork is feature complete: public read-only access, tiers, hidden titles, public stats and fixing imports by URL all work and are covered by tests. It is not affiliated with upstream Watcharr and does not track upstream releases.

## Troubleshooting

### CGO or gcc errors when building the server

Errors such as `cgo: C compiler "gcc" not found` or `Binary was compiled with 'CGO_ENABLED=0'` mean SQLite couldn't be compiled. Install gcc and build with `CGO_ENABLED=1`. On Windows use a MinGW-w64 gcc. On Alpine, install `gcc musl-dev build-base` and set `CGO_CFLAGS="-D_LARGEFILE64_SOURCE"`. Building with Docker avoids all of this.

### Missing or rejected TMDB key

If titles, posters or search results don't load and the log shows TMDB errors, set your own `TMDB_KEY` in `data/watcharr.json` or on the server settings page, then restart.

### Lost admin password

There is no password reset command. If you are still signed in, change the password from the profile page. Otherwise restore `data/watcharr.db` from a backup. Deleting the database starts setup again, but also deletes your list.

### Playwright browsers not installed

`Executable doesn't exist at .../ms-playwright/...` means the browser is missing:

```bash
npx playwright install --with-deps chromium
```

When using the Playwright Docker image, its tag must match the `@playwright/test` version in `package.json`.

### Setup token not accepted

The token changes every time the server restarts until the admin exists. Use the one from the latest start in the log.

## License

This project is licensed under the GNU General Public License v3.0 only. See [LICENSE](LICENSE).

It is based on [Watcharr](https://github.com/sbondCo/Watcharr) by sbondCo and its contributors, who hold the copyright to the original code. This fork's changes are listed in [About This Project](#about-this-project). Title data and images come from [TMDB](https://www.themoviedb.org/). This product uses the TMDB API but is not endorsed or certified by TMDB.
