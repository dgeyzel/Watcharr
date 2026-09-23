import { expect, test } from "./fixtures";

// Seeded (server/cmd/seed): visible titles, plus The Matrix (hidden),
// Inception (on hold) and Interstellar (dropped) which visitors never see.
const VISIBLE = [
	"Fight Club",
	"Pulp Fiction",
	"Forrest Gump",
	"Breaking Bad",
	"Game of Thrones",
	"Stranger Things",
];
const NOT_VISIBLE = ["The Matrix", "Inception", "Interstellar"];

test.describe("visitor", () => {
	test("home shows only visible titles, read only", async ({ page }) => {
		await page.goto("/");
		const posters = page.locator("ul li h2");
		await expect(posters).toHaveCount(VISIBLE.length);
		for (const t of VISIBLE) {
			await expect(posters.filter({ hasText: t })).toHaveCount(1);
		}
		for (const t of NOT_VISIBLE) {
			await expect(page.getByText(t)).toHaveCount(0);
		}
		// No account / admin controls.
		await expect(page.locator("button.face")).toHaveCount(0);
		await expect(page.locator("button.discover")).toHaveCount(0);
		await expect(page.locator("button.following")).toHaveCount(0);
		await expect(page.getByText(/sign up|not a user|login/i)).toHaveCount(0);
		await expect(page.locator(".hidden-badge")).toHaveCount(0);
		// Posters have no rating buttons.
		await expect(page.locator("ul li .buttons .rating")).toHaveCount(0);
	});

	test("title page shows the review, no rating or edit controls", async ({
		page,
	}) => {
		await page.goto("/movie/550");
		await expect(page.getByText("Fight Club").first()).toBeVisible();
		await expect(
			page.getByText("A seeded review of Fight Club."),
		).toBeVisible();
		await expect(page.locator(".rating")).toHaveCount(0);
		await expect(page.getByText(/out of 10/)).toHaveCount(0);
		await expect(page.locator(".other-side")).toHaveCount(0);
		await expect(page.locator("textarea")).toHaveCount(0);
		await expect(page.getByText("Hidden from visitors")).toHaveCount(0);
	});

	test("hidden and admin only titles are not found", async ({ page }) => {
		for (const p of ["/movie/603", "/movie/27205", "/movie/157336"]) {
			await page.goto(p);
			await expect(page.getByText("Movie not found")).toBeVisible();
		}
		await page.goto("/tv/1399");
		await expect(page.getByText("Game of Thrones").first()).toBeVisible();
	});

	test("tag page lists only visible titles", async ({ page }) => {
		await page.goto("/");
		await page.locator("button.tag").click();
		await page.getByRole("button", { name: "Favourites" }).click();
		await expect(page).toHaveURL(/\/tag\/\d+$/);
		const posters = page.locator("ul li h2");
		// Favourites: Fight Club, Breaking Bad (+ Inception, which is on hold).
		await expect(posters).toHaveCount(2);
		await expect(posters.filter({ hasText: "Inception" })).toHaveCount(0);
	});

	test("search only searches the owner's list", async ({ page }) => {
		await page.goto("/search?query=fight");
		const posters = page.locator("ul li h2");
		await expect(posters).toHaveCount(1);
		await expect(posters.first()).toContainText("Fight Club");
	});

	test("admin only pages send visitors home", async ({ page }) => {
		for (const p of ["/discover", "/import", "/server", "/profile"]) {
			await page.goto(p);
			await expect(page).toHaveURL(/\/$/);
		}
	});

	test("the old login page is gone", async ({ page }) => {
		const res = await page.goto("/login");
		expect(res?.status()).toBe(404);
	});

	test("footer shows on every visitor page", async ({ page }) => {
		for (const p of [
			"/",
			"/movie/550",
			"/tv/1396",
			"/search?query=a",
			"/admin",
		]) {
			await page.goto(p);
			const footer = page.locator("footer");
			await expect(footer.getByRole("link", { name: "fork" })).toHaveAttribute(
				"href",
				"https://github.com/dgeyzel/Watcharr",
			);
			await expect(
				footer.getByRole("link", { name: "Watcharr" }),
			).toHaveAttribute("href", "https://watcharr.app/");
		}
	});
});

test.describe("visitor api", () => {
	test("writes and admin reads are rejected without a token", async ({
		request,
	}) => {
		expect((await request.put("/api/watched/1", { data: {} })).status()).toBe(
			401,
		);
		expect((await request.get("/api/watched")).status()).toBe(401);
		expect((await request.get("/api/search?query=a")).status()).toBe(401);
		const pub = await request.get("/api/public/watched");
		expect(pub.status()).toBe(200);
		expect(await pub.text()).not.toContain('"rating"');
	});
});
