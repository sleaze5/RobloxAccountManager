import { lstat, readdir, realpath, rm } from "node:fs/promises"
import { isAbsolute, relative, resolve, sep } from "node:path"

const repositoryRoot = process.argv[2]
if (!repositoryRoot) {
	throw new Error("Repository root argument is required")
}

const root = resolve(repositoryRoot)

function assertInsideRepository(target: string): void {
	const relativePath = relative(root, target)
	if (
		relativePath === "" ||
		relativePath === ".." ||
		relativePath.startsWith(`..${sep}`) ||
		isAbsolute(relativePath)
	) {
		throw new Error(`Refusing to operate outside repository: ${target}`)
	}
}

async function removeTarget(relativePath: string): Promise<void> {
	const target = resolve(root, relativePath)
	assertInsideRepository(target)

	let stats
	try {
		stats = await lstat(target)
	} catch (error) {
		if ((error as NodeJS.ErrnoException).code === "ENOENT") {
			return
		}
		throw error
	}

	const resolvedTarget = await realpath(target)
	assertInsideRepository(resolvedTarget)

	if (stats.isSymbolicLink()) {
		throw new Error(`Refusing to recursively delete a linked path: ${target}`)
	}

	await rm(target, { recursive: true, force: true })
}

const targets = [
	".task",
	"dist",
	"frontend/node_modules",
	"frontend/bindings",
	"frontend/dist",
	"frontend/frontend",
	"build/appicon.png",
	"build/windows/icon.ico",
	"RobloxAccountManager.exe",
	"storage",
	"logs",
]

await targets.reduce(
	(previous, target) => previous.then(() => removeTarget(target)),
	Promise.resolve(),
)

const frontend = resolve(root, "frontend")
try {
	const entries = await readdir(frontend, { withFileTypes: true })
	await Promise.all(
		entries
			.filter(
				(entry) =>
					entry.isDirectory() && entry.name.startsWith(".bindings-tmp-"),
			)
			.map((entry) => removeTarget(`frontend/${entry.name}`)),
	)
} catch (error) {
	if ((error as NodeJS.ErrnoException).code !== "ENOENT") {
		throw error
	}
}

const rootEntries = await readdir(root, { withFileTypes: true })
await Promise.all(
	rootEntries
		.filter(
			(entry) =>
				entry.isFile() &&
				entry.name.startsWith("wails_windows_") &&
				/\.(syso|json)$/.test(entry.name),
		)
		.map((entry) => removeTarget(entry.name)),
)
