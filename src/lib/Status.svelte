<script lang="ts">
	import Icon from "./Icon.svelte";
	import type { WatchedStatus } from "../types";
	import tooltip from "./actions/tooltip";
	import { toUnderstandableStatus } from "./util/helpers";

	interface Props {
		status: WatchedStatus | undefined;
		onChange: (newStatus: WatchedStatus) => void;
	}

	let { status, onChange }: Props = $props();

	function handleStatusClick(s: WatchedStatus) {
		if (s === status) return;
		onChange(s);
	}
</script>

<div class="status">
	<button
		class={status && status !== "PLANNED" ? "not-active" : ""}
		aria-label={toUnderstandableStatus("PLANNED")}
		aria-pressed={status === "PLANNED"}
		onclick={() => handleStatusClick("PLANNED")}
		use:tooltip={{
			text: toUnderstandableStatus("PLANNED"),
			pos: "top",
		}}
	>
		<Icon i="calendar" />
	</button>
	<button
		class={status && status !== "WATCHING" ? "not-active" : ""}
		aria-label={toUnderstandableStatus("WATCHING")}
		aria-pressed={status === "WATCHING"}
		onclick={() => handleStatusClick("WATCHING")}
		use:tooltip={{
			text: toUnderstandableStatus("WATCHING"),
			pos: "top",
		}}
	>
		<Icon i="clock" />
	</button>
	<button
		class={status && status !== "FINISHED" ? "not-active" : ""}
		aria-label={toUnderstandableStatus("FINISHED")}
		aria-pressed={status === "FINISHED"}
		onclick={() => handleStatusClick("FINISHED")}
		use:tooltip={{
			text: toUnderstandableStatus("FINISHED"),
			pos: "top",
		}}
	>
		<Icon i="check" />
	</button>
	<button
		class={status && status !== "HOLD" ? "not-active" : ""}
		aria-label={toUnderstandableStatus("HOLD")}
		aria-pressed={status === "HOLD"}
		onclick={() => handleStatusClick("HOLD")}
		use:tooltip={{
			text: toUnderstandableStatus("HOLD"),
			pos: "top",
		}}
	>
		<Icon i="pause" />
	</button>
	<button
		class={status && status !== "DROPPED" ? "not-active" : ""}
		aria-label={toUnderstandableStatus("DROPPED")}
		aria-pressed={status === "DROPPED"}
		onclick={() => handleStatusClick("DROPPED")}
		use:tooltip={{
			text: toUnderstandableStatus("DROPPED"),
			pos: "top",
		}}
	>
		<Icon i="thumb-down" />
	</button>
</div>

<style lang="scss">
	.status {
		display: flex;
		flex-flow: row;
		gap: 10px;
		width: 100%;
		height: 100%;
		container-type: inline-size;

		button {
			font-size: 10px;
			padding: 5px 10px;
			height: 58px;
		}

		/* 380px is how big the this container is usually,
		but when it starts to shrink we want to button height
		to be unset so that buttons become responsive. */
		@container (width < 380px) {
			button {
				height: unset;
			}
		}
	}
</style>
