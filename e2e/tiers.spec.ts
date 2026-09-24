import { asAdmin, expect, newVisitorPage, test } from "./fixtures";

// Seeded (server/cmd/seed) tiers: Breaking Bad S, Fight Club A, Game of
// Thrones C (watching), Pulp Fiction watching and not rated, Forrest Gump and
// Stranger Things planned. Hidden/admin only titles also have tiers that
// visitors must never see.

function poster(page: import("@playwright/test").Page, title: string) {
	return page
		.locator("ul li")
		.filter({ has: page.locator("h2", { hasText: title }) });
}

test.describe("tiers for visitors", () => {
	test("posters show the tier slot", async ({ page }) => {
		await page.goto("/");
		await expect(
			poster(page, "Fight Club").getByRole("img", { name: "Tier: A" }),
		).toBeVisible();
		await expect(
			poster(page, "Breaking Bad").getByRole("img", { name: "Tier: S" }),
		).toBeVisible();
		// Watched but not tiered.
		await expect(
			poster(page, "Pulp Fiction").locator(".tier-unrated"),
		).toHaveText("not rated yet");
		// Planned: no tier slot at all.
		for (const t of ["Forrest Gump", "Stranger Things"]) {
			await expect(
				poster(page, t).locator(".tier-badge, .tier-unrated"),
			).toHaveCount(0);
		}
	});

	test("title pages show the tier slot and review, never a number", async ({
		page,
	}) => {
		await page.goto("/movie/550");
		await expect(page.getByRole("img", { name: "Tier: A" })).toBeVisible();
		await expect(
			page.getByText("A seeded review of Fight Club."),
		).toBeVisible();
		await expect(
			page.getByText(/\b\d+(\.\d+)?\s*\/\s*10\b|out of 10/),
		).toHaveCount(0);

		await page.goto("/movie/680");
		await expect(page.getByText("Watched but not yet rated")).toBeVisible();

		await page.goto("/movie/13");
		await expect(page.getByText("planned", { exact: true })).toBeVisible();
		await expect(page.locator(".tier-badge, .tier-unrated")).toHaveCount(0);
	});

	test("sort by tier is S to F with not rated after", async ({ page }) => {
		await page.goto("/");
		await page.locator("button.sort").click();
		// The active sort gets a direction arrow (::before) in its accessible name.
		const tierSort = page.getByRole("button", { name: /^(↑ |↓ )?Tier$/ });
		// The first click sorts best first (S to F).
		await tierSort.click();
		const titles = page.locator("ul li h2");
		await expect(titles.first()).toContainText("Breaking Bad");
		const order = (await titles.allTextContents()).map((t) =>
			t.replace(/\d{4}\s*$/, "").trim(),
		);
		expect(order).toEqual([
			"Breaking Bad",
			"Fight Club",
			"Game of Thrones",
			"Pulp Fiction",
			"Forrest Gump",
			"Stranger Things",
		]);
		// Second click flips the tiers only (F to S), not rated and planned
		// stay last.
		await tierSort.click();
		await expect(titles.first()).toContainText("Game of Thrones");
		expect(
			(await titles.allTextContents()).map((t) =>
				t.replace(/\d{4}\s*$/, "").trim(),
			),
		).toEqual([
			"Game of Thrones",
			"Fight Club",
			"Breaking Bad",
			"Pulp Fiction",
			"Forrest Gump",
			"Stranger Things",
		]);
		// Third click turns the sort off (default order for other tests).
		await tierSort.click();
	});

	test("filter by tier", async ({ page }) => {
		await page.goto("/");
		await page.locator("button.filter").click();
		await page
			.getByRole("button", { name: "Watched but not yet rated" })
			.click();
		const titles = page.locator("ul li h2");
		await expect(titles).toHaveCount(1);
		await expect(titles.first()).toContainText("Pulp Fiction");
		await page
			.getByRole("button", { name: "Watched but not yet rated" })
			.click();
		await expect(titles).toHaveCount(6);
	});
});

test.describe("public stats", () => {
	test("shows the seeded numbers", async ({ page }) => {
		await page.goto("/");
		await page.getByRole("link", { name: "Stats" }).click();
		await expect(page).toHaveURL(/\/stats$/);
		const tile = (name: string) =>
			page.locator("a").filter({ has: page.getByText(name, { exact: true }) });
		await expect(tile("Titles")).toContainText("6");
		await expect(tile("Movies")).toContainText("3");
		await expect(tile("Shows")).toContainText("3");
		await expect(tile("Planned")).toContainText("2");

		const tiers = page.getByRole("table", { name: "Tiers" });
		const row = (label: string) =>
			tiers.getByRole("row").filter({
				has: page.getByRole("rowheader", { name: label, exact: true }),
			});
		await expect(row("S")).toContainText("1");
		await expect(row("A")).toContainText("1");
		await expect(row("B")).toContainText("0");
		await expect(row("C")).toContainText("1");
		await expect(row("Watched but not yet rated")).toContainText("1");

		await expect(
			page.getByRole("table", { name: "Release decades" }),
		).toContainText("1990s");
		await expect(page.getByRole("table", { name: "Top genres" })).toContainText(
			"Drama",
		);
		// No numeric ratings anywhere.
		await expect(page.getByText(/average|out of 10/i)).toHaveCount(0);
	});
});

test.describe("tiers for the admin", () => {
	test("admin sets, changes and clears a tier, visitors see it right away", async ({
		page,
		adminToken,
		browser,
	}) => {
		await asAdmin(page, adminToken, "/movie/680");
		const picker = page.getByRole("toolbar", { name: "Tier" });
		await expect(picker).toBeVisible();
		// The numeric rating widget is still there for the admin.
		await expect(
			page.locator(".review .rating, .review [class*='rating']").first(),
		).toBeVisible();

		const visitor = await newVisitorPage(browser);
		try {
			await picker.getByRole("button", { name: "Tier B" }).click();
			await expect(
				picker.getByRole("button", { name: "Tier B" }),
			).toHaveAttribute("aria-pressed", "true");
			await visitor.goto("/movie/680");
			await expect(visitor.getByRole("img", { name: "Tier: B" })).toBeVisible();

			await picker.getByRole("button", { name: "Tier S" }).click();
			await expect(
				picker.getByRole("button", { name: "Tier S" }),
			).toHaveAttribute("aria-pressed", "true");
			await visitor.reload();
			await expect(visitor.getByRole("img", { name: "Tier: S" })).toBeVisible();

			// Clear again (restores the seeded state).
			await picker.getByRole("button", { name: "Clear tier" }).click();
			await expect(
				picker.getByRole("button", { name: "Clear tier" }),
			).toBeDisabled();
			await visitor.reload();
			await expect(
				visitor.getByText("Watched but not yet rated"),
			).toBeVisible();
		} finally {
			await visitor.context().close();
		}
	});
});
