import { AccountProfileSection, accountBackend } from "../backend/bridge"
import type { CancellableRequest } from "../shared/cancellable-request"

export async function readRobuxBalance(
	accountId: number,
	track: <T>(request: CancellableRequest<T>) => Promise<T>,
): Promise<number | null> {
	try {
		const profile = await track(
			accountBackend.GetAccountProfile(
				accountId,
				AccountProfileSection.ProfileSectionRobux,
			),
		)
		return profile.robux
	} catch {
		// The backend logs the failure.
		return null
	}
}
