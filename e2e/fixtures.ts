import {
	test as base,
	expect,
	type APIRequestContext,
	type Browser,
	type Page,
} from "@playwright/test";

// Credentials created by server/cmd/seed.
export const ADMIN = { username: "admin", password: "e2e-admin-password" };

// 1x1 transparent png, served in place of TMDB poster images.
const PNG = Buffer.from(
	"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGMAAQAABQABDQottAAAAABJRU5ErkJggg==",
	"base64",
);

const LOCAL_HOSTS = new Set(["127.0.0.1", "localhost"]);

/**
 * Make a page unable to reach the network: TMDB images are answered with a
 * placeholder and anything else non-local is aborted.
 */
async function blockNetwork(page: Page) {
	await page.route("**/*", (route) => {
		const url = new URL(route.request().url());
		if (LOCAL_HOSTS.has(url.hostname) || url.protocol === "data:") {
			return route.continue();
		}
		if (url.hostname === "image.tmdb.org") {
			return route.fulfill({ contentType: "image/png", body: PNG });
		}
		return route.abort();
	});
}

/**
 * A separate, signed out (visitor) page, e.g. to check what visitors see
 * while the admin is signed in on `page`. Close it when done.
 */
export async function newVisitorPage(browser: Browser): Promise<Page> {
	const ctx = await browser.newContext({ baseURL: "http://127.0.0.1:3080" });
	const page = await ctx.newPage();
	await blockNetwork(page);
	return page;
}

type WorkerFixtures = {
	// Admin auth token, fetched once per worker. Login is rate limited, so
	// tests use this instead of logging in through the form every time.
	adminToken: string;
};

// Every test gets a page that can never reach the network.
export const test = base.extend<object, WorkerFixtures>({
	page: async ({ page }, use) => {
		await blockNetwork(page);
		await use(page);
	},
	adminToken: [
		async ({ playwright }, use) => {
			const ctx = await playwright.request.newContext({
				baseURL: "http://127.0.0.1:3080",
			});
			const res = await ctx.post("/api/auth/", { data: ADMIN });
			expect(res.status(), await res.text()).toBe(200);
			const { token } = await res.json();
			await ctx.dispose();
			await use(token);
		},
		{ scope: "worker" },
	],
});

export { expect };

/**
 * Log in through the real admin login form. Login is rate limited on the
 * server, so only use this where the form itself is being tested (prefer
 * `asAdmin`).
 */
export async function loginThroughForm(page: Page) {
	await page.goto("/admin");
	await page.getByPlaceholder("Username").fill(ADMIN.username);
	await page.getByPlaceholder("Password").fill(ADMIN.password);
	await page.locator('button[type="submit"]').first().click();
	await page.waitForURL("/");
}

/**
 * Open the app as the signed in admin (token injected, no form login).
 */
export async function asAdmin(page: Page, token: string, path = "/") {
	await page.goto("/admin");
	await page.evaluate((t) => localStorage.setItem("token", t), token);
	await page.goto(path);
	await expect(page.locator("button.face")).toBeVisible();
}

/**
 * The admin's watched entry id for a TMDB title, if it's on the list.
 */
export async function watchedIdFor(
	request: APIRequestContext,
	token: string,
	tmdbId: number,
): Promise<number | undefined> {
	const res = await request.get("/api/watched", {
		headers: { Authorization: token },
	});
	expect(res.status()).toBe(200);
	const list: { id: number; content?: { tmdbId: number } }[] = await res.json();
	return list.find((w) => w.content?.tmdbId === tmdbId)?.id;
}

/**
 * Take a title off the admin's list (if it's there), so tests that add
 * titles leave the seeded list as it was.
 */
export async function removeFromList(
	request: APIRequestContext,
	token: string,
	tmdbId: number,
) {
	const id = await watchedIdFor(request, token, tmdbId);
	if (id) {
		const res = await request.delete(`/api/watched/${id}`, {
			headers: { Authorization: token },
		});
		expect(res.status()).toBe(200);
	}
}
