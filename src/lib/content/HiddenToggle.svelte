<!-- Admin only: hide a title from visitors (keep it as a draft). -->
<script lang="ts">
	import type { Watched } from "@/types";
	import Checkbox from "../Checkbox.svelte";
	import { req } from "../util/api";
	import { notify } from "../util/notify";

	interface Props {
		watched: Watched;
	}

	let { watched = $bindable() }: Props = $props();

	let disabled = $state(false);

	async function toggled(on: boolean) {
		disabled = true;
		const nid = notify({ type: "loading", text: "Saving" });
		try {
			await req.put(`/watched/${watched.id}`, { hidden: on });
			watched.hidden = on;
			notify({
				id: nid,
				type: "success",
				text: on ? "Hidden from visitors" : "Visible to visitors",
			});
		} catch (err) {
			console.error("HiddenToggle: failed to update", err);
			notify({ id: nid, type: "error", text: "Failed to save" });
		}
		disabled = false;
	}
</script>

<label class="hidden-toggle">
	<Checkbox
		name="hidden"
		value={watched.hidden ?? false}
		{disabled}
		{toggled}
	/>
	Hidden from visitors
</label>

<style lang="scss">
	.hidden-toggle {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		max-width: 380px;
		margin: 12px auto 0 auto;
		color: $text-color;
		font-size: 14px;
		cursor: pointer;
	}
</style>
