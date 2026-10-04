<script lang="ts">
	import DownloadSimpleIcon from "phosphor-svelte/lib/DownloadSimpleIcon"
	import ArrowCounterClockwiseIcon from "phosphor-svelte/lib/ArrowCounterClockwiseIcon"
	import TrashIcon from "phosphor-svelte/lib/TrashIcon"
	import BrowserRuntimeActivity from "../browser/BrowserRuntimeActivity.svelte"
	import type { BrowserStore } from "../browser/browser-store.svelte"
	import { RuntimeStatus } from "../backend/bridge"
	import type { ShutdownEffects } from "../backend/bridge"
	import { formatCompactBytes } from "../shared/bytes"
	import SettingsStatus from "./SettingsStatus.svelte"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"

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
				`integrations browser chrome testing runtime version download redownload remove disk usage installed manage`.includes(
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
		<h3 id="browser-runtime-title">Browser</h3>
		{#if browser.error || runtime.error}<div
				class="settings-inline-error"
				role="alert">
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
							class="settings-row-action"
							type="button"
							onclick={() => void browser.cancelDownload()}
							>Cancel download</button
						>{/if}
				</div>
			{:else}
				<div class="settings-row">
					<div class="settings-row-copy">
						<strong>Manage browser</strong><span
							>The managed browser stays in this portable application
							directory.</span>
					</div>
					<div class="settings-row-actions">
						{#if runtime.status === RuntimeStatus.RuntimeMissing}<button
								class="settings-row-action"
								type="button"
								onclick={() => void browser.download()}
								><DownloadSimpleIcon
									size={16}
									aria-hidden="true" />Download</button
							>{:else}<button
								class="settings-row-action"
								type="button"
								onclick={() => void requestAction("redownload")}
								><ArrowCounterClockwiseIcon
									size={16}
									aria-hidden="true" />Redownload</button
							><button
								class="settings-row-action danger-action"
								type="button"
								onclick={() => void requestAction("remove")}
								><TrashIcon
									size={16}
									aria-hidden="true" />Remove</button
							>{/if}
					</div>
				</div>
			{/if}
		</div>
	</section>
{/if}

{#if pendingAction && effects}
	<div class="modal-backdrop" in:modalBackdropIn out:modalBackdropOut>
		<section
			class="modal-card confirm-card"
			use:modalFocus
			in:modalCardIn
			out:modalCardOut
			aria-labelledby="runtime-confirm-title">
			<div class="modal-heading">
				<div>
					<h2 id="runtime-confirm-title">
						{pendingAction === "remove"
							? "Remove browser?"
							: "Redownload browser?"}
					</h2>
					<p>This will stop browser work that is currently active.</p>
				</div>
			</div>
			<ul class="confirmation-effects">
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
			<div class="modal-actions">
				<button
					type="button"
					onclick={() => {
						pendingAction = null
						effects = null
					}}>Cancel</button
				><button
					class="danger-button"
					type="button"
					onclick={() => void runAction(pendingAction!)}
					>{pendingAction === "remove" ? "Remove" : "Redownload"}</button>
			</div>
		</section>
	</div>
{/if}
