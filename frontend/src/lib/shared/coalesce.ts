// coalesce runs one load at a time. A call during a load waits for it and for
// one more load, so every caller sees data that is at least as new as its call.
export function coalesce(load: () => Promise<void>): () => Promise<void> {
	let running: Promise<void> | null = null,
		queued = false
	const run = async (): Promise<void> => {
		queued = false
		await load()
		if (queued) await run()
	}
	return async () => {
		if (running) {
			queued = true
			return running
		}
		running = run()
		try {
			await running
		} finally {
			running = null
		}
	}
}
