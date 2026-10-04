<script lang="ts">
	import Star from "@lucide/svelte/icons/star"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import { accountBackend, TagKind } from "../backend/bridge"

	interface LogUser {
		username: string
		displayName: string
		favorite: boolean
		inVault: boolean | null
	}

	let { userId }: { userId: string } = $props()

	let user = $state<LogUser | null>(null),
		avatarUrl = $state("")

	$effect(() => {
		const id = Number(userId)
		user = null
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
		} catch {}
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
		} catch {}
	}

	async function loadAvatar(id: number, state: { current: boolean }): Promise<void> {
		try {
			const [headshot] = (await accountBackend.GetAvatarHeadshots([id])) ?? []
			if (state.current) avatarUrl = headshot?.imageUrl ?? ""
		} catch {}
	}
</script>

{#if user}
	<div class="logs-explorer-profile">
		<span class="logs-explorer-avatar" aria-hidden="true">
			<ProfileImage url={avatarUrl} />
		</span>
		<div class="logs-explorer-profile-copy">
			<strong class="account-display-name"
				><span>{user.displayName || user.username}</span
				>{#if user.favorite}<Star
						class="favorite-name-star"
						size={11}
						fill="currentColor"
						aria-label="Favorite" />{/if}</strong>
			<span
				>@{user.username}{user.inVault === false
					? " · Not in your vault"
					: ""}</span>
		</div>
	</div>
{/if}
