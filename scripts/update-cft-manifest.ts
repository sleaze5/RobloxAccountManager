import { rename, rm } from "node:fs/promises"

const sourceURL =
	"https://googlechromelabs.github.io/chrome-for-testing/last-known-good-versions-with-downloads.json"
const outputPath = new URL("../internal/browser/runtime_manifest.json", import.meta.url)
const temporaryOutputPath = new URL(
	"../internal/browser/.runtime_manifest.json.tmp",
	import.meta.url,
)
const officialHost = "storage.googleapis.com"
const officialPrefix = "/chrome-for-testing-public/"
// Windows ARM64 runs the x64 build, so these are every download the application uses.
const platforms = ["linux64", "mac-arm64", "mac-x64", "win64"] as const

interface Download {
	platform: string
	url: string
}

interface SourceManifest {
	channels?: {
		Stable?: {
			version?: string
			downloads?: { chrome?: Download[] }
		}
	}
}

function validateURL(value: string, version: string, platform: string): string {
	const url = new URL(value)
	const expectedPrefix = `${officialPrefix}${version}/${platform}/`
	if (
		url.protocol !== "https:" ||
		url.hostname !== officialHost ||
		!url.pathname.startsWith(expectedPrefix)
	) {
		throw new Error(`Unexpected Chrome download URL: ${value}`)
	}
	return value
}

const response = await fetch(sourceURL)
if (!response.ok) {
	throw new Error(`CfT manifest request failed with ${response.status}`)
}
const source = (await response.json()) as SourceManifest
const stable = source.channels?.Stable
if (!stable?.version || !/^\d+\.\d+\.\d+\.\d+$/.test(stable.version)) {
	throw new Error("The Stable channel has no valid version")
}
const downloads = stable.downloads?.chrome ?? []
const urls: Record<string, string> = {}
// The application refuses to start on a platform without a download.
for (const platform of platforms) {
	const download = downloads.find((item) => item.platform === platform)
	if (!download) {
		throw new Error(`The Stable channel has no ${platform} Chrome download`)
	}
	urls[platform] = validateURL(download.url, stable.version, platform)
}

const manifest = {
	schemaVersion: 1,
	version: stable.version,
	sourceUrl: sourceURL,
	generatedAt: new Date().toISOString(),
	downloads: urls,
}
try {
	await Bun.write(temporaryOutputPath, `${JSON.stringify(manifest, null, "\t")}\n`)
	await rename(temporaryOutputPath, outputPath)
} catch (error) {
	await rm(temporaryOutputPath, { force: true })
	throw error
}
