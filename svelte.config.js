import adapter from "@sveltejs/adapter-node";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";
import { sveltePreprocess } from "svelte-preprocess";

/** @type {import('@sveltejs/kit').Config} */
const config = {
	// Consult https://kit.svelte.dev/docs/integrations#preprocessors
	// for more information about preprocessors
	preprocess: [
		sveltePreprocess({
			scss: {
				// Only prepend partials that we want access to everywhere
				// by default. They can't have css otherwise our css output
				// will be bloated. Global styles can be in our `norm.scss` file
				// which is imported (`@use`d) in our root `+layout.svelte` file.
				prependData:
					`@use "./src/styles/_vars.scss" as *;` +
					`@use "./src/styles/_mixins.scss" as *;`,
			},
		}),
		vitePreprocess(),
	],

	kit: {
		adapter: adapter(),

		alias: {
			"@": "src",
		},

		typescript: {
			// Include the test tooling so it is type checked and linted too.
			config: (tsconfig) => {
				tsconfig.include.push(
					"../e2e/**/*.ts",
					"../playwright.config.ts",
					"../vitest.config.ts",
				);
				return tsconfig;
			},
		},
	},
};

export default config;
