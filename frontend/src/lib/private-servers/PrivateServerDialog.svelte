<script lang="ts">
	import arrowsCounterClockwiseIcon from "@phosphor-icons/core/regular/arrows-counter-clockwise.svg?raw"
	import caretDownIcon from "@phosphor-icons/core/regular/caret-down.svg?raw"
	import pencilSimpleIcon from "@phosphor-icons/core/regular/pencil-simple.svg?raw"
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
	import { replayValidationNudge } from "../shared/presence"
	import Dialog from "../shared/Dialog.svelte"
	import { PrivateServerState } from "./private-server-state.svelte"
	import RenamePrivateServerDialog from "./RenamePrivateServerDialog.svelte"

	let {
		accountId,
		username,
		serverId,
		name,
		gameName,
		notifications,
		onChanged,
		onClose,
	}: {
		accountId: number
		username: string
		serverId: number
		name: string
		gameName: string
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
		renaming = $state(false),
		userValue = $state(""),
		card = $state<HTMLDivElement>()
	const settings = $derived(server.settings),
		busy = $derived(server.pending !== null),
		locked = $derived(busy && server.pending !== "load"),
		serverName = $derived(settings?.name || name || "Unnamed server"),
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
				description: "Friends of the owner can join without the link.",
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
		await server.load()
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
		if (locked || renaming) return
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

	async function closeRename(): Promise<void> {
		renaming = false
		await tick()
		card?.querySelector<HTMLElement>("[data-rename]")?.focus()
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

{#if renaming}
	<RenamePrivateServerDialog
		{server}
		{username}
		{notifications}
		onClose={() => void closeRename()} />
{:else}
	<Dialog
		class="max-w-md text-left"
		role={phase === "reset-link" ? "alertdialog" : "dialog"}
		aria-labelledby={`${id}-title`}
		aria-describedby={`${id}-description`}
		aria-busy={server.pending === "load"}
		bind:card>
		{#if phase === "reset-link"}
			<div class="dialog-heading">
				<div>
					<h2 id={`${id}-title`}>Reset join link?</h2>
					<p id={`${id}-description`}>{serverName}</p>
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
				<button class="button" type="button" disabled={busy} onclick={close}
					>Cancel</button>
				<button
					class="button-warning"
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
					<h2 id={`${id}-title`} class="wrap-anywhere">
						{gameName || settings?.gameName || "Private server"}
					</h2>
					<p id={`${id}-description`}>@{username}</p>
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
						<button class="button" type="button" onclick={() => void load()}
							>Try again</button>
					</div>
				{/if}
			{:else}
				<div class="settings-section">
					<div class="settings-rows">
						<div class="settings-row">
							<div class="settings-row-copy">
								<strong>Name</strong>
								<span class="truncate">{serverName}</span>
							</div>
							<button
								class="button"
								type="button"
								data-rename
								disabled={busy}
								onclick={() => (renaming = true)}>
								<Icon
									svg={pencilSimpleIcon}
									size={14}
									aria-hidden="true" />
								Rename
							</button>
						</div>
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
						<div class="settings-row">
							<div class="settings-row-copy">
								<strong>Join link</strong>
								{#if settings.joinLink}
									<span
										class="truncate font-mono text-2xs"
										data-tooltip={settings.joinLink}
										>{settings.joinLink}</span>
								{:else}
									<span
										>No join link. Reset the link to create one.</span>
								{/if}
								{#if server.errorFor("link")}
									<span class="text-danger-muted" role="alert"
										>{server.errorFor("link")}</span>
								{/if}
							</div>
							<div class="flex shrink-0 items-center gap-1.5">
								<button
									class="button-icon"
									type="button"
									aria-label="Copy join link"
									data-tooltip="Copy join link"
									disabled={!settings.joinLink}
									onclick={() => void server.copyLink()}>
									<Icon svg={copyIcon} size={16} aria-hidden="true" />
								</button>
								<button
									class="button"
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
						</div>
						<details class="group">
							<summary
								class="settings-row cursor-pointer border-b-0 transition-colors duration-100 group-open:border-b hover:bg-hover">
								<div class="settings-row-copy">
									<strong>Allowed users</strong>
									<span
										>{users.length
											? `${users.length} ${users.length === 1 ? "person" : "people"} can join without the link.`
											: "No one else is allowed yet."}</span>
								</div>
								<Icon
									svg={caretDownIcon}
									class="shrink-0 text-fg-muted transition-transform duration-100 group-open:rotate-180"
									size={16}
									aria-hidden="true" />
							</summary>
							<div class="animate-enter">
								{#if users.length}
									<ul aria-label="Allowed users">
										{#each users as user (user.userId)}
											<li
												class="settings-row justify-start gap-3 py-2">
												<span
													class="grid size-6 shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
													aria-hidden="true">
													<ProfileImage url={user.imageUrl} />
												</span>
												<span
													class="flex min-w-0 flex-1 items-baseline gap-1.5">
													<strong
														class="truncate text-sm font-semibold text-fg"
														>{userName(user)}</strong>
													<span
														class="truncate text-xs text-fg-muted"
														>{user.username
															? `@${user.username}`
															: user.userId}</span>
												</span>
												<button
													class="button"
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
								{/if}
								{#each users as user (user.userId)}
									{#if server.errorFor(`remove-${user.userId}`)}
										<div
											class="mx-3 mt-2.5 notice-error"
											role="alert">
											{server.errorFor(`remove-${user.userId}`)}
										</div>
									{/if}
								{/each}
								<form
									novalidate
									class={[
										"flex flex-col gap-1 px-3 py-2.5",
										users.length && "border-t border-line-subtle",
									]}
									onsubmit={(event) => {
										event.preventDefault()
										void addUser()
									}}>
									<label
										for={`${id}-user`}
										class="text-xs text-fg-muted">Add a user</label>
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
											class="button h-8"
											type="submit"
											disabled={busy || !userValue.trim()}>
											{server.pending === "add-user"
												? "Adding…"
												: "Add user"}
										</button>
									</div>
									{#if server.errorFor("add-user")}
										<small
											id={`${id}-user-help`}
											class="text-xs text-danger-muted"
											>{server.errorFor("add-user")}</small>
									{:else}
										<small
											id={`${id}-user-help`}
											class="text-xs text-fg-muted"
											>Enter a username, or id=123 for a user ID.</small>
									{/if}
								</form>
							</div>
						</details>
					</div>
				</div>
			{/if}

			<div class="dialog-actions">
				<button class="button" type="button" disabled={locked} onclick={close}
					>Close</button>
			</div>
		{/if}
	</Dialog>
{/if}
