<script lang="ts">
	import { store } from "@/store.svelte";
	import Menu from "../Menu.svelte";
	import { userHasPermission } from "../util/helpers";
	import { UserPermission } from "@/types";
	import { goto } from "$app/navigation";
	import { clearWatcharrData } from "../logout";
	import AboutModal from "./AboutModal.svelte";
	import { resolve } from "$app/paths";

	let user = $derived(store.userInfo);
	let aboutModalOpen = $state(false);

	function logout() {
		clearWatcharrData();
		// Full reload so the app starts over in visitor mode.
		location.href = resolve("/");
	}

	function profile() {
		goto(resolve("/profile"));
	}

	function serverSettings() {
		goto(resolve("/server"));
	}

	function closeAbout() {
		aboutModalOpen = false;
	}
</script>

<Menu conf={{ width: "140px", arrowRight: "10px" }}>
	{#if user?.username}
		<h5 title={user.username}>Hi {user.username}!</h5>
	{/if}
	<button class="plain" onclick={() => profile()}>Profile</button>
	{#if user && userHasPermission(user.permissions, UserPermission.PERM_ADMIN)}
		<button class="plain" onclick={() => serverSettings()}>Settings</button>
	{/if}
	<button class="plain" onclick={() => logout()}>Logout</button>
	<span>
		<button
			class="menu-footer"
			onclick={() => {
				aboutModalOpen = !aboutModalOpen;
			}}
		>
			about
		</button>
		|
		<a
			class="menu-footer"
			href="https://github.com/sbondCo/Watcharr/releases"
			target="_blank"
		>
			v{__WATCHARR_VERSION__}
		</a>
	</span>
</Menu>

{#if aboutModalOpen}
	<AboutModal onClose={closeAbout} />
{/if}
