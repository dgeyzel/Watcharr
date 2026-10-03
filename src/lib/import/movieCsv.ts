import papa from "papaparse";
import { TIERS, type ImportedList, type Tier } from "@/types";

/** Same shape as the server's imdb id check (resolve/url.go). */
const IMDB_ID = /\btt\d{5,10}\b/;
/** A year in brackets at the end of a name, eg "Alien (1979)". */
const YEAR = /\(([0-9]{4})\)\s*$/;
const HEADER_NAMES = ["name", "movie", "title", "movie name"];

export interface MovieCsvResult {
	rows: ImportedList[];
	/** Non-empty imdb cells that had no tt id in them. */
	badLinks: string[];
	/** Non-empty tier cells that aren't one of S-F. */
	badTiers: string[];
}

function parseTier(s: string): Tier | undefined {
	const t = s.toUpperCase();
	return TIERS.includes(t as Tier) ? (t as Tier) : undefined;
}

/**
 * Parse a csv where each row is: movie name, imdb link, tier. The link and
 * tier can be empty. An optional header row is skipped. Bad links or tiers
 * are left empty (and reported) instead of failing the row.
 */
export function parseMovieCsv(data: string): MovieCsvResult {
	const res: MovieCsvResult = { rows: [], badLinks: [], badTiers: [] };
	const parsed = papa.parse<string[]>(data.trim(), {
		header: false,
		skipEmptyLines: "greedy",
	});
	for (let i = 0; i < parsed.data.length; i++) {
		const [name = "", link = "", tier = ""] = parsed.data[i].map((c) =>
			(c ?? "").trim(),
		);
		if (
			i === 0 &&
			HEADER_NAMES.includes(name.toLowerCase()) &&
			!IMDB_ID.test(link) &&
			!parseTier(tier)
		) {
			continue;
		}
		const imdbId = link.match(IMDB_ID)?.[0];
		if (!name && !imdbId) {
			if (link) res.badLinks.push(link);
			continue;
		}
		const l: ImportedList = { name, type: "movie" };
		const year = name.match(YEAR);
		if (year) {
			l.year = Number(year[1]);
			l.name = name.replace(YEAR, "").trim();
		}
		if (imdbId) {
			l.imdbId = imdbId;
			l.imdbStrict = true;
		} else if (link) {
			res.badLinks.push(link);
		}
		const t = parseTier(tier);
		if (t) {
			l.tier = t;
		} else if (tier) {
			res.badTiers.push(tier);
		}
		res.rows.push(l);
	}
	return res;
}
