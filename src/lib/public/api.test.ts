import { describe, expect, it } from "vitest";
import { MediaTypeE } from "@/types";
import {
	publicContentToMedia,
	publicWatchedToMedia,
	toPublicListParams,
	type PublicContent,
	type PublicWatched,
} from "./api";

const watched: PublicWatched = {
	mediaType: "movie",
	tmdbId: 550,
	title: "Fight Club",
	posterPath: "/fc.jpg",
	releaseDate: "1999-10-15T00:00:00Z",
	status: "FINISHED",
	review: "Great.",
	tags: [{ id: 1, name: "Favourites", color: "#fff", bgColor: "#000" }],
	createdAt: "2026-01-01T00:00:00Z",
	updatedAt: "2026-01-02T00:00:00Z",
};

describe("publicWatchedToMedia", () => {
	it("maps to a Media with a read only watched entry and no rating", () => {
		const m = publicWatchedToMedia(watched);
		expect(m.type).toBe(MediaTypeE.tmdbMovie);
		expect(m.ids.tmdb).toBe(550);
		expect(m.name).toBe("Fight Club");
		expect(m.extPosterPath).toBe("/fc.jpg");
		expect(m.watched?.status).toBe("FINISHED");
		expect(m.watched?.thoughts).toBe("Great.");
		expect(m.watched?.id).toBe(0);
		expect(m.watched?.tags?.[0].name).toBe("Favourites");
		expect(m.rating).toBeUndefined();
		expect(m.watched?.rating).toBeUndefined();
	});

	it("maps tv", () => {
		expect(publicWatchedToMedia({ ...watched, mediaType: "tv" }).type).toBe(
			MediaTypeE.tmdbShow,
		);
	});
});

describe("publicContentToMedia", () => {
	it("maps details without any vote score", () => {
		const c: PublicContent = {
			mediaType: "tv",
			tmdbId: 1396,
			title: "Breaking Bad",
			overview: "o",
			posterPath: "/bb.jpg",
			backdropPath: "",
			genres: [{ id: 18, name: "Drama" }],
			homepage: "",
			releaseDate: null,
			releaseDateLast: null,
			runtime: 0,
			status: "Ended",
			videos: [],
			seasons: [],
			providers: [],
			providersFullListLink: "",
		};
		const m = publicContentToMedia(c);
		expect(m.type).toBe(MediaTypeE.tmdbShow);
		expect(m.extBackdropPath).toBeUndefined();
		expect(m.rating).toBeUndefined();
		expect(m.ratingCount).toBeUndefined();
		expect(m.watched).toBeUndefined();
	});
});

describe("toPublicListParams", () => {
	it("keeps supported params and drops admin only sorts", () => {
		expect(
			toPublicListParams({
				page: 2,
				sort: "RATING",
				sortDir: "asc",
				status: "finished",
				type: "tv",
				other: "x",
			}),
		).toEqual({ page: 2, sortDir: "asc", status: "finished", type: "tv" });
		expect(toPublicListParams({ sort: "ALPHA" })).toEqual({ sort: "ALPHA" });
	});
});
