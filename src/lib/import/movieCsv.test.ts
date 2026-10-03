import { describe, expect, it } from "vitest";
import { parseMovieCsv } from "./movieCsv";

describe("parseMovieCsv", () => {
	it("reads name, imdb link and tier", () => {
		const r = parseMovieCsv(
			"The Thing,https://www.imdb.com/title/tt0084787/,A\n",
		);
		expect(r.rows).toEqual([
			{
				name: "The Thing",
				type: "movie",
				imdbId: "tt0084787",
				imdbStrict: true,
				tier: "A",
			},
		]);
		expect(r.badLinks).toEqual([]);
		expect(r.badTiers).toEqual([]);
	});

	it("leaves an empty link or tier empty", () => {
		const r = parseMovieCsv("Alien,,S\nAliens,tt0090605,\nHeat\n");
		expect(r.rows).toEqual([
			{ name: "Alien", type: "movie", tier: "S" },
			{
				name: "Aliens",
				type: "movie",
				imdbId: "tt0090605",
				imdbStrict: true,
			},
			{ name: "Heat", type: "movie" },
		]);
	});

	it("skips a header row and blank lines", () => {
		const r = parseMovieCsv("Movie Name,IMDb Link,Tier\n\nHeat,,b\n  \n");
		expect(r.rows).toEqual([{ name: "Heat", type: "movie", tier: "B" }]);
	});

	it("keeps a first row named like a header when it has data", () => {
		const r = parseMovieCsv("Movie,tt0000001,C\n");
		expect(r.rows[0]).toMatchObject({ name: "Movie", imdbId: "tt0000001" });
	});

	it("handles quoted names with commas and a year", () => {
		const r = parseMovieCsv(
			'"Crouching Tiger, Hidden Dragon (2000)", https://m.imdb.com/title/tt0190332/?ref_=x , F',
		);
		expect(r.rows).toEqual([
			{
				name: "Crouching Tiger, Hidden Dragon",
				year: 2000,
				type: "movie",
				imdbId: "tt0190332",
				imdbStrict: true,
				tier: "F",
			},
		]);
	});

	it("reports bad links and tiers but keeps the row", () => {
		const r = parseMovieCsv("Heat,https://letterboxd.com/film/heat-1995/,A+\n");
		expect(r.rows).toEqual([{ name: "Heat", type: "movie" }]);
		expect(r.badLinks).toEqual(["https://letterboxd.com/film/heat-1995/"]);
		expect(r.badTiers).toEqual(["A+"]);
	});

	it("imports a row with only a link", () => {
		const r = parseMovieCsv(",tt0084787,\n,not a link,\n");
		expect(r.rows).toEqual([
			{ name: "", type: "movie", imdbId: "tt0084787", imdbStrict: true },
		]);
		expect(r.badLinks).toEqual(["not a link"]);
	});
});
