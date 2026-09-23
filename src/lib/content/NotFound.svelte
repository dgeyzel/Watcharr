<!-- Shown when a title isn't on the (visible) list. -->
<script lang="ts">
	import { resolve } from "$app/paths";

	interface Props {
		what: string;
	}

	let { what }: Props = $props();
</script>

<div class="not-found">
	<h2>{what} not found</h2>
	<a href={resolve("/")}>Back to the list</a>
</div>

<style lang="scss">
	.not-found {
		display: flex;
		flex-flow: column;
		align-items: center;
		gap: 10px;
		margin: 40px 16px;

		a {
			text-decoration: underline;
		}
	}
</style>
