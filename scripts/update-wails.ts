import { readFile, writeFile } from "node:fs/promises"
import { join, resolve } from "node:path"

const goModule = "github.com/wailsapp/wails/v3"
const npmPackage = "@wailsio/runtime"
const root = resolve(import.meta.dir, "..")
const cliPins = [
	{ path: join(root, "Taskfile.yml"), count: 1 },
	{ path: join(root, "build", "config.yml"), count: 4 },
]

async function fetchJSON<T>(url: string): Promise<T> {
	const response = await fetch(url)
	if (!response.ok) {
		throw new Error(`Request to ${url} failed with ${response.status}`)
	}
	return (await response.json()) as T
}

async function run(cwd: string, ...command: string[]): Promise<void> {
	const child = Bun.spawn(command, { cwd, stdout: "inherit", stderr: "inherit" })
	const code = await child.exited
	if (code !== 0) {
		throw new Error(`${command.join(" ")} failed with exit code ${code}`)
	}
}

const goMod = await readFile(join(root, "go.mod"), "utf8")
const current = /^\s*github\.com\/wailsapp\/wails\/v3 (v\S+)$/m.exec(goMod)?.[1]
if (!current) {
	throw new Error(`go.mod does not require ${goModule}`)
}

const latest = (await fetchJSON<{ Version?: string }>(`https://proxy.golang.org/${goModule}/@latest`)).Version
if (!latest || !/^v3\.\d+\.\d+(-beta\.\d+)?$/.test(latest)) {
	throw new Error(`The Go proxy returned an unexpected ${goModule} version: ${latest}`)
}
const runtime = (await fetchJSON<{ "dist-tags"?: { latest?: string } }>(`https://registry.npmjs.org/${npmPackage}`))["dist-tags"]?.latest
if (runtime !== latest.slice(1)) {
	throw new Error(`${npmPackage} ${runtime} does not match ${goModule} ${latest}`)
}
if (latest === current) {
	console.log(`Wails is already pinned to ${current}`)
	process.exit(0)
}

const updates = await Promise.all(
	cliPins.map(async ({ path, count }) => {
		const text = await readFile(path, "utf8")
		const pins = text.split(`wails3@${current}`).length - 1
		if (pins !== count) {
			throw new Error(`Expected ${count} Wails CLI pins in ${path}, found ${pins}`)
		}
		return { path, text: text.replaceAll(`wails3@${current}`, `wails3@${latest}`) }
	}),
)
await Promise.all(updates.map(({ path, text }) => writeFile(path, text)))
await run(root, "go", "get", `${goModule}@${latest}`)
await run(join(root, "frontend"), "bun", "add", "--exact", `${npmPackage}@${runtime}`)
console.log(`Pinned Wails ${current} -> ${latest}`)
