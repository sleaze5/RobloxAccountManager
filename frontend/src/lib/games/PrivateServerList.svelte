<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import lockKeyIcon from "@phosphor-icons/core/regular/lock-key.svg?raw"
	import plusIcon from "@phosphor-icons/core/regular/plus.svg?raw"
	import userPlusIcon from "@phosphor-icons/core/regular/user-plus.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { untrack } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import Select from "../shared/Select.svelte"
	import ServerActionsButton from "./ServerActionsButton.svelte"
	import ServerPagerNav from "./ServerPagerNav.svelte"
	import type { PrivateServers, SavedServers } from "./game-servers-state.svelte"

	let {
		servers,
		saved,
		accounts,
		activeKey,
		menuID,
		onMenu,
		onAccount,
		onAddLink,
	}: {
		servers: PrivateServers
		saved: SavedServers
		accounts: AccountStore
		activeKey: string | undefined
		menuID: string
		onMenu: (key: string, trigger: HTMLElement, toggle: boolean) => void
		onAccount: (accountId: number) => void
		onAddLink: () => void
	} = $props()

	const numbers = new Intl.NumberFormat()
	let list = $state<HTMLElement | undefined>(undefined)
	const account = $derived(accounts.selectedAccount),
		shown = $derived(servers.page?.servers ?? []),
		savedShown = $derived(saved.servers ?? []),
		accountOptions = $derived(
			accounts.accounts.map((item) => ({
				value: item.id,
				label: item.displayName,
				description: `@${item.username}`,
				image: item.avatarUrl,
			})),
		)

	$effect(() => {
		void servers.page
		untrack(() => list?.scrollTo({ top: 0 }))
	})
</script>

<div class="flex min-h-0 flex-1 flex-col">
	<div
		class="flex flex-wrap items-end gap-x-3 gap-y-2 border-b border-line-subtle p-3"
		role="group"
		aria-label="Server filters">
		<div class="flex flex-col gap-1 text-2xs text-fg-muted">
			<span>Account</span>
			<Select
				className="w-56"
				label="Account used to list private servers"
				value={account.id}
				options={accountOptions}
				disabled={accountOptions.length === 0 || servers.loading}
				onChange={onAccount} />
		</div>
		<ServerPagerNav pager={servers} />
	</div>

	<div class="min-h-0 flex-1 overflow-auto" bind:this={list}>
		{#if account.id === 0}
			<div class="empty-state" role="status">
				<Icon svg={userPlusIcon} size={28} aria-hidden="true" />
				<h2>No accounts yet</h2>
				<p>Add an account on the Accounts page to see its private servers.</p>
			</div>
		{:else if servers.error}
			<div class="empty-state" role="alert">
				<Icon svg={warningIcon} size={28} aria-hidden="true" />
				<h2>Private servers are unavailable</h2>
				<p>{servers.error}</p>
				<button class="button" type="button" onclick={() => void servers.load()}
					>Try again</button>
			</div>
		{:else if !servers.page}
			<div class="empty-state" role="status">
				<Icon
					svg={circleNotchIcon}
					class="spinner"
					size={28}
					aria-hidden="true" />
				<h2>Loading private servers…</h2>
			</div>
		{:else if !shown.length}
			<div class="empty-state" role="status">
				<Icon svg={lockKeyIcon} size={28} aria-hidden="true" />
				{#if servers.pageNumber > 1}
					<h2>No more private servers</h2>
					<p>The servers on this page changed. Go back to the first page.</p>
					<button class="button" type="button" onclick={() => servers.first()}
						>First page</button>
				{:else}
					<h2>No private servers</h2>
					<p>
						@{account.username} does not own or have access to a private server
						in this place.
					</p>
				{/if}
			</div>
		{:else}
			<p
				class="flex gap-1 px-3 pt-2 font-mono text-2xs text-fg-faint"
				role="status">
				<span>Page {servers.pageNumber}</span>
				<span aria-hidden="true">-</span>
				<span
					>{numbers.format(shown.length)}
					{shown.length === 1 ? "server" : "servers"}</span>
				<span aria-hidden="true">-</span>
				<span>Owned by or shared with @{account.username}</span>
			</p>
			<div
				class="server-table flex-none overflow-visible"
				data-variant="private"
				data-loading={servers.loading || undefined}>
				<div class="server-columns" aria-hidden="true">
					<span>Name</span>
					<span>Owner</span>
					<span class="justify-self-end text-right">Players</span>
					<span>Status</span>
				</div>
				<ol aria-label="Private servers" aria-busy={servers.loading}>
					{#each shown as server (server.serverId)}
						{@const key = String(server.serverId)}
						{@const owner = server.ownerDisplayName || server.ownerName}
						<li
							class="server-row"
							data-open={activeKey === key || undefined}
							oncontextmenu={(event) => {
								const trigger =
									event.currentTarget.querySelector<HTMLElement>(
										"[data-server-actions]",
									)
								if (!trigger) return
								event.preventDefault()
								onMenu(key, trigger, true)
							}}>
							<span class="flex min-w-0 items-center gap-1.5">
								<span
									class="truncate text-xs font-medium text-fg"
									data-tooltip={server.name || undefined}
									>{server.name || "Unnamed server"}</span>
								{#if server.owned}<span class="shrink-0 server-tag"
										>Yours</span
									>{/if}
							</span>
							{#if owner}
								<span class="flex min-w-0 items-baseline gap-1 text-xs">
									<span class="truncate text-fg">{owner}</span>
									{#if server.ownerName && server.ownerName !== owner}
										<span class="truncate text-2xs text-fg-faint"
											>@{server.ownerName}</span>
									{/if}
								</span>
							{:else}
								<span class="text-fg-faint" aria-label="Owner unknown"
									>-</span>
							{/if}
							<span
								class="justify-self-end font-mono text-2xs whitespace-nowrap text-fg-muted"
								aria-label={`${server.playing} of ${server.maxPlayers} players`}
								><strong class="font-semibold text-fg"
									>{server.playing}</strong
								>/{server.maxPlayers}</span>
							<span
								class={[
									"flex items-center gap-1.5 text-xs",
									server.running ? "text-fg" : "text-fg-muted",
								]}>
								<span
									class={[
										"size-1.5 shrink-0 rounded-full",
										server.running
											? "bg-success"
											: "bg-line-strong",
									]}
									aria-hidden="true"></span>
								{server.running ? "Running" : "Idle"}
							</span>
							<ServerActionsButton
								open={activeKey === key}
								{menuID}
								onMenu={(trigger, toggle) =>
									onMenu(key, trigger, toggle)} />
						</li>
					{/each}
				</ol>
			</div>
		{/if}

		<section
			class="-mt-px border-t border-line-subtle"
			aria-labelledby={`${menuID}-saved`}>
			<div class="flex items-center gap-3 px-3 pt-3 pb-2">
				<div class="flex min-w-0 flex-1 flex-col gap-0.5">
					<h3
						id={`${menuID}-saved`}
						class="text-xs font-semibold text-fg-muted">
						Saved links
						{#if savedShown.length}<span class="font-normal"
								>({savedShown.length})</span
							>{/if}
					</h3>
					<p class="text-2xs text-fg-faint">
						Place IDs and link codes stored in the vault for this game.
					</p>
				</div>
				<button class="button" type="button" onclick={onAddLink}>
					<Icon svg={plusIcon} size={14} aria-hidden="true" />
					Add link
				</button>
			</div>
			{#if saved.error}
				<div
					class="mx-3 mb-3 flex items-center gap-3 notice-error"
					role="alert">
					<span class="flex-1">{saved.error}</span>
					<button
						class="button"
						type="button"
						onclick={() => void saved.load()}>Try again</button>
				</div>
			{:else if !saved.servers}
				<p
					class="flex items-center gap-2 px-3 pb-3 text-xs text-fg-muted"
					role="status">
					<Icon
						svg={circleNotchIcon}
						class="spinner"
						size={14}
						aria-hidden="true" />
					Loading saved links…
				</p>
			{:else if !savedShown.length}
				<p class="px-3 pb-3 text-xs text-fg-faint">
					No saved links. Add a private server link to keep it here.
				</p>
			{:else}
				<div
					class="server-table flex-none overflow-visible"
					data-variant="listed">
					<div class="server-columns" aria-hidden="true">
						<span>Name</span>
						<span>Code</span>
					</div>
					<ol aria-label="Saved private server links">
						{#each savedShown as server (server.id)}
							{@const key = `saved-${server.id}`}
							<li
								class="server-row animate-enter"
								data-open={activeKey === key || undefined}
								oncontextmenu={(event) => {
									const trigger =
										event.currentTarget.querySelector<HTMLElement>(
											"[data-server-actions]",
										)
									if (!trigger) return
									event.preventDefault()
									onMenu(key, trigger, true)
								}}>
								<span
									class="truncate text-xs font-medium text-fg"
									data-tooltip={server.name || undefined}
									>{server.name || "Saved server"}</span>
								<span
									class="truncate font-mono text-2xs text-fg-faint select-text"
									data-tooltip={server.linkCode
										? `Link code ${server.linkCode}`
										: `Access code ${server.accessCode}`}
									>{server.linkCode || server.accessCode}</span>
								<ServerActionsButton
									open={activeKey === key}
									{menuID}
									onMenu={(trigger, toggle) =>
										onMenu(key, trigger, toggle)} />
							</li>
						{/each}
					</ol>
				</div>
			{/if}
		</section>
	</div>
</div>
