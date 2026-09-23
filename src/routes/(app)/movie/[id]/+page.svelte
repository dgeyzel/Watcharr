<script lang="ts">
	import PersonPoster from "@/lib/poster/PersonPoster.svelte";
	import Spinner from "@/lib/Spinner.svelte";
	import HorizontalList from "@/lib/HorizontalList.svelte";
	import { req, updateWatched } from "@/lib/util/api";
	import { getPublicMedia } from "@/lib/public/api";
	import { ReqerError } from "@/lib/util/fetch";
	import PublicReview from "@/lib/content/PublicReview.svelte";
	import HiddenToggle from "@/lib/content/HiddenToggle.svelte";
	import NotFound from "@/lib/content/NotFound.svelte";
	import { store } from "@/store.svelte";
	import type {
		Media,
		TMDBContentCredits,
		TMDBContentCreditsCrew,
		WatchedStatus,
	} from "@/types";
	import { getTopCrew } from "@/lib/util/helpers.js";
	import Activity from "@/lib/Activity.svelte";
	import Title from "@/lib/content/Title.svelte";
	import ProvidersList from "@/lib/content/ProvidersList.svelte";
	import Icon from "@/lib/Icon.svelte";
	import SimilarContent from "@/lib/content/SimilarContent.svelte";
	import Error from "@/lib/Error.svelte";
	import FollowedThoughts from "@/lib/content/FollowedThoughts.svelte";
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
	import Genres from "@/lib/content/Genres.svelte";

	let { data } = $props();

	let movie: Media | undefined = $state();
	let pageError: unknown | undefined = $state();
	// Visitor asked for a title that is not visible.
	let notFound = $state(false);

	$effect(() => {
		(async () => {
			try {
				movie = undefined;
				pageError = undefined;
				notFound = false;
				if (!data.movieId) {
					return;
				}
				if (!store.isAdmin) {
					// Visitors only see titles on the owner's (visible) list.
					try {
						movie = await getPublicMedia("movie", data.movieId);
					} catch (err) {
						if (err instanceof ReqerError && err.response?.status === 404) {
							notFound = true;
							movie = { ids: {} };
							return;
						}
						throw err;
					}
					return;
				}
				const resp = await req.get<Media>(`/content/movie/${data.movieId}`, {
					params: { region: store.userSettings?.country },
				});
				if (resp) {
					movie = resp;
				} else {
					movie = undefined;
				}
			} catch (err) {
				movie = undefined;
				pageError = err;
			}
		})();
	});

	async function getMovieCredits() {
		const credits = await req.get<
			TMDBContentCredits & { topCrew: TMDBContentCreditsCrew[] }
		>(`/content/movie/${data.movieId}/credits`);
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
			if (!data.movieId) {
				console.error("contentChanged: no movieId");
				return false;
			}
			if (!movie) {
				console.error("contentChanged: no movie");
				return false;
			}
			movie.watched = await updateWatched(movie.watched, {
				contentId: data.movieId,
				contentType: "movie",
				status: newStatus,
				rating: newRating,
				thoughts: newThoughts,
				pinned: pinned,
			});
			return true;
		} catch {
			return false;
		}
	}
</script>

<svelte:head>
	<title>{movie?.name ? `${movie.name} - ` : ""}Movie</title>
</svelte:head>

{#if pageError}
	<Error pretty="Failed to load movie!" error={pageError} />
{:else if !movie}
	<Spinner />
{:else if !notFound && Object.keys(movie).length > 0}
	{#if movie?.extBackdropPath}
		<PageBackdrop
			src={"https://www.themoviedb.org/t/p/w1920_and_h800_multi_faces" +
				movie.extBackdropPath}
		/>
	{/if}
	<div>
		<div class="content">
			<div class="details-wrap">
				<div class="details-container">
					{#if movie.extPosterPath}
						<PosterImage
							src={"https://image.tmdb.org/t/p/w500" + movie.extPosterPath}
						/>
					{/if}

					<div class="details">
						<Title
							title={movie.name}
							homepage={movie.homepage}
							releaseDate={movie.releaseDate
								? new Date(movie.releaseDate)
								: undefined}
							voteAverage={movie.rating}
							voteCount={movie.ratingCount}
						/>

						<span class="quick-info">
							{#if movie.runtime}
								<span>{movie.runtime} min</span>
							{/if}

							<Genres genres={movie.genres} />
						</span>

						<ExpandableText text={movie.summary} style="margin-bottom: 18px;" />

						<div class="btns">
							<ViewTrailerButton videos={movie.videos} />
							{#if store.isAdmin && movie.watched}
								<div class="other-side">
									<AddToTagButton watchedItem={movie.watched} />
									<button
										onclick={() => {
											if (movie?.watched?.pinned) {
												contentChanged(undefined, undefined, undefined, false);
											} else {
												contentChanged(undefined, undefined, undefined, true);
											}
										}}
										use:tooltip={{
											text: `${movie.watched?.pinned ? "Unpin from" : "Pin to"} top of list`,
											pos: "bot",
										}}
									>
										<Icon i={movie.watched?.pinned ? "unpin" : "pin"} wh={19} />
									</button>
									<WatchedDeleteBtn
										watchedId={movie.watched.id}
										mediaName={movie.name}
										onDelete={() => {
											if (movie) {
												movie.watched = undefined;
											}
										}}
									/>
								</div>
							{/if}
						</div>

						{#if movie.providers}
							<ProvidersList
								providers={movie.providers}
								fullListLink={movie.providersFullListLink}
								fullListLinkText="JustWatch"
							/>
						{/if}
					</div>
				</div>
			</div>

			{#if store.isAdmin}
				<MyReview
					watched={movie.watched}
					contentTitle={movie.name}
					onRatingChanged={(n) => contentChanged(undefined, n)}
					onStatusChanged={(n) => contentChanged(n)}
					onThoughtsChanged={(newThoughts) => {
						return contentChanged(undefined, undefined, newThoughts);
					}}
				/>
				{#if movie.watched}
					<HiddenToggle bind:watched={movie.watched} />
				{/if}
			{:else if movie.watched}
				<PublicReview watched={movie.watched} />
			{/if}
		</div>

		<!-- Cast, similar titles and activity are admin only (visitors can't open
		  person pages or titles that aren't on the list). -->
		<div class="page">
			{#if store.isAdmin}
				{#if data.movieId}
					<FollowedThoughts mediaType="movie" mediaId={data.movieId} />
				{/if}

				{#await getMovieCredits()}
					<Spinner />
				{:then credits}
					<!-- TODO make this nicer  -->
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

				{#if movie.similar}
					<SimilarContent similar={movie.similar} />
				{/if}

				{#if movie.watched}
					<Activity
						activity={movie.watched.activity}
						onRemoved={(a) => activityRemovedHook(movie?.watched, a)}
					/>
				{/if}
			{/if}
		</div>
	</div>
{:else}
	<NotFound what="Movie" />
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
