import { setTimeout as sleep } from "node:timers/promises"

const port = Number(process.argv[2])
if (!Number.isInteger(port) || port < 1 || port > 65_535) {
	throw new Error("Pass the Vite port as the first argument")
}
const url = `http://127.0.0.1:${port}/`
const timeoutMilliseconds = 60_000
const deadline = Date.now() + timeoutMilliseconds

async function isViteReady(): Promise<boolean> {
	return fetch(url, { signal: AbortSignal.timeout(1000) }).then(
		(response) => response.ok,
		() => false,
	)
}

async function waitForVite(): Promise<void> {
	if (await isViteReady()) return
	if (Date.now() >= deadline) {
		throw new Error(
			`The Vite dev server did not become ready at ${url} within 60 seconds`,
		)
	}
	await sleep(100)
	return waitForVite()
}

await waitForVite()
