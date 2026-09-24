import { describe, expect, it } from "vitest";
import { isAdminOnlyRoute } from "./routes";

describe("isAdminOnlyRoute", () => {
	it.each([
		"/discover",
		"/person/123",
		"/import",
		"/import/process",
		"/import/some-failed",
		"/server",
		"/profile",
	])("%s is admin only", (p) => {
		expect(isAdminOnlyRoute(p)).toBe(true);
	});

	it.each(["/", "/movie/550", "/tv/1396", "/tag/1", "/search", "/admin"])(
		"%s is public",
		(p) => {
			expect(isAdminOnlyRoute(p)).toBe(false);
		},
	);

	it("does not match routes that only share a prefix", () => {
		expect(isAdminOnlyRoute("/profiles")).toBe(false);
		expect(isAdminOnlyRoute("/discovery")).toBe(false);
	});
});
