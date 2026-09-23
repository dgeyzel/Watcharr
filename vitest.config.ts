import { sveltekit } from "@sveltejs/kit/vite";
import { svelteTesting } from "@testing-library/svelte/vite";
import { defineConfig } from "vitest/config";
import { readFileSync } from "fs";
const pkg = JSON.parse(readFileSync("package.json", "utf8"));

// Kept separate from vite.config.ts so the PWA plugin isn't loaded for tests.
export default defineConfig({
	plugins: [sveltekit(), svelteTesting()],
	define: {
		__WATCHARR_VERSION__: JSON.stringify(pkg.version),
	},
	test: {
		environment: "jsdom",
		include: ["src/**/*.test.ts"],
		setupFiles: ["./src/vitest-setup.ts"],
	},
});
