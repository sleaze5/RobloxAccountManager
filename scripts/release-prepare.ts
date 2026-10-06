import { mkdtemp, readdir, readFile, rm, writeFile } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join, resolve } from "node:path"

const [wailsCLI, directoryArgument, targetsArgument, repository, signing] = process.argv.slice(2)
if (!wailsCLI || !directoryArgument || !targetsArgument || !repository || !signing) {
	throw new Error("Usage: release-prepare.ts <wails-cli> <release-directory> <targets> <repository> <production|rehearsal>")
}
if (signing !== "production" && signing !== "rehearsal") {
	throw new Error(`Unknown signing mode: ${signing}`)
}
if (!/^[\w.-]+\/[\w.-]+$/.test(repository)) {
	throw new Error(`Invalid repository: ${repository}`)
}

const root = resolve(import.meta.dir, "..")
const directory = resolve(directoryArgument)
const version = (await readFile(join(root, "internal", "appmeta", "VERSION"), "utf8")).trim()
const expected = targetsArgument.split(/\s+/).filter(Boolean)
if (expected.length === 0) {
	throw new Error("No release targets were given")
}
if (new Set(expected).size !== expected.length) {
	throw new Error(`Release targets repeat: ${expected.join(" ")}`)
}

const archives = (await readdir(directory, { withFileTypes: true }))
	.filter((entry) => entry.isFile() && /\.(zip|tar\.gz)$/.test(entry.name))
	.map((entry) => join(directory, entry.name))
	.sort()
if (archives.length === 0) {
	throw new Error(`No release archives were found in ${directory}`)
}

async function wails(...args: string[]): Promise<void> {
	const child = Bun.spawn(["go", "run", wailsCLI, "updater", ...args], { stdout: "inherit", stderr: "inherit" })
	const code = await child.exited
	if (code !== 0) {
		throw new Error(`wails3 updater ${args[0]} failed with exit code ${code}`)
	}
}

interface Artifact {
	url: string
	platform?: string
	arch?: string
}

// Wails records the platform and architecture it reads from each archive name.
function validateTargets(artifacts: Artifact[]): void {
	const found = new Map<string, string[]>()
	for (const artifact of artifacts) {
		const target = `${artifact.platform || "?"}-${artifact.arch || "?"}`
		found.set(target, [...(found.get(target) ?? []), artifact.url.split("/").pop() ?? artifact.url])
	}
	const problems: string[] = []
	for (const target of expected) {
		const files = found.get(target) ?? []
		if (files.length === 0) {
			problems.push(`${target}: no archive`)
		} else if (files.length > 1) {
			problems.push(`${target}: ${files.length} archives (${files.join(", ")})`)
		}
	}
	for (const [target, files] of found) {
		if (!expected.includes(target)) {
			problems.push(`unexpected archive for "${target}": ${files.join(", ")}`)
		}
	}
	if (problems.length > 0) {
		throw new Error(`Release archives do not match the targets ${expected.join(" ")}:\n${problems.join("\n")}`)
	}
}

const work = await mkdtemp(join(tmpdir(), "release-prepare-"))
try {
	const key = join(work, "updater.key")
	let publicKey: string
	if (signing === "production") {
		const secret = process.env.UPDATER_PRIVATE_KEY
		if (!secret) {
			throw new Error("Set UPDATER_PRIVATE_KEY to sign a production release")
		}
		await writeFile(key, secret, { mode: 0o600 })
		publicKey = join(root, "internal", "appmeta", "updater.key.pub")
	} else {
		await wails("genkey", "-out", key)
		publicKey = `${key}.pub`
	}
	const manifest = join(directory, "manifest.json")
	await wails(
		"manifest",
		"-version",
		version,
		"-key",
		key,
		"-url-prefix",
		`https://github.com/${repository}/releases/download/v${version}/`,
		"-output",
		manifest,
		...archives,
	)
	await rm(key, { force: true })
	validateTargets((JSON.parse(await readFile(manifest, "utf8")) as { artifacts: Artifact[] }).artifacts)
	await wails("verify", "-manifest", manifest, "-publickey", publicKey)
} finally {
	await rm(work, { recursive: true, force: true })
}
