import { defineConfig, devices } from "@playwright/test";

// E2E tests run against the built UI + real Go server, with TMDB served by a
// local fixture stub (no network). Before running:
//   npm run build
//   sh scripts/e2e-build-go.sh
// See e2e/README.md.
const bin = process.env.E2E_BIN ?? "server/.e2e-bin";
const dataDir = process.env.E2E_DATA ?? "e2e/.data";
const fixtures = "server/testdata/tmdb";
const stub = "http://127.0.0.1:3099";

export default defineConfig({
	testDir: "e2e",
	// Tests share one seeded server, keep them serial.
	workers: 1,
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
	use: {
		baseURL: "http://127.0.0.1:3080",
		trace: "retain-on-failure",
	},
	projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
	webServer: [
		{
			command: `${bin}/tmdbstub -addr 127.0.0.1:3099 -dir ${fixtures}`,
			url: `${stub}/movie/550?api_key=healthcheck`,
			reuseExistingServer: false,
		},
		{
			command: "node build",
			url: "http://127.0.0.1:3000",
			env: { PORT: "3000", HOST: "127.0.0.1" },
			reuseExistingServer: false,
		},
		{
			// Fresh data dir every run, seeded, then the real server.
			command: `rm -rf ${dataDir} && ${bin}/seed && ${bin}/watcharr`,
			url: "http://127.0.0.1:3080/api/auth/available",
			env: {
				WATCHARR_DATA: dataDir,
				WATCHARR_SKIP_UI: "1",
				TMDB_FIXTURES_DIR: fixtures,
				TMDB_API_BASE: stub,
				TMDB_IMAGE_BASE: `${stub}/t/p`,
			},
			reuseExistingServer: false,
		},
	],
});
