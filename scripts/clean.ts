import { lstat, readdir, realpath, rm } from "node:fs/promises"
import { isAbsolute, relative, resolve, sep } from "node:path"

const repositoryRoot = process.argv[2]
if (!repositoryRoot) {
	throw new Error("Repository root argument is required")
}

const root = resolve(repositoryRoot)

function assertInsideRepository(target: string): void {
	const relativePath = relative(root, target)
	if (relativePath === "" || relativePath === ".." || relativePath.startsWith(`..${sep}`) || isAbsolute(relativePath)) {
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

const targets = [".task", "dist", "frontend/node_modules", "frontend/bindings", "frontend/dist", "frontend/frontend", "build/appicon.png", "build/windows/icon.ico", "RobloxAccountManager.exe", "storage", "logs"]

for (const target of targets) {
	await removeTarget(target)
}

const frontend = resolve(root, "frontend")
try {
	for (const entry of await readdir(frontend, { withFileTypes: true })) {
		if (entry.isDirectory() && entry.name.startsWith(".bindings-tmp-")) {
			await removeTarget(`frontend/${entry.name}`)
		}
	}
} catch (error) {
	if ((error as NodeJS.ErrnoException).code !== "ENOENT") {
		throw error
	}
}

for (const entry of await readdir(root, { withFileTypes: true })) {
	if (entry.isFile() && entry.name.startsWith("wails_windows_") && entry.name.endsWith(".syso")) {
		await removeTarget(entry.name)
	}
}
