<script lang="ts">
	import arrowsClockwiseIcon from "@phosphor-icons/core/regular/arrows-clockwise.svg?raw"
	import calendarDotsIcon from "@phosphor-icons/core/regular/calendar-dots.svg?raw"
	import crownSimpleIcon from "@phosphor-icons/core/regular/crown-simple.svg?raw"
	import swapIcon from "@phosphor-icons/core/regular/swap.svg?raw"
	import coinsIcon from "@phosphor-icons/core/regular/coins.svg?raw"
	import diamondIcon from "@phosphor-icons/core/regular/diamond.svg?raw"
	import medalIcon from "@phosphor-icons/core/regular/medal.svg?raw"
	import trayIcon from "@phosphor-icons/core/regular/tray.svg?raw"
	import usersFourIcon from "@phosphor-icons/core/regular/users-four.svg?raw"
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import circleDashedIcon from "@phosphor-icons/core/regular/circle-dashed.svg?raw"
	import cookieIcon from "@phosphor-icons/core/regular/cookie.svg?raw"
	import databaseIcon from "@phosphor-icons/core/regular/database.svg?raw"
	import globeHemisphereWestIcon from "@phosphor-icons/core/regular/globe-hemisphere-west.svg?raw"
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import shieldCheckIcon from "@phosphor-icons/core/regular/shield-check.svg?raw"
	import shieldSlashIcon from "@phosphor-icons/core/regular/shield-slash.svg?raw"
	import userCircleIcon from "@phosphor-icons/core/regular/user-circle.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import warningCircleIcon from "@phosphor-icons/core/regular/warning-circle.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { AgeVerification, ModerationKind, SessionState } from "../backend/bridge"
	import { moderationLabel, resolvesInBrowser } from "./moderation.svelte"
	import type { AccountProfileSnapshot } from "../backend/bridge"
	import Timestamp from "../shared/Timestamp.svelte"
	import { relativeTime } from "../shared/timestamp"
	import { ageVerificationLabels } from "./account-model"
	import type { Account } from "./account-model"
	import AccountVoiceChat from "./AccountVoiceChat.svelte"

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
		exactNumbers = new Intl.NumberFormat("en-US"),
		// Roblox trade eligibility statuses from the trades v2 API.
		tradeEligibilityLabels: Record<string, string> = {
			IneligibleTradeSystemDisabled: "Trading is turned off",
			IneligibleMissingPremiumMembership: "Needs Premium",
			IneligibleLegalOrRegulatoryRestrictions: "Not available in this region",
			IneligibleFreeTradesLimitReached: "Free trade limit reached",
			IneligibleAgeCheckRequired: "Needs an age check",
		},
		premiumDate = $derived.by(() => {
			const subscription = snapshot?.premiumSubscription
			if (!subscription) return null
			if (subscription.renewsAtMs) {
				return { label: "Premium renews", valueMs: subscription.renewsAtMs }
			}
			if (subscription.expiresAtMs) {
				return { label: "Premium ends", valueMs: subscription.expiresAtMs }
			}
			return null
		}),
		trading = $derived.by(() => {
			const eligibility = snapshot?.tradeEligibility
			if (!eligibility) return null
			const freeTrades = eligibility.freeTrades
			return {
				label: eligibility.canTrade
					? "Can trade"
					: (tradeEligibilityLabels[eligibility.eligibility] ??
						"Cannot trade"),
				detail: freeTrades
					? `${exactNumbers.format(freeTrades.remaining)} of ${exactNumbers.format(freeTrades.limit)} free trades left${freeTrades.window ? ` this ${freeTrades.window}` : ""}`
					: "",
				isWarning: !eligibility.canTrade,
			}
		}),
		activityLabels = $derived.by(() => {
			if (
				!snapshot ||
				snapshot.unreadMessageCount === null ||
				snapshot.inboundTradeCount === null ||
				snapshot.unreadNotificationCount === null
			) {
				return null
			}
			return [
				countLabel(snapshot.unreadNotificationCount, "unread notification"),
				countLabel(snapshot.unreadMessageCount, "unread message"),
				countLabel(snapshot.inboundTradeCount, "trade request"),
			].filter((label) => label !== "")
		}),
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
						detail: account.stateReason || "Replace the cookie.",
					}
				case SessionState.StateChallenged:
					return {
						tone: "warning",
						label: "Security challenge",
						detail:
							account.stateReason ||
							"Roblox asked for extra verification.",
					}
				case SessionState.StateModerated:
					return {
						tone:
							account.moderation?.kind ===
								ModerationKind.ModerationWarned ||
							resolvesInBrowser(account.moderation)
								? "warning"
								: "danger",
						label: moderationLabel(account.moderation),
						detail: "The cookie works, but Roblox blocks this account.",
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

	function countLabel(count: number, noun: string): string {
		if (count <= 0) return ""
		return `${exactNumbers.format(count)} ${noun}${count === 1 ? "" : "s"}`
	}
</script>

<section class="flex flex-col gap-2" aria-labelledby="profile-account-heading">
	<h3 id="profile-account-heading" class="text-xs font-semibold text-fg-muted">
		Account
	</h3>
	<dl class="facts">
		{#if account.createdAtMs}
			<div>
				<dt>
					<Icon svg={calendarDotsIcon} size={16} aria-hidden="true" />Joined
				</dt>
				<dd>
					<Timestamp value={account.createdAtMs} />
					<small>{accountAge} old</small>
				</dd>
			</div>
		{/if}
		{#if country}
			<div>
				<dt>
					<Icon
						svg={globeHemisphereWestIcon}
						size={16}
						aria-hidden="true" />Country
				</dt>
				<dd>
					<span>{country}</span>
					<small class="font-mono">{snapshot?.countryCode}</small>
				</dd>
			</div>
		{/if}
		{#if age || ageVerification}
			<div>
				<dt>
					{#if snapshot?.ageVerification === AgeVerification.AgeVerifiedFaceScan || snapshot?.ageVerification === AgeVerification.AgeVerifiedID}<Icon
							svg={sealCheckIcon}
							size={16}
							aria-hidden="true" />{:else}<Icon
							svg={userCircleIcon}
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
			<div data-tone={snapshot.twoStepEnabled ? undefined : "warning"}>
				<dt>
					{#if snapshot.twoStepEnabled}<Icon
							svg={shieldCheckIcon}
							size={16}
							aria-hidden="true" />{:else}<Icon
							svg={shieldSlashIcon}
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
		{#if snapshot?.voiceChat}
			<AccountVoiceChat voiceChat={snapshot.voiceChat} label="Voice chat" />
		{/if}
		{#if snapshot?.primaryGroup}
			{@const group = snapshot.primaryGroup}
			<div>
				<dt>
					<Icon svg={usersThreeIcon} size={16} aria-hidden="true" />Primary
					group
				</dt>
				<dd>
					<span class="inline-flex min-w-0 items-center gap-1"
						>{group.name}{#if group.verified}<Icon
								svg={sealCheckIcon}
								class="shrink-0 text-fg-muted"
								size={14}
								aria-label="Verified group" />{/if}</span>
					<small
						>{group.role ? `${group.role} - ` : ""}{numbers.format(
							group.members,
						)}
						{group.members === 1 ? "member" : "members"}</small>
				</dd>
			</div>
		{/if}
		{#if snapshot?.premiumSubscription}
			{@const subscription = snapshot.premiumSubscription}
			<div class="animate-enter">
				<dt>
					<Icon
						svg={crownSimpleIcon}
						size={16}
						aria-hidden="true" />{premiumDate?.label ?? "Premium"}
				</dt>
				<dd>
					{#if premiumDate}<Timestamp
							value={premiumDate.valueMs} />{:else}<span
							>{subscription.name || "Premium"}</span
						>{/if}
					<small
						>{[
							premiumDate ? subscription.name : "",
							premiumDate ? relativeTime(premiumDate.valueMs) : "",
							subscription.robuxStipend
								? `R$ ${exactNumbers.format(subscription.robuxStipend)} a month`
								: "",
						]
							.filter(Boolean)
							.join(" - ")}</small>
				</dd>
			</div>
		{/if}
		{#if trading}
			<div
				class="animate-enter"
				data-tone={trading.isWarning ? "warning" : undefined}>
				<dt><Icon svg={swapIcon} size={16} aria-hidden="true" />Trading</dt>
				<dd>
					<span>{trading.label}</span>
					{#if trading.detail}<small>{trading.detail}</small>{/if}
				</dd>
			</div>
		{/if}
		{#if snapshot && snapshot.communityCount !== null}
			<div class="animate-enter">
				<dt>
					<Icon svg={usersFourIcon} size={16} aria-hidden="true" />Communities
				</dt>
				<dd><span>{exactNumbers.format(snapshot.communityCount)}</span></dd>
			</div>
		{/if}
		{#if snapshot?.robloxBadges?.length}
			<div class="animate-enter">
				<dt>
					<Icon svg={medalIcon} size={16} aria-hidden="true" />Roblox badges
				</dt>
				<dd><span>{snapshot.robloxBadges.join(", ")}</span></dd>
			</div>
		{/if}
		{#if snapshot && snapshot.incomingRobuxPastMonth !== null && snapshot.outgoingRobuxPastMonth !== null}
			<div class="animate-enter">
				<dt>
					<Icon svg={coinsIcon} size={16} aria-hidden="true" />Robux past
					month
				</dt>
				<dd>
					<span class="font-mono tabular-nums"
						>+{exactNumbers.format(snapshot.incomingRobuxPastMonth)} / -{exactNumbers.format(
							snapshot.outgoingRobuxPastMonth,
						)}</span>
					<small>Received / spent</small>
				</dd>
			</div>
		{/if}
		{#if snapshot && snapshot.collectibleCount !== null}
			<div class="animate-enter">
				<dt>
					<Icon svg={diamondIcon} size={16} aria-hidden="true" />Collectibles
				</dt>
				<dd>
					<span
						>{exactNumbers.format(snapshot.collectibleCount)}
						{snapshot.collectibleCount === 1 ? "item" : "items"}</span>
					{#if snapshot.collectibleCount > 0 && snapshot.collectibleRecentAveragePriceTotal !== null}<small
							class="font-mono"
							>RAP R$ {exactNumbers.format(
								snapshot.collectibleRecentAveragePriceTotal,
							)}</small
						>{/if}
				</dd>
			</div>
		{/if}
		{#if activityLabels}
			<div
				class="animate-enter"
				data-tone={activityLabels.length ? undefined : "muted"}>
				<dt><Icon svg={trayIcon} size={16} aria-hidden="true" />On Roblox</dt>
				<dd>
					<span
						>{activityLabels.length
							? activityLabels.join(", ")
							: "Nothing new"}</span>
				</dd>
			</div>
		{/if}
	</dl>
</section>

<section class="flex flex-col gap-2" aria-labelledby="profile-vault-heading">
	<h3 id="profile-vault-heading" class="text-xs font-semibold text-fg-muted">
		Vault
	</h3>
	<dl class="facts">
		<div data-tone={session.tone}>
			<dt>
				{#if session.tone === "success"}<Icon
						svg={checkCircleIcon}
						size={16}
						aria-hidden="true" />{:else if session.tone === "warning"}<Icon
						svg={warningCircleIcon}
						size={16}
						aria-hidden="true" />{:else if session.tone === "danger"}<Icon
						svg={prohibitIcon}
						size={16}
						aria-hidden="true" />{:else}<Icon
						svg={circleDashedIcon}
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
				<dt>
					<Icon svg={cookieIcon} size={16} aria-hidden="true" />Cookie expires
				</dt>
				<dd>
					<Timestamp value={account.cookieExpiresAtMs} />
					<small>{relativeTime(account.cookieExpiresAtMs)}</small>
				</dd>
			</div>
		{/if}
		{#if account.lastValidatedAtMs}
			<div>
				<dt>
					<Icon svg={checkCircleIcon} size={16} aria-hidden="true" />Validated
				</dt>
				<dd>
					<Timestamp value={account.lastValidatedAtMs} />
					<small>{relativeTime(account.lastValidatedAtMs)}</small>
				</dd>
			</div>
		{/if}
		{#if account.rotatedAtMs}
			<div>
				<dt>
					<Icon
						svg={arrowsClockwiseIcon}
						data-turn
						size={16}
						aria-hidden="true" />Cookie renewed
				</dt>
				<dd>
					<Timestamp value={account.rotatedAtMs} />
					<small>{relativeTime(account.rotatedAtMs)}</small>
				</dd>
			</div>
		{/if}
		{#if account.importedAtMs}
			<div>
				<dt><Icon svg={databaseIcon} size={16} aria-hidden="true" />Added</dt>
				<dd>
					<Timestamp value={account.importedAtMs} />
					<small>{relativeTime(account.importedAtMs)}</small>
				</dd>
			</div>
		{/if}
	</dl>
</section>
