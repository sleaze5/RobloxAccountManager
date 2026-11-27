<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import { Events } from "@wailsio/runtime"
	import { onMount } from "svelte"
	import { accountBackend, type LaunchConfirmation } from "../backend/bridge"
	import Dialog from "../shared/Dialog.svelte"

	let pending = $state<LaunchConfirmation | null>(null),
		submitting = $state(false),
		error = $state("")
	let revision = 0
	const isAskingToSkipCleaning = $derived(
		pending !== null && pending.accountCount > 1 && pending.isCleaningBeforeLaunch,
	)

	async function refresh(): Promise<void> {
		const current = ++revision
		try {
			const next = await accountBackend.GetLaunchConfirmation()
			if (current !== revision) return
			pending = next.id ? next : null
			error = ""
		} catch {
			if (current === revision && pending) {
				error = "Could not check the launch. Cancel and try joining again."
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
			error = "Could not send your choice. Try again."
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

	notifyErrors("Roblox not launched", () => error)
</script>

{#if pending}
	<Dialog
		class="max-w-sm text-left"
		role="alertdialog"
		tabindex={-1}
		aria-labelledby="replace-roblox-title"
		aria-describedby="replace-roblox-description"
		onkeydown={(event) => {
			if (event.key === "Escape") {
				event.preventDefault()
				event.stopPropagation()
				void respond(false)
			}
		}}>
		<div class="dialog-heading mb-2">
			<div>
				<h2 id="replace-roblox-title">
					{#if isAskingToSkipCleaning}
						Launch without cleaning?
					{:else if pending.accountCount > 1}
						Launch without Multi-client?
					{:else}
						Close Roblox and join?
					{/if}
				</h2>
				<p id="replace-roblox-description">
					{#if isAskingToSkipCleaning}
						Clean before launch allows one account at a time. Multiple
						clients is off, so the {pending.accountCount} accounts may replace
						each other.
					{:else if pending.accountCount > 1}
						{pending.multiClientEnabled
							? "Multi-client is not ready."
							: "Multi-client is off."}
						The {pending.accountCount} selected accounts may replace each other
						instead of staying open together.
						{#if pending.multiClientMessage}To get ready: {pending.multiClientMessage}.{/if}
						The app asks you before it closes running Roblox clients.
					{:else}
						{pending.multiClientEnabled
							? "Roblox must restart before Multi-client can work."
							: "Multi-client is off."}
						This will close {pending.processCount === 1
							? "the running Roblox client"
							: `all ${pending.processCount} running Roblox clients`} and join
						the new game. Unsaved progress may be lost.
						{#if pending.isCleaningBeforeLaunch}Roblox is cleaned first.{/if}
					{/if}
				</p>
			</div>
		</div>
		<div class="dialog-actions">
			<button
				class="button"
				type="button"
				disabled={submitting}
				onclick={() => void respond(false)}>Cancel</button>
			<button
				class={isAskingToSkipCleaning ? "button-warning" : "button-danger"}
				type="button"
				disabled={submitting}
				onclick={() => void respond(true)}>
				{#if isAskingToSkipCleaning}
					Launch without cleaning
				{:else if pending.accountCount > 1}
					Launch anyway
				{:else}
					Close Roblox and join
				{/if}
			</button>
		</div>
	</Dialog>
{/if}
