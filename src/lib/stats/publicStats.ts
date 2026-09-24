/**
 * Public stats (GET /api/public/stats), computed by the server from visible
 * items only. No numeric ratings.
 */
import { noAuthReq } from "@/lib/util/api";
import { TIERS, type Tier } from "@/types";
import type { Bar } from "./bars";

export interface PublicStats {
	totals: { titles: number; movies: number; shows: number };
	byStatus: { finished: number; watching: number; planned: number };
	tiers: Record<Tier, number> & { unrated: number };
	addedPerMonth: { month: string; count: number }[];
	byDecade: { decade: number; count: number }[];
	topGenres: { name: string; count: number }[];
	tags: { id: number; name: string; count: number }[];
	finishedMovieHours: number;
}

export async function getPublicStats(): Promise<PublicStats> {
	return await noAuthReq.get<PublicStats>("/public/stats");
}

const MONTHS = [
	"Jan",
	"Feb",
	"Mar",
	"Apr",
	"May",
	"Jun",
	"Jul",
	"Aug",
	"Sep",
	"Oct",
	"Nov",
	"Dec",
];

/** "2026-09" -> "Sep 2026" */
export function monthLabel(month: string): string {
	const [y, m] = month.split("-").map(Number);
	return `${MONTHS[(m ?? 1) - 1] ?? month} ${y}`;
}

export function tierBars(s: PublicStats): Bar[] {
	return [
		...TIERS.map((g) => ({
			label: g,
			value: s.tiers[g] ?? 0,
			barClass: `tier-${g.toLowerCase()}`,
		})),
		{ label: "Watched but not yet rated", value: s.tiers.unrated ?? 0 },
	];
}

export function monthBars(s: PublicStats): Bar[] {
	return s.addedPerMonth.map((m) => ({
		label: monthLabel(m.month),
		value: m.count,
	}));
}

export function decadeBars(s: PublicStats): Bar[] {
	return s.byDecade.map((d) => ({ label: `${d.decade}s`, value: d.count }));
}

export function genreBars(s: PublicStats): Bar[] {
	return s.topGenres.map((g) => ({ label: g.name, value: g.count }));
}

export function tagBars(s: PublicStats): Bar[] {
	return s.tags.map((t) => ({ label: t.name, value: t.count }));
}
