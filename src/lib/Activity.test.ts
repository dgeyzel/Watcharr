import { render, screen } from "@testing-library/svelte";
import { describe, expect, it } from "vitest";
import Activity from "./Activity.svelte";
import type { Activity as ActivityT } from "@/types";

function tierActivity(id: number, data: string | undefined): ActivityT {
	return {
		id,
		watchedId: 1,
		type: "TIER_CHANGED",
		data,
		createdAt: new Date(2026, 8, 20 + id).toISOString(),
	} as unknown as ActivityT;
}

describe("Activity", () => {
	it("describes tier changes instead of showing the raw type", () => {
		render(Activity, {
			activity: [
				tierActivity(1, JSON.stringify({ old: null, new: "B" })),
				tierActivity(2, JSON.stringify({ old: "B", new: "D" })),
				tierActivity(3, JSON.stringify({ old: "D", new: null })),
				tierActivity(4, "not json"),
			],
			onRemoved: () => {},
		});
		expect(screen.getByText("Tier Set to B")).toBeInTheDocument();
		expect(screen.getByText("Tier Changed from B to D")).toBeInTheDocument();
		expect(screen.getByText("Tier D Cleared")).toBeInTheDocument();
		expect(screen.getByText("Tier Changed")).toBeInTheDocument();
		expect(screen.queryByText("TIER_CHANGED")).toBeNull();
	});
});
