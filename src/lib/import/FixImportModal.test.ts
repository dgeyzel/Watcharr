import { fireEvent, render, screen } from "@testing-library/svelte";
import { describe, expect, it, vi } from "vitest";
import FixImportModal from "./FixImportModal.svelte";
import { ReqerError } from "@/lib/util/fetch";
import type { ResolveCandidate } from "@/types";

const theThing: ResolveCandidate = {
	tmdbId: 1091,
	mediaType: "movie",
	title: "The Thing",
	year: 1982,
	posterPath: "/thing.jpg",
};
const theThing2011: ResolveCandidate = {
	tmdbId: 60935,
	mediaType: "movie",
	title: "The Thing",
	year: 2011,
	posterPath: "",
};

async function find(url: string) {
	await fireEvent.input(screen.getByRole("textbox", { name: "Title url" }), {
		target: { value: url },
	});
	await fireEvent.click(screen.getByRole("button", { name: "Find" }));
}

describe("FixImportModal", () => {
	it("resolves the pasted url and confirms the single match", async () => {
		const resolveUrl = vi.fn().mockResolvedValue([theThing]);
		const onConfirm = vi.fn();
		render(FixImportModal, {
			resolveUrl,
			onConfirm,
			onClose: () => {},
			mediaTypeHint: "movie",
		});
		await find(" https://www.imdb.com/title/tt0084787/ ");
		expect(resolveUrl).toHaveBeenCalledWith(
			"https://www.imdb.com/title/tt0084787/",
			"movie",
		);
		const match = await screen.findByRole("button", { name: /The Thing/ });
		expect(match).toHaveTextContent("1982");
		expect(match).toHaveAttribute("aria-pressed", "true");
		expect(match.querySelector("img")).toHaveAttribute(
			"src",
			"https://image.tmdb.org/t/p/w92/thing.jpg",
		);
		await fireEvent.click(screen.getByRole("button", { name: "Import" }));
		expect(onConfirm).toHaveBeenCalledWith(theThing);
	});

	it("makes the admin pick when there are several candidates", async () => {
		const onConfirm = vi.fn();
		render(FixImportModal, {
			resolveUrl: vi.fn().mockResolvedValue([theThing, theThing2011]),
			onConfirm,
			onClose: () => {},
		});
		await find("https://www.rottentomatoes.com/m/the_thing");
		const options = await screen.findAllByRole("button", { name: /The Thing/ });
		expect(options).toHaveLength(2);
		const importBtn = screen.getByRole("button", { name: "Import" });
		expect(importBtn).toBeDisabled();
		await fireEvent.click(options[1]);
		expect(options[1]).toHaveTextContent("2011");
		await fireEvent.click(importBtn);
		expect(onConfirm).toHaveBeenCalledWith(theThing2011);
	});

	it("shows the server's error message", async () => {
		const err = new ReqerError("request failed with 422", {
			error: "couldn't read that page, paste an IMDb or TMDB link instead",
		});
		render(FixImportModal, {
			resolveUrl: vi.fn().mockRejectedValue(err),
			onConfirm: vi.fn(),
			onClose: () => {},
		});
		await find("https://letterboxd.com/film/blocked/");
		expect(
			await screen.findByText(/paste an IMDb or TMDB link instead/),
		).toBeInTheDocument();
		expect(screen.queryByRole("button", { name: "Import" })).toBeNull();
	});

	it("shows an error when nothing matches", async () => {
		render(FixImportModal, {
			resolveUrl: vi.fn().mockResolvedValue([]),
			onConfirm: vi.fn(),
			onClose: () => {},
		});
		await find("https://www.imdb.com/title/tt9999999/");
		expect(await screen.findByText(/No matching title/)).toBeInTheDocument();
	});

	it("shows why confirming failed and lets the admin retry", async () => {
		const onConfirm = vi
			.fn()
			.mockRejectedValueOnce(new Error("Import failed, try another link."))
			.mockResolvedValueOnce(undefined);
		render(FixImportModal, {
			resolveUrl: vi.fn().mockResolvedValue([theThing]),
			onConfirm,
			onClose: () => {},
		});
		await find("https://www.themoviedb.org/movie/1091");
		await screen.findByRole("button", { name: /The Thing/ });
		await fireEvent.click(screen.getByRole("button", { name: "Import" }));
		expect(
			await screen.findByText("Import failed, try another link."),
		).toBeInTheDocument();
		await fireEvent.click(screen.getByRole("button", { name: "Import" }));
		expect(onConfirm).toHaveBeenCalledTimes(2);
	});

	it("closes", async () => {
		const onClose = vi.fn();
		render(FixImportModal, {
			resolveUrl: vi.fn(),
			onConfirm: vi.fn(),
			onClose,
		});
		// The modal's close button is the only button without text.
		const close = screen
			.getAllByRole("button")
			.find((b) => !b.textContent?.trim());
		await fireEvent.click(close!);
		expect(onClose).toHaveBeenCalled();
	});
});
