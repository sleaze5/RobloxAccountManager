<script lang="ts">
	import vaultIcon from "@phosphor-icons/core/regular/vault.svg?raw"
	import { getProfilePreviews } from "../shared/profile-preview"
	import type { LabeledBadge } from "../shared/labeled-badge"
	import ProfileCard from "../shared/ProfileCard.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import { accountBackend, TagKind } from "../backend/bridge"

	interface LogUser {
		username: string
		displayName: string
		favorite: boolean
		inVault: boolean | null
	}

	let { userId }: { userId: string } = $props()

	const previews = getProfilePreviews(),
		notInVaultBadge: LabeledBadge = {
			label: "Not in your vault",
			icon: vaultIcon,
			isStrong: false,
		}

	let user = $state<LogUser | null>(null),
		isUserUnavailable = $state(false),
		avatarUrl = $state("")

	$effect(() => {
		const id = Number(userId)
		user = null
		isUserUnavailable = false
		avatarUrl = ""
		if (!Number.isSafeInteger(id) || id <= 0) return
		const state = { current: true }
		void load(id, state)
		void loadAvatar(id, state)
		return () => {
			state.current = false
		}
	})

	async function load(id: number, state: { current: boolean }): Promise<void> {
		let inVault: boolean | null = null
		try {
			const account = await accountBackend.FindAccountByRobloxUserID(id)
			if (!state.current) return
			if (account) {
				user = {
					username: account.username,
					displayName: account.displayName,
					favorite:
						account.tags?.some(
							(tag) => tag.kind === TagKind.TagKindFavorite,
						) ?? false,
					inVault: true,
				}
				return
			}
			inVault = false
		} catch {
			// The Roblox lookup below still names the user. Without the vault
			// lookup, the row leaves out whether the user is in the vault.
		}
		try {
			const found = await accountBackend.GetRobloxUser(id)
			if (state.current) {
				user = {
					username: found.username,
					displayName: found.displayName,
					favorite: false,
					inVault,
				}
			}
		} catch {
			if (state.current) isUserUnavailable = true
		}
	}

	async function loadAvatar(id: number, state: { current: boolean }): Promise<void> {
		try {
			const [headshot] = (await accountBackend.GetAvatarHeadshots([id])) ?? []
			if (state.current) avatarUrl = headshot?.imageUrl ?? ""
		} catch {
			// ProfileImage shows the user icon in place of a missing avatar.
		}
	}
</script>

<div class="w-72 max-w-full" aria-busy={!user && !isUserUnavailable}>
	{#if user}
		{@const shownUser = user}
		<ProfileCard
			kind="User"
			imageUrl={avatarUrl}
			title={shownUser.displayName || shownUser.username}
			subtitle={`@${shownUser.username}`}
			isFavorite={shownUser.favorite}
			badges={shownUser.inVault === false ? [notInVaultBadge] : []}
			action={{
				label: "Show profile",
				open: () =>
					previews.show({
						kind: "User",
						userId: Number(userId),
						username: shownUser.username,
						displayName: shownUser.displayName || shownUser.username,
					}),
			}} />
	{:else if isUserUnavailable}
		<ProfileCard
			kind="User"
			imageUrl={avatarUrl}
			title="Unavailable"
			subtitle={`User ID ${userId}`} />
	{:else}
		<div class="flex min-w-0 items-center gap-2 rounded-lg border border-line p-2">
			<span class="size-8 shrink-0 skeleton rounded-md"></span>
			<span class="flex min-w-0 flex-1 flex-col leading-snug">
				<span class="text-xs"
					><Skeleton className="w-28" label="Loading user…" /></span>
				<span class="text-2xs"
					><Skeleton className="w-20" label="Loading username…" /></span>
			</span>
		</div>
	{/if}
</div>
