<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import gearSixIcon from "@phosphor-icons/core/regular/gear-six.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import starIcon from "@phosphor-icons/core/regular/star.svg?raw"
	import thumbsDownFillIcon from "@phosphor-icons/core/fill/thumbs-down-fill.svg?raw"
	import thumbsDownIcon from "@phosphor-icons/core/regular/thumbs-down.svg?raw"
	import thumbsUpFillIcon from "@phosphor-icons/core/fill/thumbs-up-fill.svg?raw"
	import thumbsUpIcon from "@phosphor-icons/core/regular/thumbs-up.svg?raw"
	import { GameVote } from "../backend/bridge"
	import type { Account } from "../accounts/account-model"
	import ConfirmationDialog from "../shared/ConfirmationDialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import RobloxActionStrip from "../shared/RobloxActionStrip.svelte"
	import { favoriteConfirmation } from "../shared/roblox-confirmations"
	import type { ConfirmationText } from "../shared/roblox-confirmations"
	import type {
		GameRobloxActions,
		GameRobloxChange,
	} from "./game-roblox-actions.svelte"

	let {
		actions,
		account,
		gameName,
		onManage,
	}: {
		actions: GameRobloxActions
		account: Account
		gameName: string
		// onManage opens Manage game. It is null when the account may not
		// change the game.
		onManage: (() => void) | null
	} = $props()

	let pendingChange = $state<GameRobloxChange | null>(null)

	const status = $derived(actions.status),
		isFavorite = $derived(status?.isFavorite ?? false),
		vote = $derived(status?.vote.vote ?? GameVote.GameVoteNone),
		isWaiting = $derived(!status && !actions.hasFailed),
		isFavoriteUnavailable = $derived(actions.isUnavailable("favorite")),
		isVoteUnavailable = $derived(actions.isUnavailable("vote")),
		voteRefusal = $derived(
			status && !status.vote.canVote ? status.vote.voteRefusal : "",
		),
		favoriteLabel = $derived(
			isFavoriteUnavailable
				? "Favorite unavailable"
				: isFavorite
					? "Remove from favorites"
					: "Add to favorites",
		),
		confirmation = $derived(pendingChange ? confirmationFor(pendingChange) : null)

	function voteLabel(target: GameVote): string {
		const isCurrent = vote === target,
			noun = target === GameVote.GameVoteLike ? "like" : "dislike"
		if (isVoteUnavailable) return "Vote unavailable"
		if (voteRefusal) return voteRefusal
		if (isCurrent) return `Remove ${noun}`
		return target === GameVote.GameVoteLike ? "Like game" : "Dislike game"
	}

	function confirmationFor(change: GameRobloxChange): ConfirmationText {
		if (change.kind === "favorite")
			return favoriteConfirmation(gameName, account.username, change.isFavorite)
		if (change.vote === GameVote.GameVoteLike) {
			return {
				title: "Like this game on Roblox?",
				description: `The like of @${account.username} counts in the rating of ${gameName}.`,
				confirmLabel: "Like game",
				progressLabel: "Liking…",
			}
		}
		if (change.vote === GameVote.GameVoteDislike) {
			return {
				title: "Dislike this game on Roblox?",
				description: `The dislike of @${account.username} counts in the rating of ${gameName}.`,
				confirmLabel: "Dislike game",
				progressLabel: "Disliking…",
			}
		}
		const noun = vote === GameVote.GameVoteLike ? "like" : "dislike"
		return {
			title: `Remove ${noun} on Roblox?`,
			description: `The ${noun} of @${account.username} no longer counts in the rating of ${gameName}.`,
			confirmLabel: `Remove ${noun}`,
			progressLabel: "Removing…",
		}
	}

	function requestVote(target: GameVote): void {
		pendingChange = {
			kind: "vote",
			vote: vote === target ? GameVote.GameVoteNone : target,
		}
	}

	async function confirm(): Promise<void> {
		if (!pendingChange) return
		if (await actions.save(pendingChange, gameName)) pendingChange = null
	}

	notifyErrors("Roblox change not saved", () => actions.saveError)
</script>

<RobloxActionStrip username={account.username}>
	<button
		type="button"
		aria-label={favoriteLabel}
		aria-pressed={isFavorite}
		data-tooltip={favoriteLabel}
		disabled={isWaiting || isFavoriteUnavailable || actions.isSaving}
		onclick={() => (pendingChange = { kind: "favorite", isFavorite: !isFavorite })}>
		{#if isFavorite}
			<Icon
				svg={starFillIcon}
				class="animate-pop text-accent-strong"
				size={14}
				aria-hidden="true" />
		{:else}
			<Icon svg={starIcon} size={14} aria-hidden="true" />
		{/if}
	</button>
	{#each [GameVote.GameVoteLike, GameVote.GameVoteDislike] as target (target)}
		{@const isCurrent = vote === target}
		{@const label = voteLabel(target)}
		<!-- A refused vote stays focusable, so that the keyboard reaches the
		tooltip that tells how to vote. -->
		<button
			type="button"
			aria-label={label}
			aria-pressed={isCurrent}
			aria-disabled={voteRefusal ? true : undefined}
			data-tooltip={label}
			disabled={isWaiting || isVoteUnavailable || actions.isSaving}
			onclick={() => {
				if (!voteRefusal) requestVote(target)
			}}>
			{#if target === GameVote.GameVoteLike}
				<Icon
					svg={isCurrent ? thumbsUpFillIcon : thumbsUpIcon}
					class={[isCurrent && "animate-pop"]}
					size={14}
					aria-hidden="true" />
			{:else}
				<Icon
					svg={isCurrent ? thumbsDownFillIcon : thumbsDownIcon}
					class={[isCurrent && "animate-pop"]}
					size={14}
					aria-hidden="true" />
			{/if}
		</button>
	{/each}
	{#if onManage}
		<button
			type="button"
			aria-label="Manage game"
			data-tooltip="Manage game"
			disabled={actions.isSaving}
			onclick={onManage}>
			<Icon svg={gearSixIcon} size={14} aria-hidden="true" />
		</button>
	{/if}
</RobloxActionStrip>

{#if pendingChange && confirmation}
	<ConfirmationDialog
		title={confirmation.title}
		description={confirmation.description}
		confirmLabel={confirmation.confirmLabel}
		progressLabel={confirmation.progressLabel}
		tone="primary"
		{account}
		isWorking={actions.isSaving}
		onConfirm={() => void confirm()}
		onCancel={() => (pendingChange = null)} />
{/if}
