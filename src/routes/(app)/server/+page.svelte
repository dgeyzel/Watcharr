<script lang="ts">
	import Checkbox from "@/lib/Checkbox.svelte";
	import Spinner from "@/lib/Spinner.svelte";
	import { notify } from "@/lib/util/notify";
	import type { Content, ServerConfig } from "@/types";
	import SettingsList from "@/lib/settings/SettingsList.svelte";
	import Setting from "@/lib/settings/Setting.svelte";
	import SettingButton from "@/lib/settings/SettingButton.svelte";
	import { getServerFeatures, req } from "@/lib/util/api";
	import Stats from "@/lib/stats/Stats.svelte";
	import Error from "@/lib/Error.svelte";
	import Stat from "@/lib/stats/Stat.svelte";
	import TwitchModal from "./modals/TwitchModal.svelte";
	import RegionDropDown from "@/lib/RegionDropDown.svelte";
	import TaskScheduleModal from "./modals/TaskScheduleModal.svelte";
	import { resolve } from "$app/paths";

	let serverConfig: ServerConfig | undefined = $state();
	let twitchModalOpen = $state(false);
	let taskScheduleModalOpen = $state(false);
	// Disabled vars for disabling inputs until api request completes
	let debugDisabled = $state(false);
	let tmdbkDisabled = $state(false);
	let countryDisabled = $state(false);

	async function getServerConfig() {
		serverConfig = await req.get<ServerConfig>(`/server/config`);
	}

	export function updateServerConfig<K extends keyof ServerConfig>(
		name: K,
		value: ServerConfig[K],
		done?: (respData?: object) => void,
	) {
		if (!serverConfig) {
			console.error("updateServerConfig: No server config to update!");
			notify({ type: "error", text: "No server config to update!" });
			return;
		}
		console.log("Updating server setting", name, "to", value);
		const originalValue = serverConfig[name];
		const nid = notify({ type: "loading", text: "Updating" });
		req
			.postWhole<object>("/server/config", { key: name, value: value })
			.then((r) => {
				if (r.status === 200) {
					serverConfig![name] = value;
					notify({ id: nid, type: "success", text: "Updated" });
					if (typeof done !== "undefined") done(r?.body);
				}
			})
			.catch((err) => {
				console.error("Failed to update user setting", err);
				notify({ id: nid, type: "error", text: "Couldn't Update" });
				serverConfig![name] = originalValue;
				if (typeof done !== "undefined") done();
			});
	}

	interface ServerStats {
		users: number;
		privateUsers: number;
		watchedMovies: number;
		watchedShows: number;
		watchedSeasons: number;
		mostWatchedMovie: Content;
		mostWatchedShow: Content;
		activities: number;
	}

	async function getServerStats() {
		return await req.get<ServerStats>("/server/stats");
	}
</script>

<div class="content">
	<div class="inner">
		<SettingsList>
			<h2>Server Settings</h2>

			<Stats>
				{#await getServerStats()}
					<Spinner />
				{:then stats}
					<Stat
						name="Users"
						value={stats.users}
						href={resolve("/manage_users")}
						large
					/>
					<Stat name="Private Users" value={stats.privateUsers} large />
					<Stat name="Watched Movies" value={stats.watchedMovies} large />
					<Stat name="Watched Shows" value={stats.watchedShows} large />
					<Stat name="Watched Seasons" value={stats.watchedSeasons} large />
					<Stat name="Activities" value={stats.activities} large />
					{#if stats.mostWatchedMovie?.title}
						<Stat
							name="Most Watched Movie"
							value={stats.mostWatchedMovie.title}
							href={resolve("/movie/{stats.mostWatchedMovie.tmdbId}")}
						/>
					{/if}
					{#if stats.mostWatchedShow?.title}
						<Stat
							name="Most Watched Show"
							value={stats.mostWatchedShow.title}
							href={resolve("/tv/{stats.mostWatchedShow.tmdbId}")}
						/>
					{/if}
				{:catch err}
					<Error error={err} pretty="Failed to get server stats!" />
				{/await}
			</Stats>

			{#await getServerConfig()}
				<Spinner />
			{:then}
				<h3>General</h3>
				{#if serverConfig}
					<Setting
						title="Default Country"
						desc="Default country for new users. This can be changed per user and won't affect existing users."
					>
						<RegionDropDown
							selectedCountry={serverConfig.DEFAULT_COUNTRY}
							disabled={countryDisabled}
							onChange={(c) => {
								countryDisabled = true;
								updateServerConfig("DEFAULT_COUNTRY", c, () => {
									countryDisabled = false;
								});
							}}
						/>
					</Setting>
					<Setting title="TMDB Key" desc="Provide your own TMDB API Key">
						<input
							type="password"
							placeholder="TMDB Key"
							bind:value={serverConfig.TMDB_KEY}
							onblur={() => {
								tmdbkDisabled = true;
								updateServerConfig("TMDB_KEY", serverConfig!.TMDB_KEY, () => {
									tmdbkDisabled = false;
								});
							}}
							disabled={tmdbkDisabled}
						/>
					</Setting>
					<Setting title="Debug" desc="Enable debug logging." row>
						<Checkbox
							name="DEBUG"
							disabled={debugDisabled}
							value={serverConfig.DEBUG}
							toggled={(on) => {
								debugDisabled = true;
								updateServerConfig("DEBUG", on, () => {
									debugDisabled = false;
								});
							}}
						/>
					</Setting>
					<Setting>
						<SettingButton
							title="Task Schedule"
							desc="View and configure server task schedule."
							icon="arrow"
							onClick={() => {
								taskScheduleModalOpen = true;
							}}
						/>
					</Setting>
					{#if taskScheduleModalOpen}
						<TaskScheduleModal onClose={() => (taskScheduleModalOpen = false)}
						></TaskScheduleModal>
					{/if}
					<div>
						<h3>Services</h3>
						<h5 class="norm">
							These integrations are not in their final stages. Consider them a
							preview/beta, if you have any issues,
							<a
								style="text-decoration: underline;"
								href="https://github.com/sbondCo/Watcharr/issues/new/choose"
								target="_blank"
							>
								please report them.
							</a>
						</h5>
					</div>

					<Setting title="Twitch">
						<SettingButton
							title="Twitch"
							desc="Twitch application credentials for enabling game support (via IGDB)."
							icon={serverConfig.TWITCH &&
							Object.keys(serverConfig.TWITCH).length > 0
								? "arrow"
								: "add"}
							onClick={() => {
								twitchModalOpen = true;
							}}
						/>
					</Setting>

					{#if twitchModalOpen}
						<TwitchModal
							cfg={serverConfig.TWITCH}
							onClose={() => {
								// "temporary" solution to showing added servers
								// and reloading data to revert modified but not saved changes.
								getServerConfig();
								getServerFeatures();
								twitchModalOpen = false;
							}}
						/>
					{/if}
				{/if}
			{:catch err}
				<Error error={err} pretty="Failed to load server config!" />
			{/await}
		</SettingsList>
	</div>
</div>

<style lang="scss">
	.content {
		display: flex;
		width: 100%;
		justify-content: center;
		padding: 0 30px 30px 30px;

		.inner {
			min-width: 400px;
			max-width: 400px;
			overflow: hidden;

			h2 {
				overflow: hidden;
				white-space: nowrap;
				text-overflow: ellipsis;
			}

			& > div:not(:first-of-type) {
				margin-top: 30px;
			}

			@media screen and (max-width: 440px) {
				width: 100%;
				min-width: unset;
			}
		}
	}
</style>
