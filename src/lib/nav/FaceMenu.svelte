<script lang="ts">
	import { store } from "@/store.svelte";
	import Menu from "../Menu.svelte";
	import { parseTokenPayload, userHasPermission } from "../util/helpers";
	import { UserPermission } from "@/types";
	import { goto } from "$app/navigation";
	import { clearWatcharrData } from "../logout";
	import { notify } from "../util/notify";
	import AboutModal from "./AboutModal.svelte";
	import { resolve } from "$app/paths";

	let user = $derived(store.userInfo);
	let aboutModalOpen = $state(false);

	function logout() {
		clearWatcharrData();
		goto(resolve("/"));
	}

	function profile() {
		goto(resolve("/profile"));
	}

	function serverSettings() {
		goto(resolve("/server"));
	}

	function userManagement() {
		goto(resolve("/manage_users"));
	}

	function shareWatchedList() {
		const nid = notify({ type: "loading", text: "Getting link" });
		const ud = parseTokenPayload();
		console.log(ud);
		if (ud?.userId && ud?.username) {
			const shareLink = `${window.location.origin}/lists/${ud.userId}/${ud.username}`;
			navigator.clipboard
				.writeText(shareLink)
				.then(() => {
					notify({ id: nid, type: "success", text: "Copied share link" });
				})
				.catch((r) => {
					console.error("Failed to copy list share link", r);
					notify({
						id: nid,
						type: "error",
						text: `Failed to copy share link:<br/><a href="${shareLink}" target="_blank">${shareLink}</a>`,
						time: 20000,
					});
				});
		} else {
			notify({ id: nid, type: "error", text: "Failed to get link" });
		}
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
	{#if !store.userSettings?.private}
		<button class="plain" onclick={() => shareWatchedList()}>Share List</button>
	{/if}
	{#if user && userHasPermission(user.permissions, UserPermission.PERM_ADMIN)}
		<button class="plain" onclick={() => serverSettings()}>Settings</button>
		<button class="plain" onclick={() => userManagement()}>Users</button>
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
