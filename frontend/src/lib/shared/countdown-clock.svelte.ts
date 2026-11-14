// CountdownClock keeps the current time for a countdown. It ticks once a
// second only while the end is still ahead.
export class CountdownClock {
	nowMs = $state(Date.now())

	track(endsAtMs: number): () => void {
		this.nowMs = Date.now()
		if (endsAtMs <= this.nowMs) return () => undefined
		const timer = window.setInterval(() => {
			this.nowMs = Date.now()
			if (this.nowMs >= endsAtMs) window.clearInterval(timer)
		}, 1000)
		return () => window.clearInterval(timer)
	}
}
