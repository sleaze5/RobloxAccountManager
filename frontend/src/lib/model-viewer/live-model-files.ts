import { CACHE } from "roavatar-renderer"
import { accountBackend } from "../backend/bridge"
import type { AssetRepresentation } from "../backend/bridge"

// The avatar renderer reads a file in two requests, as from the asset
// delivery API: it asks where the file of an asset is, then downloads the file
// from that location. The backend downloads the file at the first request, and
// the answer names a location in this module, which holds the file until the
// renderer reads it. No request from the renderer reaches Roblox directly.
const locationPrefix = "live-model-file:",
	assetDeliveryHost = "assetdelivery.roblox.com",
	assetDeliveryPaths = ["/v2/asset", "/v2/asset/"],
	// The renderer writes the priority list into the address as raw base64,
	// whose plus signs a URL parser would read as spaces.
	priorityListPattern = /[?&]contentRepresentationPriorityList=([^&#]*)/

const fileBase64ByLocation = new Map<string, string>()
const pendingDownloads = new Set<
	ReturnType<typeof accountBackend.GetAvatarRenderFile>
>()
let lastLocationNumber = 0,
	requestingAccountId = 0

export function setLiveModelAccount(accountId: number): void {
	if (requestingAccountId === accountId) return
	requestingAccountId = accountId
	for (const request of pendingDownloads) void request.cancel()
	pendingDownloads.clear()
	fileBase64ByLocation.clear()
	clearLiveModelCache()
}

// RoAvatar keys its caches by asset rather than account, and caches failed reads.
export function clearLiveModelCache(): void {
	for (const cache of [CACHE.AssetBuffer, CACHE.RBX, CACHE.Mesh, CACHE.Image]) {
		for (const key of cache.map.keys()) cache.delete(key)
	}
}

// fetchLiveModelFile answers a request of the renderer. It answers a request
// for an asset with the location of the downloaded file, and a request for a
// location with the file. It refuses every other request.
export async function fetchLiveModelFile(
	input: RequestInfo | URL,
	init?: RequestInit,
): Promise<Response> {
	const request = input instanceof Request ? input : undefined,
		method = init?.method ?? request?.method ?? "GET"
	if (method !== "GET") return new Response(null, { status: 405 })
	const address = input instanceof Request ? input.url : input.toString()
	const fileBase64 = takeFile(address)
	if (fileBase64 !== undefined) return new Response(bytesOfBase64(fileBase64))
	const assetRequest = assetRequestOf(
		address,
		new Headers(init?.headers ?? request?.headers),
	)
	if (!assetRequest || requestingAccountId <= 0) {
		return new Response(null, { status: 403 })
	}
	return await downloadLiveModelFile(
		assetRequest.assetId,
		assetRequest.representation,
	)
}

async function downloadLiveModelFile(
	assetId: number,
	representation: AssetRepresentation,
): Promise<Response> {
	const download = accountBackend.GetAvatarRenderFile(
		requestingAccountId,
		assetId,
		representation,
	)
	pendingDownloads.add(download)
	try {
		const downloadedBase64 = await download
		if (!pendingDownloads.has(download)) return new Response(null, { status: 410 })
		if (!downloadedBase64) return new Response(null, { status: 502 })
		const location = `${locationPrefix}${++lastLocationNumber}`
		fileBase64ByLocation.set(location, downloadedBase64)
		return Response.json({ locations: [{ location }] })
	} catch {
		// The renderer leaves out the asset, and the backend logs the failure.
		return new Response(null, { status: 502 })
	} finally {
		pendingDownloads.delete(download)
	}
}

// liveModelImageUrl turns the location of an image file into an address that
// an image element loads.
export async function liveModelImageUrl(location: string): Promise<string> {
	const fileBase64 = takeFile(location)
	if (fileBase64 === undefined) return "data:,"
	return `data:${imageTypeOf(fileBase64)};base64,${fileBase64}`
}

function takeFile(location: string): string | undefined {
	if (!location.startsWith(locationPrefix)) return undefined
	const fileBase64 = fileBase64ByLocation.get(location)
	fileBase64ByLocation.delete(location)
	return fileBase64
}

// assetRequestOf reads the asset and its representation from a request for
// the location of an asset file.
function assetRequestOf(
	address: string,
	headers: Headers,
): { assetId: number; representation: AssetRepresentation } | null {
	let requestUrl: URL
	try {
		requestUrl = new URL(address)
	} catch {
		return null
	}
	const assetId = Number(requestUrl.searchParams.get("id"))
	if (
		requestUrl.protocol !== "https:" ||
		requestUrl.hostname !== assetDeliveryHost ||
		requestUrl.port ||
		requestUrl.username ||
		requestUrl.password ||
		!assetDeliveryPaths.includes(requestUrl.pathname) ||
		!Number.isSafeInteger(assetId) ||
		assetId <= 0
	) {
		return null
	}
	return {
		assetId,
		representation: {
			format: headers.get("Roblox-AssetFormat") ?? "",
			priorityList: priorityListPattern.exec(address)?.[1] ?? "",
		},
	}
}

function bytesOfBase64(fileBase64: string): Uint8Array<ArrayBuffer> {
	const binary = atob(fileBase64),
		bytes = new Uint8Array(binary.length)
	for (let index = 0; index < binary.length; index++) {
		bytes[index] = binary.charCodeAt(index)
	}
	return bytes
}

// imageTypeOf names the type of an image by the base64 of its first bytes.
// Roblox stores textures as PNG or JPEG.
function imageTypeOf(fileBase64: string): string {
	return fileBase64.startsWith("/9j/") ? "image/jpeg" : "image/png"
}
