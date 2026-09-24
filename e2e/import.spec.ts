import type { APIRequestContext } from "@playwright/test";
import { asAdmin, expect, test } from "./fixtures";

// The Thing (1982), resolved from its IMDb url by the TMDB stub.
const THE_THING = 1091;

async function watchedIdFor(
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

async function removeWatched(
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

test.describe("import", () => {
	// Leave the seeded list as it was for the other specs.
	test.afterEach(async ({ request, adminToken }) => {
		await removeWatched(request, adminToken, THE_THING);
	});

	test("a failed row is fixed by pasting an IMDb url", async ({
		page,
		adminToken,
		request,
	}) => {
		expect(await watchedIdFor(request, adminToken, THE_THING)).toBeUndefined();

		await asAdmin(page, adminToken, "/import");
		await page
			.locator(".drop-file-btn")
			.filter({ hasText: ".txt list" })
			.locator('input[type="file"]')
			.setInputFiles({
				name: "list.txt",
				mimeType: "text/plain",
				buffer: Buffer.from("A Title Nobody Can Find (1982) [8]\n"),
			});
		await page.waitForURL("/import/process");
		await expect(page.locator("td.name input")).toHaveValue(
			"A Title Nobody Can Find",
		);

		await page.getByRole("button", { name: "Start Importing" }).click();
		const fix = page.getByRole("button", {
			name: "Fix A Title Nobody Can Find",
		});
		await expect(fix).toBeVisible({ timeout: 15000 });
		// Failed rows keep the admin on the page.
		await expect(page).toHaveURL(/\/import\/process$/);

		await fix.click();
		const url = page.getByRole("textbox", { name: "Title url" });
		// A site the resolver doesn't support.
		await url.fill("https://fakeimdb.com/title/tt0084787/");
		await page.getByRole("button", { name: "Find", exact: true }).click();
		await expect(page.locator(".modal .error")).toBeVisible();

		await url.fill("https://www.imdb.com/title/tt0084787/");
		await page.getByRole("button", { name: "Find", exact: true }).click();
		const match = page.getByRole("button", { name: /The Thing.*1982/ });
		await expect(match).toHaveAttribute("aria-pressed", "true");
		await page.getByRole("button", { name: "Import", exact: true }).click();

		await expect(page.locator(".modal")).toHaveCount(0);
		await expect(fix).toHaveCount(0);
		await expect(page.locator("td.name input")).toHaveValue("The Thing");

		// Imported with the row's rating and no tier.
		await page.goto(`/movie/${THE_THING}`);
		await expect(
			page.getByRole("button", { name: "Set thoughts on The Thing" }),
		).toBeVisible();
		const res = await request.get("/api/watched", {
			headers: { Authorization: adminToken },
		});
		const w = (
			(await res.json()) as {
				rating: number;
				status: string;
				tier: string | null;
				content?: { tmdbId: number };
			}[]
		).find((x) => x.content?.tmdbId === THE_THING);
		expect(w).toMatchObject({ rating: 8, status: "FINISHED", tier: null });
	});
});
