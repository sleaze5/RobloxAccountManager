<script lang="ts">
	import arrowsCounterClockwiseIcon from "@phosphor-icons/core/regular/arrows-counter-clockwise.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import copyIcon from "@phosphor-icons/core/regular/copy.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { onMount, tick, untrack } from "svelte"
	import { PrivateServerOption } from "../backend/bridge"
	import type { PrivateServerSettings } from "../backend/bridge"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
		replayValidationNudge,
	} from "../shared/presence"
	import { PrivateServerState } from "./private-server-state.svelte"

	let {
		accountId,
		username,
		serverId,
		name,
		notifications,
		onChanged,
		onClose,
	}: {
		accountId: number
		username: string
		serverId: number
		name: string
		notifications: NotificationCenter
		onChanged: (settings: PrivateServerSettings) => void
		onClose: () => void
	} = $props()

	const server = untrack(
			() => new PrivateServerState(accountId, serverId, notifications, onChanged),
		),
		id = $props.id(),
		dates = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }),
		numbers = new Intl.NumberFormat()
	let phase = $state<"manage" | "reset-link">("manage"),
		nameValue = $state(untrack(() => name)),
		userValue = $state(""),
		card = $state<HTMLDivElement>()
	const settings = $derived(server.settings),
		busy = $derived(server.pending !== null),
		locked = $derived(busy && server.pending !== "load"),
		title = $derived(settings?.name || name || "Private server"),
		renewal = $derived(renewalText(settings)),
		users = $derived(settings?.users ?? []),
		toggles = $derived([
			{
				option: PrivateServerOption.PrivateServerActive,
				label: "Active",
				description: "Let people join this server.",
				checked: settings?.active ?? false,
			},
			{
				option: PrivateServerOption.PrivateServerFriendsAllowed,
				label: "Friends can join",
				description: `Friends of @${username} can join without the link.`,
				checked: settings?.friendsAllowed ?? false,
			},
			{
				option: PrivateServerOption.PrivateServerRenews,
				label: "Renew monthly",
				description: renewal,
				checked: settings?.renews ?? false,
			},
		])

	onMount(() => {
		void load()
		return () => server.dispose()
	})

	async function load(): Promise<void> {
		if ((await server.load()) && server.settings) nameValue = server.settings.name
	}

	function renewalText(current: PrivateServerSettings | null): string {
		if (!current?.expiresAtMs) {
			return current?.renews
				? "Renews every month."
				: "Ends when the current month runs out."
		}
		const date = dates.format(current.expiresAtMs),
			price = current.price ? ` for ${numbers.format(current.price)} Robux` : ""
		if (current.expired) return `Expired on ${date}.`
		return current.renews ? `Renews on ${date}${price}.` : `Ends on ${date}.`
	}

	function close(): void {
		if (locked) return
		if (phase === "reset-link") {
			void showManage()
			return
		}
		onClose()
	}

	async function showManage(): Promise<void> {
		phase = "manage"
		await tick()
		card?.querySelector<HTMLElement>("[data-reset-link]")?.focus()
	}

	async function showReset(): Promise<void> {
		phase = "reset-link"
		await tick()
		card?.querySelector<HTMLElement>("[data-confirm-reset]")?.focus()
	}

	async function rename(): Promise<void> {
		const text = nameValue.trim()
		if (!text || text === settings?.name) {
			replayValidationNudge(card)
			return
		}
		if (await server.rename(text)) nameValue = server.settings?.name ?? text
		else replayValidationNudge(card)
	}

	async function addUser(): Promise<void> {
		if (!userValue.trim()) {
			replayValidationNudge(card)
			return
		}
		if (await server.addUser(userValue)) userValue = ""
		else replayValidationNudge(card)
	}

	async function resetLink(): Promise<void> {
		await server.resetLink()
		await showManage()
	}

	function userName(user: { displayName: string; username: string; userId: number }) {
		return user.displayName || user.username || `User ${user.userId}`
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") close()
	}} />

<div class="dialog-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="dialog-card max-w-md text-left"
		role={phase === "reset-link" ? "alertdialog" : "dialog"}
		aria-modal="true"
		aria-labelledby={`${id}-title`}
		aria-describedby={`${id}-description`}
		aria-busy={server.pending === "load"}
		bind:this={card}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		{#if phase === "reset-link"}
			<div class="dialog-heading">
				<div>
					<h2 id={`${id}-title`}>Reset join link?</h2>
					<p id={`${id}-description`}>{title}</p>
				</div>
				<button
					type="button"
					aria-label="Cancel resetting the join link"
					disabled={busy}
					onclick={close}>
					<Icon svg={xIcon} size={14} aria-hidden="true" />
				</button>
			</div>
			<div class="notice-warning">
				The current link stops working. People who already joined keep access.
			</div>
			<div class="dialog-actions">
				<button class="btn" type="button" disabled={busy} onclick={close}
					>Cancel</button>
				<button
					class="btn-warning"
					type="button"
					data-confirm-reset
					disabled={busy}
					onclick={() => void resetLink()}>
					{server.pending === "link" ? "Resetting…" : "Reset link"}
				</button>
			</div>
		{:else}
			<div class="dialog-heading">
				<div class="min-w-0">
					<h2 id={`${id}-title`} class="wrap-anywhere">{title}</h2>
					<p id={`${id}-description`}>
						{settings?.gameName ? `${settings.gameName} - ` : ""}@{username}
					</p>
				</div>
				<button
					type="button"
					aria-label="Close private server"
					disabled={locked}
					onclick={close}>
					<Icon svg={xIcon} size={14} aria-hidden="true" />
				</button>
			</div>

			{#if !settings}
				{#if server.pending === "load"}
					<div class="empty-state" role="status">
						<Icon
							svg={circleNotchIcon}
							class="spinner"
							size={28}
							aria-hidden="true" />
						<h2>Loading private server…</h2>
					</div>
				{:else}
					<div class="empty-state" role="alert">
						<Icon svg={warningIcon} size={28} aria-hidden="true" />
						<h2>Private server is unavailable</h2>
						<p>{server.errorFor("load")}</p>
						<button class="btn" type="button" onclick={() => void load()}
							>Try again</button>
					</div>
				{/if}
			{:else}
				<form
					novalidate
					class="dialog-field mt-0"
					onsubmit={(event) => {
						event.preventDefault()
						void rename()
					}}>
					<label for={`${id}-name`}>Name</label>
					<div class="flex gap-2">
						<input
							id={`${id}-name`}
							class="field min-w-0 flex-1"
							type="text"
							autocomplete="off"
							maxlength={50}
							aria-invalid={server.errorFor("rename")
								? "true"
								: undefined}
							aria-describedby={server.errorFor("rename")
								? `${id}-name-error`
								: undefined}
							disabled={busy}
							bind:value={nameValue} />
						<button
							class="btn h-8"
							type="submit"
							disabled={busy ||
								!nameValue.trim() ||
								nameValue.trim() === settings.name}>
							{server.pending === "rename" ? "Renaming…" : "Rename"}
						</button>
					</div>
					{#if server.errorFor("rename")}
						<small id={`${id}-name-error`} class="text-xs text-danger-muted"
							>{server.errorFor("rename")}</small>
					{/if}
				</form>

				<div class="settings-section mt-3">
					<div class="settings-rows">
						{#each toggles as toggle (toggle.option)}
							<div class="settings-row">
								<div class="settings-row-copy">
									<strong id={`${id}-${toggle.option}`}
										>{toggle.label}</strong>
									<span id={`${id}-${toggle.option}-description`}
										>{toggle.description}</span>
									{#if server.errorFor(toggle.option)}
										<span class="text-danger-muted" role="alert"
											>{server.errorFor(toggle.option)}</span>
									{/if}
								</div>
								{#if server.pending === toggle.option}
									<Icon
										svg={circleNotchIcon}
										class="spinner text-fg-muted"
										size={14}
										aria-hidden="true" />
								{/if}
								<label class="switch">
									<input
										type="checkbox"
										role="switch"
										aria-labelledby={`${id}-${toggle.option}`}
										aria-describedby={`${id}-${toggle.option}-description`}
										checked={toggle.checked}
										disabled={busy}
										onchange={(event) => {
											const enabled = event.currentTarget.checked
											event.currentTarget.checked = toggle.checked
											void server.setOption(
												toggle.option,
												enabled,
											)
										}} />
									<span aria-hidden="true"></span>
								</label>
							</div>
						{/each}
					</div>
				</div>

				<section
					class="mt-4 flex flex-col gap-2"
					aria-labelledby={`${id}-link`}>
					<h3 id={`${id}-link`} class="text-xs font-semibold text-fg-muted">
						Join link
					</h3>
					{#if settings.joinLink}
						<p class="text-block text-2xs">{settings.joinLink}</p>
					{:else}
						<p class="text-xs text-fg-faint">
							This server has no join link. Reset the link to create one.
						</p>
					{/if}
					<div class="flex flex-wrap gap-2">
						<button
							class="btn"
							type="button"
							disabled={!settings.joinLink}
							onclick={() => void server.copyLink()}>
							<Icon svg={copyIcon} size={14} aria-hidden="true" />
							Copy link
						</button>
						<button
							class="btn"
							type="button"
							data-reset-link
							disabled={busy}
							onclick={() => void showReset()}>
							<Icon
								svg={arrowsCounterClockwiseIcon}
								size={14}
								aria-hidden="true" />
							Reset link
						</button>
					</div>
					{#if server.errorFor("link")}
						<div class="notice-error" role="alert">
							{server.errorFor("link")}
						</div>
					{/if}
				</section>

				<section
					class="mt-4 flex flex-col gap-2"
					aria-labelledby={`${id}-users`}>
					<h3 id={`${id}-users`} class="text-xs font-semibold text-fg-muted">
						Allowed users
						{#if users.length}<span class="font-normal"
								>({users.length})</span
							>{/if}
					</h3>
					{#if users.length}
						<div class="settings-section">
							<ul class="settings-rows" aria-label="Allowed users">
								{#each users as user (user.userId)}
									<li
										class="settings-row animate-enter justify-start gap-3">
										<span
											class="grid size-7 shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
											aria-hidden="true">
											<ProfileImage url={user.imageUrl} />
										</span>
										<div class="settings-row-copy">
											<strong class="truncate"
												>{userName(user)}</strong>
											<span
												>{user.username
													? `@${user.username}`
													: user.userId}</span>
										</div>
										<button
											class="btn"
											type="button"
											aria-label={`Remove ${userName(user)}`}
											disabled={busy}
											onclick={() =>
												void server.removeUser(user)}>
											{#if server.pending === `remove-${user.userId}`}
												<Icon
													svg={circleNotchIcon}
													class="spinner"
													size={14}
													aria-hidden="true" />
											{/if}
											Remove
										</button>
									</li>
								{/each}
							</ul>
						</div>
					{:else}
						<p class="text-xs text-fg-faint">
							No one else is allowed. Add a user to let them join.
						</p>
					{/if}
					{#each users as user (user.userId)}
						{#if server.errorFor(`remove-${user.userId}`)}
							<div class="notice-error" role="alert">
								{server.errorFor(`remove-${user.userId}`)}
							</div>
						{/if}
					{/each}
					<form
						novalidate
						class="dialog-field mt-1"
						onsubmit={(event) => {
							event.preventDefault()
							void addUser()
						}}>
						<label for={`${id}-user`}>Add a user</label>
						<div class="flex gap-2">
							<input
								id={`${id}-user`}
								class="field min-w-0 flex-1"
								type="text"
								autocomplete="off"
								spellcheck="false"
								placeholder="Username"
								aria-invalid={server.errorFor("add-user")
									? "true"
									: undefined}
								aria-describedby={`${id}-user-help`}
								disabled={busy}
								bind:value={userValue} />
							<button
								class="btn h-8"
								type="submit"
								disabled={busy || !userValue.trim()}>
								{server.pending === "add-user" ? "Adding…" : "Add user"}
							</button>
						</div>
						{#if server.errorFor("add-user")}
							<small
								id={`${id}-user-help`}
								class="text-xs text-danger-muted"
								>{server.errorFor("add-user")}</small>
						{:else}
							<small id={`${id}-user-help`} class="text-xs text-fg-muted"
								>Enter a username, or id=123 for a user ID.</small>
						{/if}
					</form>
				</section>
			{/if}

			<div class="dialog-actions">
				<button class="btn" type="button" disabled={locked} onclick={close}
					>Close</button>
			</div>
		{/if}
	</div>
</div>
