/**
 * Client for the public (visitor) api, `/api/public/*`.
 *
 * Visitor pages reuse the admin components (posters, detail pages), so public
 * responses are adapted into the same `Media`/`Watched` shapes here. Public
 * responses never contain numeric ratings, so the adapted objects don't
 * either.
 */
import { noAuthReq } from "@/lib/util/api";
import {
	MediaTypeE,
	type Media,
	type MediaGenre,
	type MediaProvider,
	type MediaSeason,
	type MediaVideo,
	type PaginationResponse,
	type Tag,
	type Watched,
	type WatchedStatus,
} from "@/types";

export type PublicMediaType = "movie" | "tv";

export interface PublicTag {
	id: number;
	name: string;
	color: string;
	bgColor: string;
}

export interface PublicWatched {
	mediaType: PublicMediaType;
	tmdbId: number;
	title: string;
	posterPath: string;
	releaseDate: string | null;
	status: WatchedStatus;
	review: string;
	tags: PublicTag[];
	createdAt: string;
	updatedAt: string;
}

export interface PublicWatchedPage {
	page: number;
	limit: number;
	totalPages: number;
	totalResults: number;
	results: PublicWatched[];
}

export interface PublicContent {
	mediaType: PublicMediaType;
	tmdbId: number;
	title: string;
	overview: string;
	posterPath: string;
	backdropPath: string;
	genres: MediaGenre[];
	homepage: string;
	releaseDate: string | null;
	releaseDateLast: string | null;
	runtime: number;
	status: string;
	videos: MediaVideo[];
	seasons: MediaSeason[];
	providers: MediaProvider[];
	providersFullListLink: string;
}

export interface PublicOwner {
	username: string;
	bio: string;
	avatar: { path: string; blurHash: string } | null;
}

/** Sorts the public list supports. Others fall back to date added. */
const PUBLIC_SORTS = ["DATEADDED", "ALPHA", "DATERELEASED"];

/**
 * Keeps only the sort/filter params the public list understands.
 */
export function toPublicListParams(
	params: Record<string, unknown>,
): Record<string, unknown> {
	const out: Record<string, unknown> = {};
	for (const [k, v] of Object.entries(params)) {
		if (k === "sort") {
			if (PUBLIC_SORTS.includes(String(v))) out.sort = v;
		} else if (
			["page", "limit", "sortDir", "type", "status", "q"].includes(k)
		) {
			out[k] = v;
		}
	}
	return out;
}

function toMediaType(t: PublicMediaType): MediaTypeE {
	return t === "tv" ? MediaTypeE.tmdbShow : MediaTypeE.tmdbMovie;
}

function toTag(t: PublicTag): Tag {
	return { ...t, createdAt: "", updatedAt: "", deletedAt: "" };
}

/**
 * Adapt a public watched entry to the `Watched` shape our components use.
 * `id` is 0 because visitors never get watched ids (nothing can be updated).
 */
export function publicWatchedToWatched(w: PublicWatched): Watched {
	return {
		id: 0,
		createdAt: w.createdAt,
		updatedAt: w.updatedAt,
		status: w.status,
		thoughts: w.review,
		pinned: false,
		tags: w.tags.map(toTag),
	};
}

export function publicWatchedToMedia(w: PublicWatched): Media {
	return {
		type: toMediaType(w.mediaType),
		ids: { tmdb: w.tmdbId },
		name: w.title,
		extPosterPath: w.posterPath || undefined,
		releaseDate: w.releaseDate ?? undefined,
		watched: publicWatchedToWatched(w),
	};
}

export function publicContentToMedia(
	c: PublicContent,
	w?: PublicWatched,
): Media {
	return {
		type: toMediaType(c.mediaType),
		ids: { tmdb: c.tmdbId },
		name: c.title,
		summary: c.overview,
		extPosterPath: c.posterPath || undefined,
		extBackdropPath: c.backdropPath || undefined,
		genres: c.genres,
		homepage: c.homepage || undefined,
		releaseDate: c.releaseDate ?? undefined,
		releaseDateLast: c.releaseDateLast ?? undefined,
		runtime: c.runtime || undefined,
		status: c.status || undefined,
		videos: c.videos,
		seasons: c.seasons,
		providers: c.providers,
		providersFullListLink: c.providersFullListLink || undefined,
		watched: w ? publicWatchedToWatched(w) : undefined,
	};
}

/**
 * Get a page of the owner's visible list, adapted for `paginatedLoader`.
 * `path` is `/public/watched` or `/public/tag/{id}/watched`.
 */
export async function getPublicWatchedPage(
	path: string,
	params: Record<string, unknown>,
	signal?: AbortSignal,
): Promise<PaginationResponse<Media, undefined>> {
	const r = await noAuthReq.get<PublicWatchedPage>(path, {
		params: toPublicListParams(params),
		signal,
	});
	return {
		page: r.page,
		limit: r.limit,
		totalPages: r.totalPages,
		totalResults: r.totalResults,
		results: r.results.map(publicWatchedToMedia),
	};
}

/**
 * Get a visible title's details and the owner's entry for it. Rejects (404)
 * when the title isn't visible.
 */
export async function getPublicMedia(
	type: PublicMediaType,
	tmdbId: number,
): Promise<Media> {
	const [c, w] = await Promise.all([
		noAuthReq.get<PublicContent>(`/public/content/${type}/${tmdbId}`),
		noAuthReq.get<PublicWatched>(`/public/watched/${type}/${tmdbId}`),
	]);
	return publicContentToMedia(c, w);
}

export async function getPublicTags(): Promise<Tag[]> {
	const ts = await noAuthReq.get<PublicTag[]>("/public/tags");
	return ts.map(toTag);
}

export async function getPublicTag(id: number): Promise<Tag> {
	return toTag(await noAuthReq.get<PublicTag>(`/public/tag/${id}`));
}
