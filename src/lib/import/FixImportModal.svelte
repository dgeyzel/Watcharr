<!--
  Paste a TMDB, IMDb, Letterboxd or Rotten Tomatoes url, see what it
  matches on TMDB and confirm the right title.
 -->

<script lang="ts">
	import Modal from "@/lib/Modal.svelte";
	import SpinnerTiny from "@/lib/SpinnerTiny.svelte";
	import { req } from "@/lib/util/api";
	import { ReqerError } from "@/lib/util/fetch";
	import type { ResolveCandidate, ResolveResponse } from "@/types";

	interface Props {
		title?: string;
		desc?: string;
		/** Helps pick between results when the url doesn't say. */
		mediaTypeHint?: "movie" | "tv";
		confirmText?: string;
		onConfirm: (c: ResolveCandidate) => Promise<void> | void;
		onClose: () => void;
		/** Overridable for tests. */
		resolveUrl?: (
			url: string,
			mediaTypeHint?: "movie" | "tv",
		) => Promise<ResolveCandidate[]>;
	}

	let {
		title = "Fix by URL",
		desc = "Paste a TMDB, IMDb, Letterboxd or Rotten Tomatoes link.",
		mediaTypeHint = undefined,
		confirmText = "Import",
		onConfirm,
		onClose,
		resolveUrl = defaultResolveUrl,
	}: Props = $props();

	let url = $state("");
	let candidates: ResolveCandidate[] = $state([]);
	let selected: ResolveCandidate | undefined = $state();
	let error: string | undefined = $state();
	let loading = $state(false);
	let confirming = $state(false);

	async function defaultResolveUrl(u: string, hint?: "movie" | "tv") {
		const r = await req.post<ResolveResponse>("/import/resolve", {
			url: u,
			mediaTypeHint: hint,
		});
		return r.candidates ?? [];
	}

	function errorMessage(err: unknown, fallback: string) {
		if (err instanceof ReqerError && err.hasErrorInBody()) {
			return err.body.error;
		}
		return fallback;
	}

	async function find(ev: SubmitEvent) {
		ev.preventDefault();
		if (!url.trim() || loading) {
			return;
		}
		loading = true;
		error = undefined;
		candidates = [];
		selected = undefined;
		try {
			candidates = await resolveUrl(url.trim(), mediaTypeHint);
			if (candidates.length === 1) {
				selected = candidates[0];
			} else if (candidates.length === 0) {
				error = "No matching title found on TMDB.";
			}
		} catch (err) {
			console.error("FixImportModal: resolve failed", err);
			error = errorMessage(err, "Couldn't look up that link.");
		}
		loading = false;
	}

	async function confirm() {
		if (!selected || confirming) {
			return;
		}
		confirming = true;
		error = undefined;
		try {
			await onConfirm(selected);
		} catch (err) {
			console.error("FixImportModal: confirm failed", err);
			error = errorMessage(
				err,
				err instanceof Error && err.message ? err.message : "Import failed.",
			);
		}
		confirming = false;
	}
</script>

<Modal {title} {desc} {error} {onClose} maxWidth="600px">
	<form class="find" onsubmit={find}>
		<input
			type="url"
			aria-label="Title url"
			placeholder="https://www.imdb.com/title/tt0137523/"
			bind:value={url}
			disabled={loading || confirming}
		/>
		<button type="submit" disabled={!url.trim() || loading || confirming}>
			{#if loading}<SpinnerTiny />{:else}Find{/if}
		</button>
	</form>

	{#if candidates.length > 0}
		<ul class="candidates" aria-label="Matches">
			{#each candidates as c (`${c.mediaType}-${c.tmdbId}`)}
				<li>
					<button
						class="plain candidate"
						class:selected={selected === c}
						aria-pressed={selected === c}
						onclick={() => (selected = c)}
						disabled={confirming}
					>
						{#if c.posterPath}
							<img
								src="https://image.tmdb.org/t/p/w92{c.posterPath}"
								alt=""
								loading="lazy"
							/>
						{:else}
							<span class="no-poster"></span>
						{/if}
						<span class="info">
							<b>{c.title}</b>
							<span>
								{c.year ? c.year : "Unknown year"} · {c.mediaType === "tv"
									? "TV"
									: "Movie"}
							</span>
						</span>
					</button>
				</li>
			{/each}
		</ul>
		<div class="actions">
			<button onclick={confirm} disabled={!selected || confirming}>
				{#if confirming}<SpinnerTiny />{:else}{confirmText}{/if}
			</button>
		</div>
	{/if}
</Modal>

<style lang="scss">
	form.find {
		display: flex;
		gap: 8px;
		margin-top: 12px;

		input {
			flex: 1;
			min-width: 0;
		}

		button {
			width: max-content;
		}
	}

	ul.candidates {
		display: flex;
		flex-flow: column;
		gap: 6px;
		margin: 14px 0 0 0;
		padding: 0;
		list-style: none;
	}

	button.candidate {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		padding: 6px;
		border: 2px solid transparent;
		border-radius: 8px;
		text-align: left;
		color: $text-color;

		&:hover,
		&:focus-visible {
			background-color: $accent-color;
		}

		&.selected {
			border-color: $text-color;
		}

		img,
		.no-poster {
			width: 46px;
			height: 69px;
			flex-shrink: 0;
			border-radius: 4px;
			object-fit: cover;
			background-color: $accent-color;
		}

		.info {
			display: flex;
			flex-flow: column;
			gap: 2px;
		}
	}

	.actions {
		display: flex;
		justify-content: flex-end;
		margin-top: 12px;

		button {
			width: max-content;
		}
	}
</style>
