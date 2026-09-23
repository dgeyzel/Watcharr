import { expect, loginThroughForm, test } from "./fixtures";

test("admin login page shows the fork footer", async ({ page }) => {
	await page.goto("/admin");
	const footer = page.locator("footer");
	await expect(footer.getByRole("link", { name: "fork" })).toHaveAttribute(
		"href",
		"https://github.com/dgeyzel/Watcharr",
	);
	await expect(footer.getByRole("link", { name: "Watcharr" })).toHaveAttribute(
		"href",
		"https://watcharr.app/",
	);
	await expect(footer).toContainText(
		"This product uses the TMDB API but is not endorsed or certified by TMDB.",
	);
});

test("admin can log in through the form", async ({ page }) => {
	await loginThroughForm(page);
	await expect(page.getByText("Fight Club").first()).toBeVisible();
	await expect(page.locator("footer")).toBeVisible();
});

test("write api rejects anonymous requests", async ({ request }) => {
	const res = await request.put("/api/watched/1", {
		data: { status: "DROPPED" },
	});
	expect(res.status()).toBe(401);
});
