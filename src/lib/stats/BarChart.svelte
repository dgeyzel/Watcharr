<!-- Simple horizontal bar chart built from a table, so the chart is its own
  accessible text fallback (label and value are real table cells, the bar is
  decoration). No charting dependency. -->
<script lang="ts">
	import type { Bar } from "./bars";

	interface Props {
		title: string;
		bars: Bar[];
		emptyText?: string;
	}

	let { title, bars, emptyText = "Nothing to show yet." }: Props = $props();

	const max = $derived(Math.max(0, ...bars.map((b) => b.value)));
</script>

<section class="chart">
	<h3>{title}</h3>
	{#if bars.length === 0 || max === 0}
		<p class="empty">{emptyText}</p>
	{:else}
		<table>
			<caption class="sr-only">{title}</caption>
			<thead class="sr-only">
				<tr>
					<th scope="col">Name</th>
					<th scope="col">Count</th>
				</tr>
			</thead>
			<tbody>
				{#each bars as b (b.label)}
					<tr>
						<th scope="row">{b.label}</th>
						<td>
							<span class="bar-track" aria-hidden="true">
								<span
									class="bar {b.barClass ?? ''}"
									style:width="{max ? (b.value / max) * 100 : 0}%"
								></span>
							</span>
							<span class="value">{b.value}</span>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</section>

<style lang="scss">
	.chart {
		width: 100%;

		h3 {
			margin-bottom: 8px;
		}
	}

	table {
		width: 100%;
		border-collapse: collapse;
	}

	th[scope="row"] {
		width: 1%;
		padding: 3px 10px 3px 0;
		text-align: left;
		font-weight: normal;
		white-space: nowrap;
		max-width: 45vw;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	td {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 3px 0;
	}

	.bar-track {
		flex: 1 1 auto;
		min-width: 0;
		height: 14px;
	}

	.bar {
		display: block;
		height: 100%;
		min-width: 2px;
		border-radius: 3px;
		background-color: var(--text-color-accent);
	}

	.value {
		min-width: 2ch;
		text-align: right;
		font-variant-numeric: tabular-nums;
	}

	.empty {
		color: var(--text-color-accent);
	}

	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border: 0;
	}

	@each $g in s, a, b, c, d, f {
		.bar.tier-#{$g} {
			background-color: var(--tier-#{$g}-bg);
		}
	}
</style>
