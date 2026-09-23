import { test as base, expect } from "@playwright/test";

// Credentials created by server/cmd/seed.
export const ADMIN = { username: "admin", password: "e2e-admin-password" };

// 1x1 transparent png, served in place of TMDB poster images.
const PNG = Buffer.from(
	"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGMAAQAABQABDQottAAAAABJRU5ErkJggg==",
	"base64",
);

const LOCAL_HOSTS = new Set(["127.0.0.1", "localhost"]);

// Every test gets a page that can never reach the network: TMDB images are
// answered with a placeholder and anything else non-local is aborted.
export const test = base.extend({
	page: async ({ page }, use) => {
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
		await use(page);
	},
});

export { expect };

export async function loginAsAdmin(page: import("@playwright/test").Page) {
	await page.goto("/login");
	await page.getByPlaceholder("Username").fill(ADMIN.username);
	await page.getByPlaceholder("Password").fill(ADMIN.password);
	await page.locator('button[type="submit"]').first().click();
	await page.waitForURL("/");
}
