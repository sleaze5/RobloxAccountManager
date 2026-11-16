import { AccountProfileSection, accountBackend } from "../backend/bridge"
import type { CancellableRequest } from "../shared/cancellable-request"

// readRobuxBalance reads the Robux balance of the account for a purchase
// confirmation. It returns null when the balance is unavailable, and the
// backend logs why. track receives the request, so that the caller can cancel
// it.
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
		return null
	}
}
