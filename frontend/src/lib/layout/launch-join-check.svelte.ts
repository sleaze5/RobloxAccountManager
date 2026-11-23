import { accountBackend } from "../backend/bridge"
import type { AccountJoinCheck, LaunchInput } from "../backend/bridge"

// The panel waits this long after the last change to the join fields, so that
// typing a place ID costs one check instead of one per digit.
const checkDelayMs = 600

// LaunchJoinCheck asks how joining the launch target would end for the
// selected accounts. It only warns: launching never waits for it.
export class LaunchJoinCheck {
	checks = $state.raw<AccountJoinCheck[]>([])
	isChecking = $state(false)
	#timer = 0
	#request: ReturnType<typeof accountBackend.CheckLaunchJoins> | null = null

	// schedule replaces the shown checks, which describe the previous target.
	schedule(accountIds: number[], input: LaunchInput): void {
		this.cancel()
		if (accountIds.length === 0) return
		this.#timer = window.setTimeout(
			() => void this.#check(accountIds, input),
			checkDelayMs,
		)
	}

	cancel(): void {
		clearTimeout(this.#timer)
		void this.#request?.cancel()
		this.#request = null
		this.checks = []
		this.isChecking = false
	}

	async #check(accountIds: number[], input: LaunchInput): Promise<void> {
		const request = accountBackend.CheckLaunchJoins(accountIds, input)
		this.#request = request
		this.isChecking = true
		try {
			const checks = await request
			if (this.#request === request) this.checks = checks ?? []
		} catch {
			// The launch reports invalid input, and a check that fails says
			// nothing about the join, so neither shows a warning.
		} finally {
			if (this.#request === request) {
				this.#request = null
				this.isChecking = false
			}
		}
	}
}
