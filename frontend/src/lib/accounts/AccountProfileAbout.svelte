<script lang="ts">
	import { Browser } from "@wailsio/runtime"
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import clockCounterClockwiseIcon from "@phosphor-icons/core/regular/clock-counter-clockwise.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import pencilSimpleIcon from "@phosphor-icons/core/regular/pencil-simple.svg?raw"
	import Icon from "../shared/Icon.svelte"
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

<section class="flex flex-col gap-2" aria-labelledby="profile-about-heading">
	<div class="flex items-center gap-1">
		<h3 id="profile-about-heading" class="text-xs font-semibold text-fg-muted">
			About
		</h3>
		<button
			class="btn-ghost-sm"
			type="button"
			aria-label="Edit description"
			data-tooltip="Edit description"
			disabled={disabled || snapshot.unavailable?.includes("about")}
			onclick={() => onEditDescription(snapshot.description)}>
			<Icon svg={pencilSimpleIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	{#if snapshot.unavailable?.includes("about") && !snapshot.description}
		<p class="text-xs text-danger-muted" role="alert">
			The description is unavailable. Refresh to retry.
		</p>
	{:else}
		<pre class="text-block">{snapshot.description || "No description yet."}</pre>
	{/if}
	{#if snapshot.socialLinks?.length || snapshot.previousNames?.length}
		<dl class="facts">
			{#if snapshot.socialLinks?.length}
				<div>
					<dt>
						<Icon svg={linkIcon} size={16} aria-hidden="true" />Social links
					</dt>
					<dd>
						{#each snapshot.socialLinks as link (link.name)}
							<button
								class="btn"
								type="button"
								aria-label={`Open ${link.name} (${link.url})`}
								data-tooltip={link.url}
								aria-busy={opening === link.name}
								disabled={opening !== ""}
								onclick={() => void openLink(link)}>
								<Icon
									svg={arrowSquareOutIcon}
									size={16}
									aria-hidden="true" />
								{link.name}
							</button>
						{/each}
					</dd>
				</div>
			{/if}
			{#if snapshot.previousNames?.length}
				<div>
					<dt>
						<Icon
							svg={clockCounterClockwiseIcon}
							size={16}
							aria-hidden="true" />Previous names
					</dt>
					<dd><span>{snapshot.previousNames.join(", ")}</span></dd>
				</div>
			{/if}
		</dl>
	{/if}
</section>
