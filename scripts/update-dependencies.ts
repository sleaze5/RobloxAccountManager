import { readFile, writeFile } from "node:fs/promises"
import { join, resolve } from "node:path"
import { parseArgs } from "node:util"

import { run } from "./run"

type Version = { major: number; minor: number; patch: number; prerelease?: string }
type Manifest = {
	dependencies?: Record<string, string>
	devDependencies?: Record<string, string>
	devEngines?: { packageManager?: { name: string; version: string } }
}

const root = resolve(import.meta.dir, "..")
const frontend = join(root, "frontend")
const manifestPath = join(frontend, "package.json")
const goModPaths = [join(root, "go.mod"), join(root, "build", "tools", "go.mod")]
const wailsRuntime = "@wailsio/runtime"
const wailsModule = "github.com/wailsapp/wails/v3"

const { values } = parseArgs({
	options: {
		all: { type: "boolean", default: false },
		uncovered: { type: "boolean", default: false },
	},
})
if (values.all && values.uncovered) {
	throw new Error("Pass either --all or --uncovered, not both")
}

async function fetchVersions(name: string): Promise<string[]> {
	const url = `https://registry.npmjs.org/${name.replace("/", "%2f")}`
	const response = await fetch(url, {
		headers: { Accept: "application/vnd.npm.install-v1+json" },
	})
	if (!response.ok) {
		throw new Error(`Request to ${url} failed with ${response.status}`)
	}
	const { versions } = (await response.json()) as {
		versions: Record<string, { deprecated?: string }>
	}
	return Object.entries(versions)
		.filter(([, metadata]) => metadata.deprecated === undefined)
		.map(([version]) => version)
}

function parseVersion(text: string): Version | undefined {
	const match = /^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$/.exec(text)
	if (!match) {
		return undefined
	}
	return {
		major: Number(match[1]),
		minor: Number(match[2]),
		patch: Number(match[3]),
		prerelease: match[4],
	}
}

function compareRelease(a: Version, b: Version): number {
	return a.major - b.major || a.minor - b.minor || a.patch - b.patch
}

function prereleaseLine(
	prerelease: string | undefined,
): { tag: string; number: number } | undefined {
	const match = /^([A-Za-z-]+)\.(\d+)$/.exec(prerelease ?? "")
	return match ? { tag: match[1] ?? "", number: Number(match[2]) } : undefined
}

function rank(from: Version, version: Version): number[] | undefined {
	if (from.prerelease === undefined) {
		return version.prerelease === undefined && version.major === from.major
			? [version.major, version.minor, version.patch]
			: undefined
	}
	if (compareRelease(version, from) !== 0) {
		return undefined
	}
	if (version.prerelease === undefined) {
		return [Number.POSITIVE_INFINITY]
	}
	const line = prereleaseLine(from.prerelease)
	const candidate = prereleaseLine(version.prerelease)
	return line && candidate?.tag === line.tag ? [candidate.number] : undefined
}

function compareRanks(a: number[], b: number[]): number {
	for (const [index, value] of a.entries()) {
		const difference = value - (b[index] ?? 0)
		if (difference !== 0) {
			return difference
		}
	}
	return 0
}

function newestCompatible(current: string, candidates: string[]): string | undefined {
	const from = parseVersion(current)
	let best = from && rank(from, from)
	let newest: string | undefined
	for (const text of candidates) {
		const version = parseVersion(text)
		const candidate = from && version && rank(from, version)
		if (candidate && best && compareRanks(candidate, best) > 0) {
			best = candidate
			newest = text
		}
	}
	return newest
}

function goVersions(text: string): Map<string, string> {
	const versions = new Map<string, string>()
	for (const [, name, version] of text.matchAll(
		/^[ \t]*(?:require[ \t]+)?(toolchain|[\w.-]+\.[\w./-]+)[ \t]+(\S+)/gm,
	)) {
		if (name && version) {
			versions.set(name, version)
		}
	}
	return versions
}

const changes: string[] = []
const manifest = (await Bun.file(manifestPath).json()) as Manifest

const packageUpdates = await Promise.all(
	[manifest.dependencies, manifest.devDependencies].flatMap((section) =>
		Object.entries(section ?? {}).map(async ([key, spec]) => {
			const alias = /^npm:(@?[^@]+)@(.+)$/.exec(spec)
			if (!section || key === wailsRuntime || (values.uncovered && !alias)) {
				return undefined
			}
			const name = alias?.[1] ?? key
			const current = alias?.[2] ?? spec
			const next = newestCompatible(current, await fetchVersions(name))
			return next
				? {
						section,
						key,
						current,
						next,
						spec: alias ? `npm:${name}@${next}` : next,
					}
				: undefined
		}),
	),
)
for (const update of packageUpdates) {
	if (update) {
		update.section[update.key] = update.spec
		changes.push(`- \`${update.key}\` ${update.current} → ${update.next}`)
	}
}

const packageManager = manifest.devEngines?.packageManager
if (packageManager) {
	const next = newestCompatible(
		packageManager.version,
		await fetchVersions(packageManager.name),
	)
	if (next) {
		changes.push(`- \`${packageManager.name}\` ${packageManager.version} → ${next}`)
		packageManager.version = next
	}
}

await writeFile(manifestPath, `${JSON.stringify(manifest, null, "\t")}\n`)
await run(frontend, "bun", "install")

const goBefore = await Promise.all(
	goModPaths.map(async (path) => goVersions(await readFile(path, "utf8"))),
)
await run(root, "go", "get", "toolchain@latest")
// Dependabot ignores Wails, because @wailsio/runtime must change with it. The
// fix task that follows this script updates @wailsio/runtime to match.
await run(root, "go", "get", `${wailsModule}@latest`)
if (!values.uncovered) {
	const modules = (
		await run(
			root,
			"go",
			"list",
			"-m",
			"-f",
			"{{if not (or .Main .Indirect)}}{{.Path}}{{end}}",
			"all",
		)
	)
		.split("\n")
		.filter((path) => path !== "")
	if (modules.length > 0) {
		await run(root, "go", "get", "-u", ...modules)
	}
	await run(root, "go", "-C", "build/tools", "get", "tool")
}
// Binding generation runs before the fix task tidies, and it fails when a
// new Wails version needs a go.sum entry that go get did not add.
await run(root, "go", "mod", "tidy")
const goAfter = await Promise.all(
	goModPaths.map(async (path) => goVersions(await readFile(path, "utf8"))),
)
for (const [index, after] of goAfter.entries()) {
	for (const [name, version] of after) {
		const previous = goBefore[index]?.get(name)
		if (previous !== undefined && previous !== version) {
			changes.push(`- \`${name}\` ${previous} → ${version}`)
		}
	}
}

const chromeForTestingChange = (
	await run(root, "bun", "run", "./scripts/update-cft-manifest.ts")
).trim()
if (chromeForTestingChange !== "") {
	changes.push(chromeForTestingChange)
}

const summary =
	changes.length === 0 ? "Every dependency is up to date." : changes.join("\n")
console.log(summary)
const summaryFile = process.env["DEPS_SUMMARY_FILE"]
if (summaryFile) {
	await writeFile(summaryFile, `${summary}\n`)
}
