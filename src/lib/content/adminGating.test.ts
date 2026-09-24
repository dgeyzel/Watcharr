// Visitors (isAdmin=false) never get edit controls or a numeric rating, the
// admin gets both.
import { render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";
import Poster from "@/lib/poster/Poster.svelte";
import PublicReview from "@/lib/content/PublicReview.svelte";
import { store } from "@/store.svelte";
import {
	MediaTypeE,
	UserPermission,
	type Media,
	type PrivateUser,
	type Watched,
} from "@/types";

const media: Media = {
	type: MediaTypeE.tmdbMovie,
	ids: { tmdb: 550 },
	name: "Fight Club",
	summary: "An insomniac office worker...",
	releaseDate: "1999-10-15",
	extPosterPath: "/fc.jpg",
};

function watched(over: Partial<Watched> = {}): Watched {
	return {
		id: 1,
		status: "FINISHED",
		rating: 7.5,
		thoughts: "Great, but the second half drags.",
		tier: "A",
		hidden: true,
		...over,
	} as Watched;
}

function signInAs(permissions: UserPermission | undefined) {
	store.userInfo = permissions
		? ({ id: 1, username: "admin", permissions } as unknown as PrivateUser)
		: undefined;
}

afterEach(() => signInAs(undefined));

describe("Poster gating", () => {
	it("visitors see the tier and status, no rating or edit controls", () => {
		signInAs(undefined);
		expect(store.isAdmin).toBe(false);
		const { container } = render(Poster, { media, watched: watched() });
		expect(screen.getByLabelText("Tier: A")).toBeInTheDocument();
		expect(container.querySelector("button.rating")).toBeNull();
		expect(container.textContent).not.toContain("7.5");
		// Status is shown, but can't be changed.
		const status = container.querySelector(".buttons button");
		expect(status).toHaveClass("interaction-disabled");
		// No hidden badge (and no extra admin details) for visitors.
		expect(container.querySelector(".hidden-badge")).toBeNull();
	});

	it("a non admin account is treated as a visitor", () => {
		signInAs(UserPermission.PERM_NONE);
		expect(store.isAdmin).toBe(false);
		const { container } = render(Poster, { media, watched: watched() });
		expect(container.querySelector("button.rating")).toBeNull();
	});

	it("the admin gets the rating, status buttons and hidden badge", () => {
		signInAs(UserPermission.PERM_ADMIN);
		expect(store.isAdmin).toBe(true);
		const { container } = render(Poster, { media, watched: watched() });
		expect(container.querySelector("button.rating")).toBeInTheDocument();
		expect(container.querySelector(".hidden-badge")).toBeInTheDocument();
		expect(screen.getByLabelText("Tier: A")).toBeInTheDocument();
	});
});

describe("PublicReview", () => {
	it("shows the tier, status and review, never the rating", () => {
		const { container } = render(PublicReview, { watched: watched() });
		expect(screen.getByLabelText("Tier: A")).toBeInTheDocument();
		expect(
			screen.getByText("Great, but the second half drags."),
		).toBeInTheDocument();
		expect(container.textContent).not.toContain("7.5");
		expect(container.querySelector("button, input, textarea")).toBeNull();
	});

	it("an untiered watched title says so", () => {
		render(PublicReview, { watched: watched({ tier: null }) });
		expect(screen.getByText("Watched but not yet rated")).toBeInTheDocument();
	});

	it("a planned title has no tier slot", () => {
		const { container } = render(PublicReview, {
			watched: watched({ status: "PLANNED", tier: "S", thoughts: "" }),
		});
		expect(screen.queryByLabelText(/^Tier:/)).toBeNull();
		expect(container.textContent).not.toContain("not yet rated");
		expect(container.textContent?.toLowerCase()).toContain("planned");
	});
});
