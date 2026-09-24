import { render, screen, within } from "@testing-library/svelte";
import { describe, expect, it } from "vitest";
import BarChart from "./BarChart.svelte";
import {
	decadeBars,
	tierBars,
	monthLabel,
	type PublicStats,
} from "./publicStats";

describe("BarChart", () => {
	it("renders every bar as a table row (text fallback)", () => {
		render(BarChart, {
			title: "Top genres",
			bars: [
				{ label: "Drama", value: 4 },
				{ label: "Crime", value: 2 },
			],
		});
		expect(
			screen.getByRole("heading", { name: "Top genres" }),
		).toBeInTheDocument();
		const table = screen.getByRole("table", { name: "Top genres" });
		const rows = within(table).getAllByRole("row");
		// Header row + 2 data rows.
		expect(rows).toHaveLength(3);
		expect(within(rows[1]).getByRole("rowheader")).toHaveTextContent("Drama");
		expect(rows[1]).toHaveTextContent("4");
		expect(rows[2]).toHaveTextContent("Crime");
		expect(rows[2]).toHaveTextContent("2");
	});

	it("scales bars to the largest value", () => {
		const { container } = render(BarChart, {
			title: "x",
			bars: [
				{ label: "a", value: 4 },
				{ label: "b", value: 1 },
			],
		});
		const bars = container.querySelectorAll<HTMLElement>(".bar");
		expect(bars[0].style.width).toBe("100%");
		expect(bars[1].style.width).toBe("25%");
	});

	it("shows an empty state when there is no data", () => {
		render(BarChart, {
			title: "Tiers",
			bars: [{ label: "S", value: 0 }],
		});
		expect(screen.queryByRole("table")).toBeNull();
		expect(screen.getByText("Nothing to show yet.")).toBeInTheDocument();
	});
});

const stats: PublicStats = {
	totals: { titles: 6, movies: 3, shows: 3 },
	byStatus: { finished: 2, watching: 2, planned: 2 },
	tiers: { S: 1, A: 1, B: 0, C: 1, D: 0, F: 0, unrated: 1 },
	addedPerMonth: [{ month: "2026-09", count: 6 }],
	byDecade: [
		{ decade: 1990, count: 3 },
		{ decade: 2000, count: 1 },
	],
	topGenres: [],
	tags: [],
	finishedMovieHours: 2.3,
};

describe("public stats helpers", () => {
	it("builds tier bars S to F plus the not rated count", () => {
		expect(tierBars(stats).map((b) => [b.label, b.value])).toEqual([
			["S", 1],
			["A", 1],
			["B", 0],
			["C", 1],
			["D", 0],
			["F", 0],
			["Watched but not yet rated", 1],
		]);
	});

	it("labels decades and months", () => {
		expect(decadeBars(stats).map((b) => b.label)).toEqual(["1990s", "2000s"]);
		expect(monthLabel("2026-09")).toBe("Sep 2026");
		expect(monthLabel("2025-12")).toBe("Dec 2025");
	});
});
