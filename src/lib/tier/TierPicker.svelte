<!-- Admin only: pick an S-F tier (or clear it). Plain buttons, so it is
  keyboard operable (Tab to a tier, Enter/Space to pick it); arrow keys move
  between tiers. -->
<script lang="ts">
	import { TIERS, type Tier } from "@/types";

	interface Props {
		tier: Tier | null | undefined;
		onChange: (tier: Tier | null) => void;
		disabled?: boolean;
	}

	let { tier, onChange, disabled = false }: Props = $props();

	let groupEl: HTMLDivElement | undefined = $state();

	function pick(g: Tier | null) {
		if (disabled || g === (tier ?? null)) {
			return;
		}
		onChange(g);
	}

	function onKeydown(e: KeyboardEvent) {
		if (e.key !== "ArrowRight" && e.key !== "ArrowLeft") {
			return;
		}
		const btns = Array.from(
			groupEl?.querySelectorAll<HTMLButtonElement>("button") ?? [],
		);
		const i = btns.indexOf(document.activeElement as HTMLButtonElement);
		if (i === -1) {
			return;
		}
		e.preventDefault();
		const next = e.key === "ArrowRight" ? i + 1 : i - 1;
		btns[(next + btns.length) % btns.length]?.focus();
	}
</script>

<div class="tier-picker">
	<span class="label" id="tier-picker-label">Tier</span>
	<div
		class="tiers"
		role="toolbar"
		aria-labelledby="tier-picker-label"
		bind:this={groupEl}
		onkeydown={onKeydown}
	>
		{#each TIERS as g (g)}
			<button
				type="button"
				class="tier tier-{g.toLowerCase()}"
				class:active={tier === g}
				aria-pressed={tier === g}
				aria-label="Tier {g}"
				{disabled}
				onclick={() => pick(g)}
			>
				{g}
			</button>
		{/each}
		<button
			type="button"
			class="clear"
			aria-label="Clear tier"
			disabled={disabled || !tier}
			onclick={() => pick(null)}
		>
			Clear
		</button>
	</div>
</div>

<style lang="scss">
	.tier-picker {
		display: flex;
		flex-flow: column;
		gap: 6px;
		width: 100%;

		.label {
			font-weight: bold;
		}
	}

	.tiers {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}

	button {
		min-width: 40px;
		height: 40px;
		padding: 0 10px;
		font-weight: bold;
		font-size: 17px;
		border-radius: 8px;
		border: 2px solid var(--tier-border);
		opacity: 0.55;
		transition: opacity 100ms ease;

		&:hover:not(:disabled),
		&.active {
			opacity: 1;
		}

		&.active {
			outline: 3px solid var(--text-color);
			outline-offset: 2px;
		}

		&:focus-visible {
			opacity: 1;
			outline: 3px dashed var(--text-color);
			outline-offset: 2px;
		}

		&.clear {
			font-size: 13px;
			opacity: 1;
		}
	}

	@each $g in s, a, b, c, d, f {
		.tier-#{$g} {
			background-color: var(--tier-#{$g}-bg);
			color: var(--tier-#{$g}-fg);
		}
	}
</style>
