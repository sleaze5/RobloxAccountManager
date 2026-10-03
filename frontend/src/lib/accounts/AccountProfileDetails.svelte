<script lang="ts">
	import BadgeCheck from "@lucide/svelte/icons/badge-check"
	import CalendarDays from "@lucide/svelte/icons/calendar-days"
	import CircleAlert from "@lucide/svelte/icons/circle-alert"
	import CircleCheck from "@lucide/svelte/icons/circle-check"
	import CircleDashed from "@lucide/svelte/icons/circle-dashed"
	import Cookie from "@lucide/svelte/icons/cookie"
	import DatabaseZap from "@lucide/svelte/icons/database-zap"
	import Globe2 from "@lucide/svelte/icons/earth"
	import RefreshCw from "@lucide/svelte/icons/refresh-cw"
	import ShieldCheck from "@lucide/svelte/icons/shield-check"
	import ShieldOff from "@lucide/svelte/icons/shield-off"
	import UserRound from "@lucide/svelte/icons/user-round"
	import UsersRound from "@lucide/svelte/icons/users-round"
	import moment from "moment"
	import { SessionState } from "../backend/bridge"
	import type { AccountProfileSnapshot } from "../backend/bridge"
	import Timestamp from "../shared/Timestamp.svelte"
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
		ageGroups: Record<string, string> = {
			AgeUnder9: "Under 9",
			Age9To12: "9–12",
			AgeUnder13To17: "13–17",
			Age18OrOver: "18+",
		},
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
			const checked = ageGroups[snapshot.ageGroup]
			if (checked) return checked
			if (snapshot.ageBracket === 0) return "13+"
			if (snapshot.ageBracket === 1) return "Under 13"
			return ""
		}),
		ageVerification = $derived(
			snapshot?.ageVerified === true
				? "Age verified"
				: snapshot?.ageVerified === false
					? "Not age verified"
					: "",
		),
		methods = $derived(
			snapshot?.twoStepMethods
				?.map((method) => verificationMethods[method] ?? method)
				.join(", ") ?? "",
		),
		accountAge = $derived(
			account.createdAtMs ? moment(account.createdAtMs).fromNow(true) : "",
		),
		cookieExpiry = $derived(
			account.cookieExpiresAtMs
				? moment(account.cookieExpiresAtMs).fromNow()
				: "",
		),
		session = $derived.by(() => {
			switch (account.state) {
				case SessionState.StateActive:
					return { tone: "success", label: "Session active", detail: "" }
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
						label: "Session disabled",
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
	<ul class="profile-facts">
		{#if account.createdAtMs}
			<li>
				<CalendarDays size={14} aria-hidden="true" />
				<span>Joined <Timestamp value={account.createdAtMs} /></span>
				<small>{accountAge} old</small>
			</li>
		{/if}
		{#if country}
			<li>
				<Globe2 size={14} aria-hidden="true" />
				<span>{country}</span>
				<small class="mono">{snapshot?.countryCode}</small>
			</li>
		{/if}
		{#if age || ageVerification}
			<li>
				{#if snapshot?.ageVerified}<BadgeCheck
						size={14}
						aria-hidden="true" />{:else}<UserRound
						size={14}
						aria-hidden="true" />{/if}
				<span>{age ? `Age ${age}` : "Age unavailable"}</span>
				{#if ageVerification}<small>{ageVerification}</small>{/if}
			</li>
		{/if}
		{#if snapshot && snapshot.twoStepEnabled !== null}
			<li class:warning={!snapshot.twoStepEnabled}>
				{#if snapshot.twoStepEnabled}<ShieldCheck
						size={14}
						aria-hidden="true" />{:else}<ShieldOff
						size={14}
						aria-hidden="true" />{/if}
				<span
					>{snapshot.twoStepEnabled
						? "2-step verification on"
						: "2-step verification off"}</span>
				{#if methods}<small>{methods}</small>{/if}
			</li>
		{/if}
		{#if snapshot?.primaryGroup}
			{@const group = snapshot.primaryGroup}
			<li>
				<UsersRound size={14} aria-hidden="true" />
				<span class="profile-fact-name"
					>{group.name}{#if group.verified}<BadgeCheck
							size={12}
							aria-label="Verified group" />{/if}</span>
				<small
					>{group.role ? `${group.role} · ` : ""}{numbers.format(
						group.members,
					)}
					{group.members === 1 ? "member" : "members"}</small>
			</li>
		{/if}
	</ul>
</section>

<section class="profile-section" aria-labelledby="profile-vault-heading">
	<h3 id="profile-vault-heading" class="profile-section-label">Vault</h3>
	<ul class="profile-facts">
		<li class={session.tone}>
			{#if session.tone === "success"}<CircleCheck
					size={14}
					aria-hidden="true" />{:else if session.tone === "warning"}<CircleAlert
					size={14}
					aria-hidden="true" />{:else}<CircleDashed
					size={14}
					aria-hidden="true" />{/if}
			<span>{session.label}</span>
			{#if session.detail}<small>{session.detail}</small>{/if}
		</li>
		{#if account.cookieExpiresAtMs}
			<li>
				<Cookie size={14} aria-hidden="true" />
				<span
					>Cookie expires <Timestamp
						value={account.cookieExpiresAtMs} /></span>
				<small>{cookieExpiry}</small>
			</li>
		{/if}
		{#if account.lastValidatedAtMs}
			<li>
				<CircleCheck size={14} aria-hidden="true" />
				<span>Validated <Timestamp value={account.lastValidatedAtMs} /></span>
			</li>
		{/if}
		{#if account.rotatedAtMs}
			<li>
				<RefreshCw size={14} aria-hidden="true" />
				<span>Cookie renewed <Timestamp value={account.rotatedAtMs} /></span>
			</li>
		{/if}
		{#if account.importedAtMs}
			<li>
				<DatabaseZap size={14} aria-hidden="true" />
				<span>Added <Timestamp value={account.importedAtMs} /></span>
			</li>
		{/if}
	</ul>
</section>
