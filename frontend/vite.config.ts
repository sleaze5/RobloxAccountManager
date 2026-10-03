import { readFileSync } from "node:fs"
import { svelte } from "@sveltejs/vite-plugin-svelte"
import wails from "@wailsio/runtime/plugins/vite"
import { defineConfig, type Plugin } from "vite"

const svelteRuntimeCompatibility: Plugin = {
	name: "svelte-runtime-rolldown-compatibility",
	enforce: "pre",
	transform(code, id) {
		if (
			!id
				.replaceAll("\\", "/")
				.includes("/svelte/src/internal/client/dom/operations.js")
		) {
			return null
		}
		return code.replace(
			"/** @type {string} */ (text.nodeValue) += /** @type {string} */ (next.nodeValue)",
			"text.nodeValue = /** @type {string} */ (text.nodeValue) + /** @type {string} */ (next.nodeValue)",
		)
	},
}

function installedVersion(name: string): string {
	const manifest = new URL(`./node_modules/${name}/package.json`, import.meta.url)
	return (JSON.parse(readFileSync(manifest, "utf8")) as { version: string }).version
}

export default defineConfig({
	define: {
		APP_VERSION: JSON.stringify(
			readFileSync(
				new URL("../internal/appmeta/VERSION", import.meta.url),
				"utf8",
			).trim(),
		),
		FRONTEND_DEPENDENCIES: JSON.stringify({
			runtime: [{ name: "svelte", version: installedVersion("svelte") }],
			build: [
				{ name: "bun", version: process.versions.bun ?? "unavailable" },
				{ name: "vite", version: installedVersion("vite") },
				{
					name: "typescript",
					version: installedVersion("@typescript/native"),
				},
			],
		}),
	},
	css: { transformer: "lightningcss" },
	build: { target: "esnext" },
	optimizeDeps: { exclude: ["svelte", "@lucide/svelte"] },
	plugins: [
		svelteRuntimeCompatibility,
		svelte({ prebundleSvelteLibraries: false }),
		wails("./bindings"),
	],
	ssr: { optimizeDeps: { exclude: ["svelte", "@lucide/svelte"] } },
	server: {
		host: "127.0.0.1",
		port: Number(process.env.WAILS_VITE_PORT) || 9245,
		strictPort: true,
	},
})
