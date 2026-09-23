import { asAdmin, expect, test } from "./fixtures";

test.describe("admin", () => {
	test("sees hidden and admin only titles with edit controls", async ({
		page,
		adminToken,
	}) => {
		await asAdmin(page, adminToken);
		for (const t of ["The Matrix", "Inception", "Interstellar"]) {
			await expect(page.locator("ul li h2").filter({ hasText: t })).toHaveCount(
				1,
			);
		}
		// Only The Matrix is hidden.
		await expect(page.locator(".hidden-badge")).toHaveCount(1);
	});

	test("can hide and unhide a title from visitors", async ({
		page,
		adminToken,
		request,
	}) => {
		const publicEntry = "/api/public/watched/movie/550";
		await asAdmin(page, adminToken, "/movie/550");
		const checkbox = page.locator("label.hidden-toggle input");
		await expect(checkbox).not.toBeChecked();

		await page.locator("label.hidden-toggle").click();
		await expect(checkbox).toBeChecked();
		await expect(page.getByText("Hidden from visitors").last()).toBeVisible();
		await expect
			.poll(async () => (await request.get(publicEntry)).status())
			.toBe(404);

		// Unhide again so other tests see the seeded state.
		await page.locator("label.hidden-toggle").click();
		await expect(checkbox).not.toBeChecked();
		await expect
			.poll(async () => (await request.get(publicEntry)).status())
			.toBe(200);
	});

	test("logout returns to the visitor view", async ({ page, adminToken }) => {
		await asAdmin(page, adminToken);
		await page.locator("button.face").click();
		await page.getByRole("button", { name: "Logout" }).click();
		await expect(page.locator("button.face")).toHaveCount(0);
		await expect(page.locator("ul li h2").first()).toBeVisible();
		await expect(
			page.locator("ul li h2").filter({ hasText: "The Matrix" }),
		).toHaveCount(0);
	});
});
