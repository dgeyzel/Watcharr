<script lang="ts">
	import Activity from "@/lib/Activity.svelte";
	import Error from "@/lib/Error.svelte";
	import HorizontalList from "@/lib/HorizontalList.svelte";
	import Icon from "@/lib/Icon.svelte";
	import PersonPoster from "@/lib/poster/PersonPoster.svelte";
	import SeasonsList from "@/lib/season/SeasonsList.svelte";
	import Spinner from "@/lib/Spinner.svelte";
	import ProvidersList from "@/lib/content/ProvidersList.svelte";
	import SimilarContent from "@/lib/content/SimilarContent.svelte";
	import Title from "@/lib/content/Title.svelte";
	import {
		req,
		updateWatched,
		type UpdateWatchedOptions,
	} from "@/lib/util/api";
	import { getTopCrew } from "@/lib/util/helpers.js";
	import { store } from "@/store.svelte.js";
	import type {
		Media,
		TMDBContentCredits,
		TMDBContentCreditsCrew,
		WatchedStatus,
	} from "@/types";
	import { getPublicMedia } from "@/lib/public/api";
	import { ReqerError } from "@/lib/util/fetch";
	import PublicReview from "@/lib/content/PublicReview.svelte";
	import HiddenToggle from "@/lib/content/HiddenToggle.svelte";
	import NotFound from "@/lib/content/NotFound.svelte";
	import tooltip from "@/lib/actions/tooltip.js";
	import AddToTagButton from "@/lib/tag/AddToTagButton.svelte";
	import PageBackdrop from "@/lib/generic/PageBackdrop.svelte";
	import MyReview from "@/lib/content/MyReview.svelte";
	import ViewTrailerButton from "@/lib/content/ViewTrailerButton.svelte";
	import PosterImage from "@/lib/content/PosterImage.svelte";
	import ExpandableText from "@/lib/content/ExpandableText.svelte";
	import WatchedDeleteBtn from "@/lib/content/WatchedDeleteBtn.svelte";
	import TopCrewList from "@/lib/content/TopCrewList.svelte";
	import { activityRemovedHook } from "@/lib/activity.js";
	import CountAsPlayModal from "@/lib/watched/CountAsPlayModal.svelte";
	import { createSignal, type Signal } from "@/lib/util/signal.js";
	import Genres from "@/lib/content/Genres.svelte";
	import { MediaStatusShow } from "@/lib/types/mediaStatus.js";

	let { data } = $props();

	let show: Media | undefined = $state();
	let pageError: unknown | undefined = $state();
	// Visitor asked for a title that is not visible.
	let notFound = $state(false);
	let countAsPlayModalSignal: Signal<boolean> | undefined = $state();

	$effect(() => {
		(async () => {
			try {
				show = undefined;
				pageError = undefined;
				notFound = false;
				if (!data.tvId) {
					return;
				}
				if (!store.isAdmin) {
					// Visitors only see titles on the owner's (visible) list.
					try {
						show = await getPublicMedia("tv", data.tvId);
					} catch (err) {
						if (err instanceof ReqerError && err.response?.status === 404) {
							notFound = true;
							show = { ids: {} };
							return;
						}
						throw err;
					}
					return;
				}
				const resp = await req.get<Media>(`/content/tv/${data.tvId}`, {
					params: { region: store.userSettings?.country },
				});
				if (resp) {
					show = resp;
				} else {
					show = undefined;
				}
			} catch (err) {
				show = undefined;
				pageError = err;
			}
		})();
	});

	async function getTvCredits() {
		const credits = await req.get<
			TMDBContentCredits & { topCrew: TMDBContentCreditsCrew[] }
		>(`/content/tv/${data.tvId}/credits`);
		if (credits.crew?.length > 0) {
			credits.topCrew = getTopCrew(credits.crew);
		}
		return credits;
	}

	async function contentChanged(
		newStatus?: WatchedStatus,
		newRating?: number,
		newThoughts?: string,
		pinned?: boolean,
	): Promise<boolean> {
		try {
			if (!data.tvId) {
				console.error("contentChanged: no tvId");
				return false;
			}
			if (!show) {
				console.error("contentChanged: no show");
				return false;
			}
			const reqOpts: UpdateWatchedOptions = {
				contentId: data.tvId,
				contentType: "tv",
				status: newStatus,
				rating: newRating,
				thoughts: newThoughts,
				pinned: pinned,
			};
			// Series differ from other media in that people are likely to go
			// back out of the 'FINISHED' state when a new season releases
			// while they watch it, then go back to 'FINISHED' after only
			// watching the new season. Because of that use case, for series,
			// we will ask the user if any 'FINISHED' statuses set when plays>1
			// should count as a play..
			if (show.watched?.plays && newStatus == "FINISHED") {
				countAsPlayModalSignal = createSignal<boolean>();
				const allow = await countAsPlayModalSignal.promise;
				countAsPlayModalSignal = undefined;
				if (!allow) {
					reqOpts.letCountAsPlay = false;
				}
			}
			show.watched = await updateWatched(show.watched, reqOpts);
			return true;
		} catch {
			return false;
		}
	}
</script>

<svelte:head>
	<title>{show?.name ? `${show.name} - ` : ""}Show</title>
</svelte:head>

{#if pageError}
	<Error pretty="Failed to load tv show!" error={pageError} />
{:else if !show}
	<Spinner />
{:else if !notFound && Object.keys(show).length > 0}
	{#if show?.extBackdropPath}
		<PageBackdrop
			src={"https://www.themoviedb.org/t/p/w1920_and_h800_multi_faces" +
				show.extBackdropPath}
		/>
	{/if}
	<div>
		<div class="content">
			<div class="details-wrap">
				<div class="details-container">
					{#if show.extPosterPath}
						<PosterImage
							src={"https://image.tmdb.org/t/p/w500" + show.extPosterPath}
						/>
					{/if}

					<div class="details">
						<Title
							title={show.name}
							homepage={show.homepage}
							releaseDate={show.releaseDate
								? new Date(show.releaseDate)
								: undefined}
							endDate={(show.status === MediaStatusShow.Ended ||
								show.status === MediaStatusShow.Canceled) &&
							show.releaseDateLast
								? new Date(show.releaseDateLast)
								: undefined}
							voteAverage={show.rating}
							voteCount={show.ratingCount}
						/>

						<span class="quick-info">
							<Genres genres={show.genres} />
						</span>

						<ExpandableText text={show.summary} style="margin-bottom: 18px;" />

						<div class="btns">
							<ViewTrailerButton videos={show.videos} />
							{#if store.isAdmin && show.watched}
								<div class="other-side">
									<AddToTagButton watchedItem={show.watched} />
									<button
										onclick={() => {
											if (show?.watched?.pinned) {
												contentChanged(undefined, undefined, undefined, false);
											} else {
												contentChanged(undefined, undefined, undefined, true);
											}
										}}
										use:tooltip={{
											text: `${show.watched?.pinned ? "Unpin from" : "Pin to"} top of list`,
											pos: "bot",
										}}
									>
										<Icon i={show.watched?.pinned ? "unpin" : "pin"} wh={19} />
									</button>
									<WatchedDeleteBtn
										watchedId={show.watched.id}
										mediaName={show.name}
										onDelete={() => {
											if (show) {
												show.watched = undefined;
											}
										}}
									/>
								</div>
							{/if}
						</div>

						{#if show.providers}
							<ProvidersList
								providers={show.providers}
								fullListLink={show.providersFullListLink}
								fullListLinkText="JustWatch"
							/>
						{/if}
					</div>
				</div>
			</div>

			{#if store.isAdmin}
				<MyReview
					watched={show.watched}
					contentTitle={show.name}
					onRatingChanged={(n) => contentChanged(undefined, n)}
					onStatusChanged={(n) => contentChanged(n)}
					onThoughtsChanged={(newThoughts) => {
						return contentChanged(undefined, undefined, newThoughts);
					}}
				/>
				{#if show.watched}
					<HiddenToggle bind:watched={show.watched} />
				{/if}
			{:else if show.watched}
				<PublicReview watched={show.watched} />
			{/if}
		</div>

		{#if countAsPlayModalSignal}
			<CountAsPlayModal onDecision={countAsPlayModalSignal} />
		{/if}

		<!-- Cast, similar titles, activity and seasons are admin only (visitors
		  can't open person pages or titles that aren't on the list). -->
		<div class="page">
			{#if store.isAdmin}
				{#await getTvCredits()}
					<Spinner />
				{:then credits}
					{#if credits.topCrew?.length > 0}
						<TopCrewList topCrew={credits.topCrew} />
					{/if}

					{#if credits.cast?.length > 0}
						<HorizontalList title="Cast">
							{#each credits.cast?.slice(0, 50) as cast (cast.credit_id)}
								<PersonPoster
									id={cast.id}
									name={cast.name}
									path={cast.profile_path}
									role={cast.character}
									zoomOnHover={false}
								/>
							{/each}
						</HorizontalList>
					{/if}
				{:catch err}
					<Error error={err} pretty="Failed to load cast!" />
				{/await}

				{#if show.similar}
					<SimilarContent similar={show.similar} />
				{/if}

				{#if show.watched}
					<Activity
						activity={show.watched.activity}
						onRemoved={(a) => activityRemovedHook(show?.watched, a)}
					/>
				{/if}

				{#if data?.tvId && show.seasons}
					<SeasonsList
						tvId={data.tvId}
						seasons={show.seasons}
						watchedItem={show.watched}
						lastViewedSeason={show.watched?.lastViewedSeason}
						lastViewedSeasonChanged={(wid, lvs) => {
							if (show?.watched && show.watched.id === wid) {
								show.watched.lastViewedSeason = lvs;
							}
						}}
					/>
				{/if}
			{/if}
		</div>
	</div>
{:else}
	<NotFound what="Show" />
{/if}

<style lang="scss">
	@use "../../../../lib/content/page.scss";

	.content {
		position: relative;
		color: white;

		.details-container .details {
			.quick-info {
				display: flex;
				gap: 10px;
				margin-bottom: 8px;
			}

			.btns {
				display: flex;
				flex-flow: row;
				flex-wrap: wrap;
				gap: 8px;
				margin-top: auto;

				button {
					max-width: fit-content;
					overflow: hidden;
					animation: 50ms cubic-bezier(0.86, 0, 0.07, 1) forwards otherbtn;
					white-space: nowrap;
					gap: 6px;
					justify-content: flex-start;
					font-size: 14px;

					@keyframes otherbtn {
						from {
							width: 0px;
						}
						to {
							width: 100%;
						}
					}
				}

				.other-side {
					display: flex;
					flex-flow: row;
					gap: 8px;

					@media screen and (min-width: 900px) {
						margin-left: auto;
					}
				}
			}
		}
	}

	.page {
		display: flex;
		flex-flow: column;
		align-items: center;
		margin-left: auto;
		margin-right: auto;
		gap: 30px;
		padding: 20px 50px;
		max-width: 1200px;

		@media screen and (max-width: 500px) {
			padding: 20px;
		}
	}
</style>
