<!-- Public stats, computed from visible titles only (the admin's private
  stats stay on /profile). -->
<script lang="ts">
	import Error from "@/lib/Error.svelte";
	import Spinner from "@/lib/Spinner.svelte";
	import Stat from "@/lib/stats/Stat.svelte";
	import Stats from "@/lib/stats/Stats.svelte";
	import BarChart from "@/lib/stats/BarChart.svelte";
	import {
		decadeBars,
		genreBars,
		getPublicStats,
		tierBars,
		monthBars,
		tagBars,
	} from "@/lib/stats/publicStats";

	const statsReq = getPublicStats();
</script>

<svelte:head>
	<title>Stats</title>
</svelte:head>

<div class="content">
	<div class="inner">
		<h2>Stats</h2>
		{#await statsReq}
			<Spinner />
		{:then s}
			{#if s.totals.titles === 0}
				<p class="empty">No titles yet, check back soon.</p>
			{:else}
				<Stats>
					<Stat name="Titles" value={s.totals.titles} large />
					<Stat name="Movies" value={s.totals.movies} large />
					<Stat name="Shows" value={s.totals.shows} large />
					<Stat name="Finished" value={s.byStatus.finished} large />
					<Stat name="Watching" value={s.byStatus.watching} large />
					<Stat name="Planned" value={s.byStatus.planned} large />
					<Stat
						name="Hours of finished movies"
						value={s.finishedMovieHours}
						large
					/>
				</Stats>
				<div class="charts">
					<BarChart title="Tiers" bars={tierBars(s)} />
					<BarChart title="Added per month" bars={monthBars(s)} />
					<BarChart title="Release decades" bars={decadeBars(s)} />
					<BarChart title="Top genres" bars={genreBars(s)} />
					{#if s.tags.length > 0}
						<BarChart title="Tags" bars={tagBars(s)} />
					{/if}
				</div>
			{/if}
		{:catch err}
			<Error pretty="Failed to load stats!" error={err} />
		{/await}
	</div>
</div>

<style lang="scss">
	.content {
		display: flex;
		width: 100%;
		justify-content: center;
		padding: 0 16px 30px 16px;

		.inner {
			width: 100%;
			max-width: 760px;
			min-width: 0;
		}
	}

	.charts {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: 28px;
		margin-top: 28px;
	}

	.empty {
		margin-top: 15px;
		color: $text-color-accent;
	}
</style>
