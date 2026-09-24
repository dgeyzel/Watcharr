import { render, screen } from "@testing-library/svelte";
import { describe, expect, it } from "vitest";
import TierBadge from "./TierBadge.svelte";

describe("TierBadge", () => {
	it("shows the letter with an accessible label", () => {
		render(TierBadge, { status: "FINISHED", tier: "A", size: "page" });
		const badge = screen.getByRole("img", { name: "Tier: A" });
		expect(badge).toHaveTextContent("A");
		expect(badge).toHaveClass("tier-badge", "page", "tier-a");
	});

	it("shows the exact unrated text per size", () => {
		const { unmount } = render(TierBadge, {
			status: "FINISHED",
			tier: null,
			size: "page",
		});
		expect(screen.getByText("Watched but not yet rated")).toBeInTheDocument();
		unmount();
		render(TierBadge, { status: "WATCHING", tier: null, size: "poster" });
		// Exact text, no text-transform tricks.
		expect(screen.getByText("not rated yet").textContent).toBe("not rated yet");
	});

	it("renders nothing for planned titles, even with a saved tier", () => {
		const { container } = render(TierBadge, {
			status: "PLANNED",
			tier: "S",
			size: "poster",
		});
		expect(container.textContent?.trim()).toBe("");
		expect(screen.queryByRole("img")).toBeNull();
	});
});
