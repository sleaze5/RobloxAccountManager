import { accountBackend } from "../backend/bridge"

export type CreatorKind = "User" | "Group"

const imageRequestsByCreatorKey = new Map<string, Promise<string>>()

// loadCreatorImage reads the avatar headshot of a user or the icon of a
// group once, and shares it between every card that shows the creator. A
// failed read resolves to an empty image, and the next card reads it again.
export function loadCreatorImage(
	kind: CreatorKind,
	creatorId: number,
): Promise<string> {
	const key = `${kind}:${creatorId}`,
		cachedRequest = imageRequestsByCreatorKey.get(key)
	if (cachedRequest) return cachedRequest
	const request = (async () => {
		try {
			if (kind === "Group") {
				const icons = await accountBackend.GetGroupIcons([creatorId])
				return icons?.[0]?.imageUrl ?? ""
			}
			const headshots = await accountBackend.GetAvatarHeadshots([creatorId])
			return headshots?.[0]?.imageUrl ?? ""
		} catch {
			// The card shows the creator without an image, and the backend logs
			// the failure.
			imageRequestsByCreatorKey.delete(key)
			return ""
		}
	})()
	imageRequestsByCreatorKey.set(key, request)
	return request
}
