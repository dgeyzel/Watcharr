import { render, screen, within } from "@testing-library/svelte";
import { describe, expect, it, vi } from "vitest";
import type { PublicStats } from "@/lib/stats/publicStats";
import StatsPage from "./+page.svelte";

// What the page gets from getPublicStats (a plain function: vi.fn would
// report the rejected promise in the error test as unhandled).
let nextStats: () => Promise<PublicStats>;

vi.mock("@/lib/stats/publicStats", async (orig) => ({
	...(await orig<typeof import("@/lib/stats/publicStats")>()),
	getPublicStats: () => nextStats(),
}));

const stats: PublicStats = {
	totals: { titles: 6, movies: 4, shows: 2 },
	byStatus: { finished: 3, watching: 1, planned: 2 },
	tiers: { S: 1, A: 1, B: 0, C: 1, D: 0, F: 0, unrated: 1 },
	addedPerMonth: [{ month: "2026-09", count: 6 }],
	byDecade: [
		{ decade: 1990, count: 2 },
		{ decade: 2010, count: 4 },
	],
	topGenres: [{ name: "Drama", count: 3 }],
	tags: [{ id: 1, name: "Favourites", count: 2 }],
	finishedMovieHours: 7,
};

function stat(name: string) {
	return screen.getByText(name, { selector: "*" }).parentElement!;
}

describe("stats page", () => {
	it("renders the totals and every chart", async () => {
		nextStats = async () => stats;
		render(StatsPage);
		expect(
			await screen.findByRole("heading", { name: "Tiers" }),
		).toBeInTheDocument();
		expect(stat("Titles")).toHaveTextContent("6");
		expect(stat("Planned")).toHaveTextContent("2");
		expect(stat("Hours of finished movies")).toHaveTextContent("7");
		for (const title of [
			"Added per month",
			"Release decades",
			"Top genres",
			"Tags",
		]) {
			expect(screen.getByRole("heading", { name: title })).toBeInTheDocument();
		}
		// Chart data is also available as a table.
		const tables = screen.getAllByRole("table");
		const tierTable = tables.find((t) =>
			within(t).queryByText("Watched but not yet rated"),
		);
		expect(tierTable).toBeDefined();
		expect(within(tierTable!).getAllByRole("row").length).toBeGreaterThan(6);
		expect(screen.getByText("Sep 2026")).toBeInTheDocument();
		expect(screen.getByText("1990s")).toBeInTheDocument();
		expect(screen.getByText("Favourites")).toBeInTheDocument();
		expect(screen.queryByText(/No titles yet/)).toBeNull();
	});

	it("leaves out the tags chart when there are no tags", async () => {
		nextStats = async () => ({ ...stats, tags: [] });
		render(StatsPage);
		await screen.findByRole("heading", { name: "Tiers" });
		expect(screen.queryByRole("heading", { name: "Tags" })).toBeNull();
	});

	it("shows the empty state", async () => {
		nextStats = async () => ({
			...stats,
			totals: { titles: 0, movies: 0, shows: 0 },
		});
		render(StatsPage);
		expect(await screen.findByText(/No titles yet/)).toBeInTheDocument();
		expect(screen.queryByRole("heading", { name: "Tiers" })).toBeNull();
	});

	it("shows an error when stats can't be loaded", async () => {
		nextStats = async () => {
			throw new Error("offline");
		};
		render(StatsPage);
		expect(
			await screen.findByText("Failed to load stats!"),
		).toBeInTheDocument();
	});
});
