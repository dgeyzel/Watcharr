<!-- The tier slot for a title (see tierSlot for the rule): a coloured
  letter chip, the "not rated" text, or nothing. -->
<script lang="ts">
	import type { Tier, WatchedStatus } from "@/types";
	import { tierSlot, type TierSlotSize } from "./tierSlot";

	interface Props {
		status: WatchedStatus | undefined;
		tier: Tier | null | undefined;
		size: TierSlotSize;
	}

	let { status, tier, size }: Props = $props();

	const slot = $derived(tierSlot(status, tier, size));
</script>

{#if slot.kind === "tier"}
	<span
		class="tier-badge {size} tier-{slot.letter.toLowerCase()}"
		role="img"
		aria-label="Tier: {slot.letter}"
		title="Tier: {slot.letter}">{slot.letter}</span
	>
{:else if slot.kind === "unrated"}
	<span class="tier-unrated {size}">{slot.text}</span>
{/if}

<style lang="scss">
	.tier-badge {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		font-weight: bold;
		line-height: 1;
		border: 1px solid var(--tier-border);
		border-radius: 6px;
		user-select: none;

		&.poster {
			min-width: 24px;
			height: 24px;
			padding: 0 5px;
			font-size: 15px;
		}

		&.page {
			min-width: 54px;
			height: 54px;
			padding: 0 10px;
			font-size: 34px;
			border-radius: 10px;
		}
	}

	@each $g in s, a, b, c, d, f {
		.tier-#{$g} {
			background-color: var(--tier-#{$g}-bg);
			color: var(--tier-#{$g}-fg);
		}
	}

	.tier-unrated {
		display: inline-flex;
		align-items: center;
		background-color: var(--tier-unrated-bg);
		color: var(--tier-unrated-fg);
		border: 1px solid var(--tier-border);
		border-radius: 6px;

		&.poster {
			padding: 2px 6px;
			font-size: 11px;
		}

		&.page {
			padding: 6px 10px;
			font-size: 14px;
		}
	}
</style>
