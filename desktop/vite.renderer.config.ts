import { defineConfig } from "vite"

// https://vitejs.dev/config
export default defineConfig({
	build: {
		rollupOptions: {
			input: {
				exited: "src/chrome/exited.html",
			},
		},
	},
})
