// NOTE: this file intentionally does not import `defineConfig` from "vitest/config".
// vite 8.2.1's rolldown config bundler fails to externalize transitive `node:url`
// imports from vitest/config ("Cannot find package 'node'" startup error), so the
// config is a plain object with zero bundlable imports.
export default {
	test: {
		environment: "jsdom",
		include: ["src/**/*.test.{ts,tsx}"],
		coverage: {
			provider: "v8",
			reporter: ["text", "json-summary", "lcov"],
			include: ["src/util/**", "src/ui/keybindings.ts", "src/api/util/**", "src/ui/util/**"],
			exclude: ["src/util/emoji/data.json", "src/util/emoji/generate.go", "src/util/**/*.css", "coverage/**"],
			thresholds: { lines: 0, statements: 0, functions: 0, branches: 0 },
		},
	},
	resolve: {
		alias: {
			"@": "/src",
		},
	},
}
