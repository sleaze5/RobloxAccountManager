<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import eyeIcon from "@phosphor-icons/core/regular/eye.svg?raw"
	import eyeSlashIcon from "@phosphor-icons/core/regular/eye-slash.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { onDestroy } from "svelte"
	import { accountBackend } from "../backend/bridge"
	import { errorMessage } from "../shared/errors"

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
				error = errorMessage(cause, "Could not show the password.")
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
	class="-my-1 button-ghost-sm self-center"
	type="button"
	aria-label={password ? "Hide password" : "Show password"}
	data-tooltip={password ? "Hide password" : "Show password"}
	aria-busy={loading}
	onclick={() => {
		if (password) conceal()
		else if (!loading) void show()
	}}>
	{#if loading}
		<Icon svg={circleNotchIcon} class="spinner" size={14} aria-hidden="true" />
	{:else if password}
		<Icon svg={eyeSlashIcon} size={14} aria-hidden="true" />
	{:else}
		<Icon svg={eyeIcon} size={14} aria-hidden="true" />
	{/if}
</button>
{#if error}<small role="alert">{error}</small>{/if}
