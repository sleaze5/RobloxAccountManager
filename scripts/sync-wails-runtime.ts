import { writeFile } from "node:fs/promises"
import { join, resolve } from "node:path"
import { parseArgs } from "node:util"

type Manifest = { dependencies?: Record<string, string> }

const root = resolve(import.meta.dir, "..")
const frontend = join(root, "frontend")
const manifestPath = join(frontend, "package.json")
const goModule = "github.com/wailsapp/wails/v3"
const npmPackage = "@wailsio/runtime"

const { values } = parseArgs({
	options: { check: { type: "boolean", default: false } },
})

async function run(cwd: string, ...command: string[]): Promise<string> {
	const child = Bun.spawn(command, { cwd, stdout: "pipe", stderr: "inherit" })
	const [output, code] = await Promise.all([
		new Response(child.stdout).text(),
		child.exited,
	])
	if (code !== 0) {
		throw new Error(`${command.join(" ")} failed with exit code ${code}`)
	}
	return output
}

const goVersion = (
	await run(root, "go", "list", "-m", "-f", "{{.Version}}", goModule)
).trim()
const expected = goVersion.replace(/^v/, "")
const manifest = (await Bun.file(manifestPath).json()) as Manifest
const dependencies = manifest.dependencies
const current = dependencies?.[npmPackage]
if (!dependencies || current === undefined) {
	throw new Error(`${manifestPath} does not depend on ${npmPackage}`)
}
if (current === expected) {
	process.exit(0)
}
if (values.check) {
	throw new Error(
		`${npmPackage} ${current} does not match ${goModule} ${goVersion} in go.mod. Run the fix task.`,
	)
}

dependencies[npmPackage] = expected
await writeFile(manifestPath, `${JSON.stringify(manifest, null, "\t")}\n`)
await run(frontend, "bun", "install")
console.log(`Pinned ${npmPackage} ${current} -> ${expected} to match go.mod`)
