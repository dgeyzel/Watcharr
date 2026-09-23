<script lang="ts">
	import { goto } from "$app/navigation";
	import { page } from "$app/state";
	import { type AuthResponse, type AvailableAuthProviders } from "@/types";
	import { noAuthReq } from "@/lib/util/api";
	import { onMount } from "svelte";
	import { notify, unNotify } from "@/lib/util/notify";
	import { ReqerError } from "@/lib/util/fetch";
	import { resolve } from "$app/paths";

	let error: string | undefined = $state();

	onMount(() => {
		if (localStorage.getItem("token")) {
			goto(resolve("/"));
		}

		if (!error && page.url.searchParams.get("again")) {
			error = "Please Login Again";
		}

		noAuthReq
			.get<AvailableAuthProviders>("/auth/available")
			.then((r) => {
				if (r?.isInSetup) {
					console.log("Server is in setup.. navigating to web setup page.");
					goto(resolve("/setup"));
				}
			})
			.catch((err) => {
				console.error("Failed to check if server is in setup", err);
			});
	});

	function handleLogin(ev: SubmitEvent) {
		ev.preventDefault();
		const fd = new FormData(ev.target! as HTMLFormElement);
		const user = fd.get("username");
		const pass = fd.get("password");

		if (!user || !pass) {
			error = "Username and Password fields are required";
			return;
		}

		const nid = notify({ text: "Logging in", type: "loading" });
		noAuthReq
			.post<AuthResponse>("/auth/", {
				username: user,
				password: pass,
			})
			.then((resp) => {
				if (resp?.token) {
					console.log("Received token... logging in.");
					localStorage.setItem("token", resp.token);
					goto(resolve("/"));
					notify({ id: nid, text: `Welcome ${user}!`, type: "success" });
				}
			})
			.catch((err) => {
				error = ReqerError.getMsg(err, "Login failed");
				unNotify(nid);
			});
	}
</script>

<svelte:head>
	<title>Admin Login</title>
</svelte:head>

<div>
	<div class="inner">
		<h2>Admin Login</h2>

		{#if error}
			<span class="error">{error}!</span>
		{/if}

		<form onsubmit={handleLogin}>
			<label for="username">Username</label>
			<input type="text" id="username" name="username" placeholder="Username" />

			<label for="password">Password</label>
			<input
				type="password"
				id="password"
				name="password"
				placeholder="Password"
			/>

			<div class="login-btns">
				<button type="submit"><span class="watcharr">W</span>Login</button>
			</div>
		</form>
	</div>
</div>

<style lang="scss">
	div,
	form {
		display: flex;
		flex-flow: column;
		align-items: center;
		gap: 10px;
		margin: 0 35px;
	}

	.inner,
	form {
		width: 100%;
		max-width: 400px;
	}

	.inner h2 {
		font-weight: normal;
	}

	label {
		align-self: flex-start;
		font-weight: bold;
	}

	.login-btns {
		display: flex;
		flex-flow: row;
		gap: 10px;
		width: 100%;

		button {
			display: flex;
			flex-flow: row;
			gap: 10px;

			.watcharr {
				font-family: "Rampart One";
				font-size: 19px;
				line-height: 19px;
			}
		}
	}

	.error {
		display: flex;
		justify-content: center;
		width: 100%;
		padding: 10px;
		background-color: rgb(221, 48, 48);
		text-transform: capitalize;
		color: white;
	}
</style>
