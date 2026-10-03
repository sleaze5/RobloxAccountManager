<script lang="ts">
	import { Events } from "@wailsio/runtime"
	import { onMount } from "svelte"
	import { accountBackend, type LaunchConfirmation } from "../backend/bridge"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"

	let pending = $state<LaunchConfirmation | null>(null),
		submitting = $state(false),
		error = $state("")
	let revision = 0

	async function refresh(): Promise<void> {
		const current = ++revision
		try {
			const next = await accountBackend.GetLaunchConfirmation()
			if (current !== revision) return
			pending = next.id ? next : null
			error = ""
		} catch {
			if (current === revision && pending) {
				error = "The launch could not be checked. Cancel and try joining again."
			}
		}
	}

	async function respond(approved: boolean): Promise<void> {
		if (!pending || submitting) return
		const id = pending.id
		submitting = true
		error = ""
		try {
			await accountBackend.ResolveLaunchConfirmation(id, approved)
			if (pending?.id === id) pending = null
		} catch {
			error = "Your choice could not be sent. Try again."
		} finally {
			submitting = false
		}
	}

	onMount(() => {
		const remove = Events.On(
			"game:launch-confirmation-changed",
			() => void refresh(),
		)
		void refresh()
		return () => {
			revision += 1
			remove()
		}
	})
</script>

{#if pending}
	<div class="modal-backdrop" in:modalBackdropIn out:modalBackdropOut>
		<div
			class="modal-card confirm-card"
			role="alertdialog"
			tabindex="-1"
			aria-modal="true"
			aria-labelledby="replace-roblox-title"
			aria-describedby="replace-roblox-description"
			use:modalFocus
			in:modalCardIn
			out:modalCardOut
			onkeydown={(event) => {
				if (event.key === "Escape") {
					event.preventDefault()
					event.stopPropagation()
					void respond(false)
				}
			}}>
			<div class="modal-heading compact-heading">
				<div>
					<h2 id="replace-roblox-title">
						{pending.accountCount > 1
							? "Launch without multi-instance?"
							: "Close Roblox and join?"}
					</h2>
					<p id="replace-roblox-description">
						{#if pending.accountCount > 1}
							{pending.multiInstanceEnabled
								? "Multi-instance is not ready."
								: "Multi-instance is off."}
							The {pending.accountCount} selected accounts may replace each
							other instead of staying open together.
							{#if pending.multiInstanceMessage}To get ready: {pending.multiInstanceMessage}.{/if}
							You will still be asked before running Roblox games are closed.
						{:else}
							{pending.multiInstanceEnabled
								? "Roblox must restart before multi-instance can work."
								: "Multi-instance is off."}
							This will close {pending.processCount === 1
								? "the running Roblox game"
								: `all ${pending.processCount} running Roblox games`} and
							join the new game. Unsaved progress may be lost.
						{/if}
					</p>
				</div>
			</div>
			{#if error}<div class="modal-error" role="alert">{error}</div>{/if}
			<div class="modal-actions">
				<button
					type="button"
					disabled={submitting}
					onclick={() => void respond(false)}>Cancel</button>
				<button
					class="danger-button"
					type="button"
					disabled={submitting}
					onclick={() => void respond(true)}
					>{pending.accountCount > 1
						? "Launch anyway"
						: "Close Roblox and join"}</button>
			</div>
		</div>
	</div>
{/if}
