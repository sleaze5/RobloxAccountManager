<script lang="ts">
	import { Browser } from "@wailsio/runtime"
	import ArrowSquareOutIcon from "phosphor-svelte/lib/ArrowSquareOutIcon"
	import ClockCounterClockwiseIcon from "phosphor-svelte/lib/ClockCounterClockwiseIcon"
	import LinkIcon from "phosphor-svelte/lib/LinkIcon"
	import PencilSimpleIcon from "phosphor-svelte/lib/PencilSimpleIcon"
	import type { AccountProfileSnapshot, UserSocialLink } from "../backend/bridge"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { errorMessage } from "../shared/errors"

	let {
		snapshot,
		disabled,
		notifications,
		onEditDescription,
	}: {
		snapshot: AccountProfileSnapshot
		disabled: boolean
		notifications: NotificationCenter
		onEditDescription: (description: string) => void
	} = $props()

	let opening = $state("")

	async function openLink(link: UserSocialLink): Promise<void> {
		if (opening) return
		opening = link.name
		try {
			await Browser.OpenURL(link.url)
		} catch (error) {
			notifications.show({
				id: `profile-social-link-${snapshot.accountId}-${link.name}`,
				title: "Could not open social link",
				message: errorMessage(error, "Try opening the link again."),
				tone: "error",
			})
		} finally {
			opening = ""
		}
	}
</script>

<section class="profile-section" aria-labelledby="profile-about-heading">
	<div class="profile-section-heading">
		<h3 id="profile-about-heading" class="profile-section-label">About</h3>
		<button
			class="icon-action compact"
			type="button"
			aria-label="Edit description"
			data-tooltip="Edit description"
			disabled={disabled || snapshot.unavailable?.includes("about")}
			onclick={() => onEditDescription(snapshot.description)}>
			<PencilSimpleIcon size={14} aria-hidden="true" />
		</button>
	</div>
	{#if snapshot.unavailable?.includes("about") && !snapshot.description}
		<p class="profile-read-error" role="alert">
			The description is unavailable. Refresh to retry.
		</p>
	{:else}
		<pre class="text-block">{snapshot.description || "No description yet."}</pre>
	{/if}
	{#if snapshot.socialLinks?.length || snapshot.previousNames?.length}
		<dl class="profile-facts">
			{#if snapshot.socialLinks?.length}
				<div>
					<dt><LinkIcon size={16} aria-hidden="true" />Social links</dt>
					<dd>
						{#each snapshot.socialLinks as link (link.name)}
							<button
								class="control-button text-button"
								type="button"
								aria-label={`Open ${link.name} (${link.url})`}
								data-tooltip={link.url}
								aria-busy={opening === link.name}
								disabled={opening !== ""}
								onclick={() => void openLink(link)}>
								<ArrowSquareOutIcon size={16} aria-hidden="true" />
								{link.name}
							</button>
						{/each}
					</dd>
				</div>
			{/if}
			{#if snapshot.previousNames?.length}
				<div>
					<dt>
						<ClockCounterClockwiseIcon
							size={16}
							aria-hidden="true" />Previous names
					</dt>
					<dd><span>{snapshot.previousNames.join(", ")}</span></dd>
				</div>
			{/if}
		</dl>
	{/if}
</section>
