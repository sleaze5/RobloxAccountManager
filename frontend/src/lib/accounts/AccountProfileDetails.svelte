<script lang="ts">
	import SealCheckIcon from "phosphor-svelte/lib/SealCheckIcon"
	import CalendarDotsIcon from "phosphor-svelte/lib/CalendarDotsIcon"
	import WarningCircleIcon from "phosphor-svelte/lib/WarningCircleIcon"
	import CheckCircleIcon from "phosphor-svelte/lib/CheckCircleIcon"
	import CircleDashedIcon from "phosphor-svelte/lib/CircleDashedIcon"
	import CookieIcon from "phosphor-svelte/lib/CookieIcon"
	import DatabaseIcon from "phosphor-svelte/lib/DatabaseIcon"
	import GlobeHemisphereWestIcon from "phosphor-svelte/lib/GlobeHemisphereWestIcon"
	import ArrowsClockwiseIcon from "phosphor-svelte/lib/ArrowsClockwiseIcon"
	import ShieldCheckIcon from "phosphor-svelte/lib/ShieldCheckIcon"
	import ShieldSlashIcon from "phosphor-svelte/lib/ShieldSlashIcon"
	import UserCircleIcon from "phosphor-svelte/lib/UserCircleIcon"
	import UsersThreeIcon from "phosphor-svelte/lib/UsersThreeIcon"
	import { AgeVerification, SessionState } from "../backend/bridge"
	import type { AccountProfileSnapshot } from "../backend/bridge"
	import Timestamp from "../shared/Timestamp.svelte"
	import { relativeTime } from "../shared/timestamp"
	import { ageVerificationLabels } from "./account-model"
	import type { Account } from "./account-model"

	let {
		account,
		snapshot,
	}: { account: Account; snapshot: AccountProfileSnapshot | null } = $props()

	const regions = new Intl.DisplayNames(undefined, { type: "region" }),
		numbers = new Intl.NumberFormat("en-US", {
			notation: "compact",
			maximumFractionDigits: 1,
		}),
		verificationMethods: Record<string, string> = {
			email: "Email",
			authenticator: "Authenticator app",
			securityKey: "Security key",
			sms: "SMS",
		}

	const country = $derived(
			snapshot?.countryCode
				? (regions.of(snapshot.countryCode) ?? snapshot.countryCode)
				: "",
		),
		age = $derived.by(() => {
			if (!snapshot) return ""
			if (snapshot.ageGroup) return snapshot.ageGroup
			if (snapshot.ageBracket === 0) return "13+"
			if (snapshot.ageBracket === 1) return "Under 13"
			return ""
		}),
		ageVerification = $derived(
			snapshot ? (ageVerificationLabels[snapshot.ageVerification] ?? "") : "",
		),
		methods = $derived(
			snapshot?.twoStepMethods
				?.map((method) => verificationMethods[method] ?? method)
				.join(", ") ?? "",
		),
		accountAge = $derived(
			account.createdAtMs
				? relativeTime(account.createdAtMs, Date.now(), false)
				: "",
		),
		session = $derived.by(() => {
			switch (account.state) {
				case SessionState.StateActive:
					return { tone: "success", label: "Active", detail: "" }
				case SessionState.StateReauthRequired:
					return {
						tone: "warning",
						label: "Sign-in required",
						detail: account.stateReason || "Replace or renew the cookie.",
					}
				case SessionState.StateChallenged:
					return {
						tone: "warning",
						label: "Security challenge",
						detail:
							account.stateReason ||
							"Roblox asked for extra verification.",
					}
				case SessionState.StateDisabled:
					return {
						tone: "muted",
						label: "Disabled",
						detail: account.stateReason,
					}
				default:
					return {
						tone: "muted",
						label: "Not validated yet",
						detail: "Refresh to check the session.",
					}
			}
		})
</script>

<section class="profile-section" aria-labelledby="profile-account-heading">
	<h3 id="profile-account-heading" class="profile-section-label">Account</h3>
	<dl class="profile-facts">
		{#if account.createdAtMs}
			<div>
				<dt><CalendarDotsIcon size={16} aria-hidden="true" />Joined</dt>
				<dd>
					<Timestamp value={account.createdAtMs} />
					<small>{accountAge} old</small>
				</dd>
			</div>
		{/if}
		{#if country}
			<div>
				<dt><GlobeHemisphereWestIcon size={16} aria-hidden="true" />Country</dt>
				<dd>
					<span>{country}</span>
					<small class="mono">{snapshot?.countryCode}</small>
				</dd>
			</div>
		{/if}
		{#if age || ageVerification}
			<div>
				<dt>
					{#if snapshot?.ageVerification === AgeVerification.AgeVerifiedFaceScan || snapshot?.ageVerification === AgeVerification.AgeVerifiedID}<SealCheckIcon
							size={16}
							aria-hidden="true" />{:else}<UserCircleIcon
							size={16}
							aria-hidden="true" />{/if}
					Age group
				</dt>
				<dd>
					<span>{age || "Unavailable"}</span>
					{#if ageVerification}<small>{ageVerification}</small>{/if}
				</dd>
			</div>
		{/if}
		{#if snapshot && snapshot.twoStepEnabled !== null}
			<div class:warning={!snapshot.twoStepEnabled}>
				<dt>
					{#if snapshot.twoStepEnabled}<ShieldCheckIcon
							size={16}
							aria-hidden="true" />{:else}<ShieldSlashIcon
							size={16}
							aria-hidden="true" />{/if}
					2-step verification
				</dt>
				<dd>
					<span>{snapshot.twoStepEnabled ? "On" : "Off"}</span>
					{#if methods}<small>{methods}</small>{/if}
				</dd>
			</div>
		{/if}
		{#if snapshot?.primaryGroup}
			{@const group = snapshot.primaryGroup}
			<div>
				<dt><UsersThreeIcon size={16} aria-hidden="true" />Primary group</dt>
				<dd>
					<span class="profile-fact-name"
						>{group.name}{#if group.verified}<SealCheckIcon
								size={14}
								aria-label="Verified group" />{/if}</span>
					<small
						>{group.role ? `${group.role} · ` : ""}{numbers.format(
							group.members,
						)}
						{group.members === 1 ? "member" : "members"}</small>
				</dd>
			</div>
		{/if}
	</dl>
</section>

<section class="profile-section" aria-labelledby="profile-vault-heading">
	<h3 id="profile-vault-heading" class="profile-section-label">Vault</h3>
	<dl class="profile-facts">
		<div class={session.tone}>
			<dt>
				{#if session.tone === "success"}<CheckCircleIcon
						size={16}
						aria-hidden="true" />{:else if session.tone === "warning"}<WarningCircleIcon
						size={16}
						aria-hidden="true" />{:else}<CircleDashedIcon
						size={16}
						aria-hidden="true" />{/if}
				Session
			</dt>
			<dd>
				<span>{session.label}</span>
				{#if session.detail}<small>{session.detail}</small>{/if}
			</dd>
		</div>
		{#if account.cookieExpiresAtMs}
			<div>
				<dt><CookieIcon size={16} aria-hidden="true" />Cookie expires</dt>
				<dd>
					<Timestamp value={account.cookieExpiresAtMs} />
					<small>{relativeTime(account.cookieExpiresAtMs)}</small>
				</dd>
			</div>
		{/if}
		{#if account.lastValidatedAtMs}
			<div>
				<dt><CheckCircleIcon size={16} aria-hidden="true" />Validated</dt>
				<dd>
					<Timestamp value={account.lastValidatedAtMs} />
					<small>{relativeTime(account.lastValidatedAtMs)}</small>
				</dd>
			</div>
		{/if}
		{#if account.rotatedAtMs}
			<div>
				<dt>
					<ArrowsClockwiseIcon size={16} aria-hidden="true" />Cookie renewed
				</dt>
				<dd>
					<Timestamp value={account.rotatedAtMs} />
					<small>{relativeTime(account.rotatedAtMs)}</small>
				</dd>
			</div>
		{/if}
		{#if account.importedAtMs}
			<div>
				<dt><DatabaseIcon size={16} aria-hidden="true" />Added</dt>
				<dd>
					<Timestamp value={account.importedAtMs} />
					<small>{relativeTime(account.importedAtMs)}</small>
				</dd>
			</div>
		{/if}
	</dl>
</section>
