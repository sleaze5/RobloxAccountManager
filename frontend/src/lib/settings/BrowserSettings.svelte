<script lang="ts">
	import arrowCounterClockwiseIcon from "@phosphor-icons/core/regular/arrow-counter-clockwise.svg?raw"
	import downloadSimpleIcon from "@phosphor-icons/core/regular/download-simple.svg?raw"
	import trashIcon from "@phosphor-icons/core/regular/trash.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import BrowserRuntimeActivity from "../browser/BrowserRuntimeActivity.svelte"
	import type { BrowserStore } from "../browser/browser-store.svelte"
	import { RuntimeStatus } from "../backend/bridge"
	import type { ShutdownEffects } from "../backend/bridge"
	import { formatCompactBytes } from "../shared/bytes"
	import Dialog from "../shared/Dialog.svelte"
	import ConnectionHosts from "./ConnectionHosts.svelte"
	import SectionLink from "./SectionLink.svelte"
	import SettingsStatus from "./SettingsStatus.svelte"

	let { browser, query = "" }: { browser: BrowserStore; query?: string } = $props(),
		pendingAction = $state<"redownload" | "remove" | null>(null),
		effects = $state<ShutdownEffects | null>(null)
	const runtime = $derived(browser.snapshot.runtime),
		active = $derived(
			runtime.status === RuntimeStatus.RuntimeDownloading ||
				runtime.status === RuntimeStatus.RuntimeInstalling,
		),
		terms = $derived(query.trim().toLowerCase().split(/\s+/).filter(Boolean)),
		visible = $derived(
			terms.every((term) =>
				`integrations browser chrome testing runtime version download redownload remove disk usage installed manage hosts connect storage.googleapis.com`.includes(
					term,
				),
			),
		)

	$effect(() => {
		if (visible) return browser.trackRuntimeView()
	})

	function runtimeSummary(): string {
		if (runtime.status === RuntimeStatus.RuntimeDamaged) {
			return runtime.installedVersion
				? `Version ${runtime.installedVersion} is invalid.`
				: "The installed browser is invalid."
		}
		if (runtime.status === RuntimeStatus.RuntimeMissing) {
			return `Version ${runtime.requiredVersion} is not installed.`
		}
		const usage =
			runtime.installedBytes > 0
				? ` - ${formatCompactBytes(runtime.installedBytes)}`
				: ""
		return `Version ${runtime.requiredVersion}${usage}`
	}

	const status = $derived.by(
		(): {
			value: string
			tone: "neutral" | "success" | "danger"
		} => {
			if (runtime.status === RuntimeStatus.RuntimeDownloading) {
				return { value: "Downloading", tone: "neutral" }
			}
			if (runtime.status === RuntimeStatus.RuntimeInstalling) {
				return { value: "Installing", tone: "neutral" }
			}
			if (runtime.status === RuntimeStatus.RuntimeReady) {
				return { value: "Ready", tone: "success" }
			}
			if (runtime.status === RuntimeStatus.RuntimeDamaged) {
				return { value: "Invalid", tone: "danger" }
			}
			return { value: "Not installed", tone: "neutral" }
		},
	)

	async function requestAction(action: "redownload" | "remove"): Promise<void> {
		const current = await browser.effects()
		if (
			current.browserSessions ||
			current.browserDownloads ||
			current.accountCandidates ||
			current.accountChecks
		) {
			effects = current
			pendingAction = action
			return
		}
		await runAction(action)
	}

	async function runAction(action: "redownload" | "remove"): Promise<void> {
		pendingAction = null
		effects = null
		if (action === "redownload") await browser.download(true)
		else await browser.removeRuntime()
	}
</script>

{#if visible}
	<section class="settings-section" aria-labelledby="browser-runtime-title">
		<h3 class="flex items-center justify-between gap-2">
			<span id="browser-runtime-title">Browser</span>
			<SectionLink
				label="Open Chrome for Testing website"
				url="https://googlechromelabs.github.io/chrome-for-testing/" />
		</h3>
		{#if browser.error || runtime.error}<div class="notice-error" role="alert">
				{browser.error || runtime.error}
			</div>{/if}
		<div class="settings-rows">
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong>Chrome for Testing</strong><span>{runtimeSummary()}</span>
					<SettingsStatus value={status.value} tone={status.tone} />
				</div>
			</div>
			{#if active}
				<div class="settings-row">
					<BrowserRuntimeActivity {runtime} />
					{#if runtime.canCancel}<button
							class="btn"
							type="button"
							onclick={() => void browser.cancelDownload()}
							>Cancel download</button
						>{/if}
				</div>
			{:else}
				<div class="settings-row">
					<div class="settings-row-copy">
						<strong>Manage browser</strong><span
							>The managed browser stays in the data folder.</span>
					</div>
					<div class="flex items-center gap-2">
						{#if runtime.status === RuntimeStatus.RuntimeMissing}<button
								class="btn"
								type="button"
								onclick={() => void browser.download()}
								><Icon
									svg={downloadSimpleIcon}
									size={16}
									aria-hidden="true" />Download</button
							>{:else}<button
								class="btn"
								type="button"
								onclick={() => void requestAction("redownload")}
								><Icon
									svg={arrowCounterClockwiseIcon}
									size={16}
									aria-hidden="true" />Redownload</button
							><button
								class="btn"
								type="button"
								onclick={() => void requestAction("remove")}
								><Icon
									svg={trashIcon}
									size={16}
									aria-hidden="true" />Remove</button
							>{/if}
					</div>
				</div>
			{/if}
			<ConnectionHosts hosts={["storage.googleapis.com"]} />
		</div>
	</section>
{/if}

{#if pendingAction && effects}
	<Dialog
		class="max-w-sm text-left"
		role="alertdialog"
		aria-labelledby="runtime-confirm-title">
		<div class="dialog-heading">
			<div>
				<h2 id="runtime-confirm-title">
					{pendingAction === "remove"
						? "Remove browser?"
						: "Redownload browser?"}
				</h2>
				<p>This will stop browser work that is currently active.</p>
			</div>
			<button
				type="button"
				aria-label="Close browser confirmation"
				onclick={() => {
					pendingAction = null
					effects = null
				}}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>
		<ul class="mt-3 flex list-disc flex-col gap-1 pl-5 text-xs text-warning">
			{#if effects.browserSessions}<li>
					Close {effects.browserSessions} active browser {effects.browserSessions ===
					1
						? "session"
						: "sessions"}.
				</li>
				<li>Browser downloads may stop.</li>{/if}
			{#if effects.browserDownloads}<li>
					Stop the browser runtime download.
				</li>{/if}
			{#if effects.accountCandidates}<li>
					Discard {effects.accountCandidates} Add-account {effects.accountCandidates ===
					1
						? "candidate"
						: "candidates"}.
				</li>{/if}
			{#if effects.accountChecks}<li>Cancel account validations.</li>{/if}
		</ul>
		<div class="dialog-actions">
			<button
				class="btn"
				type="button"
				onclick={() => {
					pendingAction = null
					effects = null
				}}>Cancel</button
			><button
				class={pendingAction === "remove" ? "btn-danger" : "btn-primary"}
				type="button"
				onclick={() => void runAction(pendingAction!)}
				>{pendingAction === "remove"
					? "Remove browser"
					: "Redownload browser"}</button>
		</div>
	</Dialog>
{/if}
