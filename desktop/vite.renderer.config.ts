import { defineConfig } from "vite"

// https://vitejs.dev/config
export default defineConfig({
	build: {
		rollupOptions: {
			input: {
				tabs: "src/chrome/tabs.html",
				exited: "src/chrome/exited.html",
			},
		},
	},
})
