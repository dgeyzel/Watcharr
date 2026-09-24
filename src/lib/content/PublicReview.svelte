<!-- Read only review shown to visitors on a title's page: status and review
  text. Never shows a numeric rating. -->
<script lang="ts">
	import type { Watched } from "@/types";
	import Icon from "../Icon.svelte";
	import { toUnderstandableStatus, watchedStatuses } from "../util/helpers";

	interface Props {
		watched: Watched;
	}

	let { watched }: Props = $props();
</script>

<section class="review" aria-label="Review">
	<div class="status">
		<Icon i={watchedStatuses[watched.status]} wh={18} />
		<span>{toUnderstandableStatus(watched.status)}</span>
	</div>
	{#if watched.thoughts}
		<p class="thoughts">{watched.thoughts}</p>
	{/if}
</section>

<style lang="scss">
	.review {
		display: flex;
		flex-flow: column;
		gap: 10px;
		width: 100%;
		max-width: 600px;
		color: $text-color;
		margin: 22px auto 0 auto;
		padding: 0 16px;

		.status {
			display: flex;
			align-items: center;
			gap: 6px;
			fill: $text-color;
			font-weight: bold;
			text-transform: capitalize;
		}

		.thoughts {
			white-space: pre-wrap;
			overflow-wrap: anywhere;
			line-height: 1.5;
		}
	}
</style>
