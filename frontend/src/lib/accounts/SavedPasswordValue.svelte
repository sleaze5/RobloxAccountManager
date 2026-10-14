<script lang="ts">
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import EyeIcon from "phosphor-svelte/lib/EyeIcon"
	import EyeSlashIcon from "phosphor-svelte/lib/EyeSlashIcon"
	import { onDestroy } from "svelte"
	import { accountBackend } from "../backend/bridge"

	let { accountId }: { accountId: number } = $props()

	let password = $state(""),
		loading = $state(false),
		error = $state("")
	let request: ReturnType<typeof accountBackend.GetAccountPassword> | null = null

	function conceal(): void {
		void request?.cancel()
		request = null
		loading = false
		password = ""
		error = ""
	}

	async function show(): Promise<void> {
		const current = accountBackend.GetAccountPassword(accountId)
		request = current
		loading = true
		error = ""
		try {
			const value = await current
			if (request === current) password = value
		} catch (cause) {
			if (request === current) {
				error =
					cause instanceof Error && cause.message.trim()
						? cause.message
						: "Could not show the password."
			}
		} finally {
			if (request === current) {
				request = null
				loading = false
			}
		}
	}

	onDestroy(conceal)
</script>

<span>{password || "••••••••••"}</span>
<button
	class="icon-action profile-fact-action"
	type="button"
	aria-label={password ? "Hide password" : "Show password"}
	data-tooltip={password ? "Hide password" : "Show password"}
	aria-busy={loading}
	onclick={() => {
		if (password) conceal()
		else if (!loading) void show()
	}}>
	{#if loading}
		<CircleNotchIcon class="spinner" size={14} aria-hidden="true" />
	{:else if password}
		<EyeSlashIcon size={14} aria-hidden="true" />
	{:else}
		<EyeIcon size={14} aria-hidden="true" />
	{/if}
</button>
{#if error}<small role="alert">{error}</small>{/if}
