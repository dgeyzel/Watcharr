import type { Page } from "@playwright/test";
import {
	expect,
	loginThroughForm,
	newVisitorPage,
	removeFromList,
	test,
	watchedIdFor,
} from "./fixtures";

// Stub titles that aren't in the seeded list.
const THE_THING = 1091;
const THE_GODFATHER = 238;

function poster(page: Page, title: string) {
	return page
		.locator("ul li")
		.filter({ has: page.locator("h2", { hasText: title }) });
}

async function setStatus(page: Page, status: "finished" | "planned") {
	const btn = page.locator(".review").getByRole("button", { name: status });
	await btn.click();
	await expect(btn).toHaveAttribute("aria-pressed", "true");
}

test.describe("admin journey", () => {
	test.afterEach(async ({ request, adminToken }) => {
		await removeFromList(request, adminToken, THE_THING);
		await removeFromList(request, adminToken, THE_GODFATHER);
	});

	test("add, tier, review and hide titles; visitors follow along", async ({
		page,
		browser,
		request,
		adminToken,
	}) => {
		for (const id of [THE_THING, THE_GODFATHER]) {
			expect(await watchedIdFor(request, adminToken, id)).toBeUndefined();
		}
		await loginThroughForm(page);
		const visitor = await newVisitorPage(browser);
		try {
			// Add a Finished title: visitors see it at once, not rated yet.
			await page.goto(`/movie/${THE_THING}`);
			await setStatus(page, "finished");
			await visitor.goto("/");
			await expect(
				poster(visitor, "The Thing").getByText("not rated yet"),
			).toBeVisible();
			await visitor.goto(`/movie/${THE_THING}`);
			await expect(
				visitor.getByText("Watched but not yet rated"),
			).toBeVisible();

			// Add a Planned title: status only, no tier slot.
			await page.goto(`/movie/${THE_GODFATHER}`);
			await setStatus(page, "planned");
			await visitor.goto("/");
			const planned = poster(visitor, "The Godfather");
			await expect(planned).toHaveCount(1);
			await expect(planned.locator(".tier-slot")).toBeEmpty();
			await visitor.goto(`/movie/${THE_GODFATHER}`);
			const review = visitor.getByRole("region", { name: "Review" });
			await expect(review).toContainText(/planned/i);
			await expect(review.getByText("not yet rated")).toHaveCount(0);
			await expect(review.getByRole("img", { name: /^Tier:/ })).toHaveCount(0);

			// Tier the first title A and review it.
			await page.goto(`/movie/${THE_THING}`);
			const picker = page.getByRole("toolbar", { name: "Tier" });
			await picker.getByRole("button", { name: "Tier A" }).click();
			await expect(
				picker.getByRole("button", { name: "Tier A" }),
			).toHaveAttribute("aria-pressed", "true");
			await page.locator("button.thoughts").click();
			await page
				.getByPlaceholder("My thoughts on The Thing")
				.fill("Paranoia on ice. Still the best creature effects.");
			await page.locator(".modal button.close").click();
			await expect(page.locator(".modal")).toHaveCount(0);

			// Signed out, the site shows the A and the review.
			await page.locator("button.face").click();
			await page.getByRole("button", { name: "Logout" }).click();
			await expect(page.locator("button.face")).toHaveCount(0);
			await page.goto(`/movie/${THE_THING}`);
			await expect(page.getByRole("img", { name: "Tier: A" })).toBeVisible();
			await expect(
				page.getByText("Paranoia on ice. Still the best creature effects."),
			).toBeVisible();
			await expect(page.locator(".rating")).toHaveCount(0);
			await page.goto("/");
			await expect(
				poster(page, "The Thing").getByRole("img", { name: "Tier: A" }),
			).toBeVisible();

			// Hide it again (as the admin): visitors lose it.
			const id = await watchedIdFor(request, adminToken, THE_THING);
			const res = await request.put(`/api/watched/${id}`, {
				headers: { Authorization: adminToken },
				data: { hidden: true },
			});
			expect(res.status()).toBe(200);
			await visitor.goto("/");
			await expect(poster(visitor, "The Godfather")).toHaveCount(1);
			await expect(poster(visitor, "The Thing")).toHaveCount(0);
			await visitor.goto(`/movie/${THE_THING}`);
			await expect(visitor.getByText("Movie not found")).toBeVisible();
		} finally {
			await visitor.context().close();
		}
	});
});
