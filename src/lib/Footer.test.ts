import { render, screen } from "@testing-library/svelte";
import { describe, expect, it } from "vitest";
import Footer from "./Footer.svelte";

describe("Footer", () => {
	it("credits the fork and upstream Watcharr with exact links", () => {
		const { container } = render(Footer);
		const footer = container.querySelector("footer");
		expect(footer).not.toBeNull();
		expect(
			footer!.querySelector("p")!.textContent!.replace(/\s+/g, " ").trim(),
		).toBe("This site is a fork of Watcharr");

		const fork = screen.getByRole("link", { name: "fork" });
		expect(fork).toHaveAttribute("href", "https://github.com/dgeyzel/Watcharr");
		const upstream = screen.getByRole("link", { name: "Watcharr" });
		expect(upstream).toHaveAttribute("href", "https://watcharr.app/");
		for (const link of [fork, upstream]) {
			expect(link).toHaveAttribute("target", "_blank");
			expect(link).toHaveAttribute("rel", "noopener noreferrer");
		}
	});

	it("shows the TMDB attribution notice", () => {
		render(Footer);
		expect(
			screen.getByText(
				"This product uses the TMDB API but is not endorsed or certified by TMDB.",
			),
		).toBeInTheDocument();
	});
});
