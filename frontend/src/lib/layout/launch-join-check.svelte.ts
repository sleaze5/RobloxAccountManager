import { accountBackend, JoinOutcome } from "../backend/bridge"
import type { AccountJoinCheck, LaunchInput } from "../backend/bridge"

// The check waits this long after the last change to the join fields, so that
// typing a place ID costs one check instead of one per digit.
const checkDelayMs = 600
// Launch waits at most this long for a check that is still running, then
// launches without it.
const maxLaunchWaitMs = 4000

// LaunchJoinCheck asks, while the join fields change, how joining the launch
// target would end for the selected accounts, so that Launch can warn without
// a wait. It only warns: launching never depends on it.
export class LaunchJoinCheck {
	#checks: AccountJoinCheck[] = []
	#timer = 0
	#scheduled: (() => Promise<void>) | null = null
	#running: Promise<void> | null = null
	#request: ReturnType<typeof accountBackend.CheckLaunchJoins> | null = null

	// schedule forgets the previous checks, which describe the previous target.
	schedule(accountIds: number[], input: LaunchInput): void {
		this.cancel()
		if (accountIds.length === 0) return
		this.#scheduled = () => this.#check(accountIds, input)
		this.#timer = window.setTimeout(() => this.#start(), checkDelayMs)
	}

	cancel(): void {
		clearTimeout(this.#timer)
		void this.#request?.cancel()
		this.#scheduled = this.#running = this.#request = null
		this.#checks = []
	}

	// readRestrictions returns the accounts that Roblox would likely refuse,
	// after the scheduled or running check ends or the wait runs out.
	async readRestrictions(): Promise<AccountJoinCheck[]> {
		clearTimeout(this.#timer)
		this.#start()
		const running = this.#running
		if (running) {
			let waitTimer = 0
			await Promise.race([
				running,
				new Promise<void>((resolve) => {
					waitTimer = window.setTimeout(resolve, maxLaunchWaitMs)
				}),
			])
			clearTimeout(waitTimer)
		}
		return this.#checks.filter(
			(accountCheck) =>
				accountCheck.check.outcome === JoinOutcome.JoinOutcomeRestricted &&
				accountCheck.check.message,
		)
	}

	#start(): void {
		const scheduled = this.#scheduled
		this.#scheduled = null
		if (scheduled) this.#running = scheduled()
	}

	async #check(accountIds: number[], input: LaunchInput): Promise<void> {
		const request = accountBackend.CheckLaunchJoins(accountIds, input)
		this.#request = request
		try {
			const checks = await request
			if (this.#request === request) this.#checks = checks ?? []
		} catch {
			// The launch reports invalid input, and a check that fails says
			// nothing about the join, so neither warns.
		} finally {
			if (this.#request === request) this.#request = null
		}
	}
}
