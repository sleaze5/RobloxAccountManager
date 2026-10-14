<script lang="ts" module>
	export const accountInfoFields = [
		{ id: "email", search: "email address contact verified change edit" },
		{ id: "phone", search: "phone number contact verified" },
		{ id: "displayName", search: "display name change edit" },
		{ id: "username", search: "username name" },
		{ id: "password", search: "password change save edit validate vault security" },
		{ id: "pin", search: "account pin parental lock unlock security" },
		{ id: "gender", search: "gender" },
		{ id: "ageGroup", search: "age group checked" },
		{ id: "birthday", search: "birthday birthdate date of birth verified" },
		{ id: "firstAccount", search: "first account original alt" },
		{ id: "language", search: "language locale" },
		{ id: "translations", search: "automatic translations experience" },
		{ id: "location", search: "account location country region" },
		{
			id: "voiceChat",
			search: "voice chat microphone status suspended suspension banned",
		},
		{
			id: "voiceOptIn",
			search: "voice chat opted in enrollment microphone setting",
		},
		{
			id: "voiceVerification",
			search: "voice chat verification verified age check",
		},
	] as const

	export type AccountInfoFieldId = (typeof accountInfoFields)[number]["id"]
</script>

<script lang="ts">
	import AtIcon from "phosphor-svelte/lib/AtIcon"
	import CakeIcon from "phosphor-svelte/lib/CakeIcon"
	import CalendarIcon from "phosphor-svelte/lib/CalendarIcon"
	import FlagIcon from "phosphor-svelte/lib/FlagIcon"
	import TranslateIcon from "phosphor-svelte/lib/TranslateIcon"
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import EnvelopeIcon from "phosphor-svelte/lib/EnvelopeIcon"
	import IdentificationCardIcon from "phosphor-svelte/lib/IdentificationCardIcon"
	import KeyIcon from "phosphor-svelte/lib/KeyIcon"
	import LockKeyIcon from "phosphor-svelte/lib/LockKeyIcon"
	import PencilSimpleIcon from "phosphor-svelte/lib/PencilSimpleIcon"
	import MapPinIcon from "phosphor-svelte/lib/MapPinIcon"
	import PhoneIcon from "phosphor-svelte/lib/PhoneIcon"
	import UserCircleIcon from "phosphor-svelte/lib/UserCircleIcon"
	import GlobeHemisphereWestIcon from "phosphor-svelte/lib/GlobeHemisphereWestIcon"
	import HourglassIcon from "phosphor-svelte/lib/HourglassIcon"
	import ToggleRightIcon from "phosphor-svelte/lib/ToggleRightIcon"
	import UserCheckIcon from "phosphor-svelte/lib/UserCheckIcon"
	import Timestamp from "../shared/Timestamp.svelte"
	import SensitiveValue from "../shared/SensitiveValue.svelte"
	import { AgeVerification, PinState, VoiceChatState } from "../backend/bridge"
	import type { AccountEditKind } from "../dialogs/AccountEditDialog.svelte"
	import type { AccountInfoState } from "./account-info-state.svelte"
	import { ageVerificationLabels } from "./account-model"
	import type { Account } from "./account-model"
	import AccountPasswordMenu from "./AccountPasswordMenu.svelte"
	import AccountVoiceChat from "./AccountVoiceChat.svelte"
	import SavedPasswordValue from "./SavedPasswordValue.svelte"

	let {
		info,
		account,
		visible,
		onEdit,
		onSavedPassword,
	}: {
		info: AccountInfoState
		account: Account
		visible: ReadonlySet<AccountInfoFieldId>
		onEdit: (kind: AccountEditKind) => void
		onSavedPassword: () => void
	} = $props()

	let passwordResets = $state(0)

	const firstAccountHint =
			"Whether Roblox thinks this is the owner's first account. It may not be accurate.",
		genders: Record<string, string> = { male: "Male", female: "Female" },
		pinLabels: Partial<Record<PinState, string>> = {
			[PinState.PinOff]: "Off",
			[PinState.PinLocked]: "Locked",
			[PinState.PinUnlocked]: "Unlocked",
		},
		birthdayFormat = new Intl.DateTimeFormat("en-US", {
			month: "long",
			day: "2-digit",
			year: "numeric",
		})

	const snapshot = $derived(info.snapshot),
		birthday = $derived(formatBirthday(snapshot?.birthdate ?? ""))

	function formatBirthday(birthdate: string): string {
		const [year = 0, month = 0, day = 0] = birthdate.split("-").map(Number),
			date = new Date(year, month - 1, day)
		if (!year || date.getMonth() !== month - 1 || date.getDate() !== day) {
			return ""
		}
		const today = new Date(),
			birthdayPassed =
				today.getMonth() > date.getMonth() ||
				(today.getMonth() === date.getMonth() &&
					today.getDate() >= date.getDate()),
			age = today.getFullYear() - year - (birthdayPassed ? 0 : 1)
		return `${birthdayFormat.format(date)} (${age}y)`
	}

	function missing(source: string): boolean {
		return snapshot?.unavailable?.includes(source) ?? false
	}

	function any(...ids: AccountInfoFieldId[]): boolean {
		return ids.some((id) => visible.has(id))
	}
</script>

{#snippet edit(kind: AccountEditKind, label: string)}
	<button
		class="icon-action profile-fact-action"
		type="button"
		aria-label={label}
		data-tooltip={label}
		onclick={() => onEdit(kind)}>
		<PencilSimpleIcon size={14} aria-hidden="true" />
	</button>
{/snippet}

{#snippet passwordFact()}
	<div>
		<dt><KeyIcon size={16} aria-hidden="true" />Password</dt>
		<dd>
			{#if account.hasSavedPassword}
				{#key `${account.id}:${passwordResets}`}
					<SavedPasswordValue accountId={account.id} />
				{/key}
			{:else}
				<span class="account-info-missing">Not saved</span>
			{/if}
			<AccountPasswordMenu
				hasSavedPassword={account.hasSavedPassword}
				onEditSaved={() => {
					passwordResets++
					onSavedPassword()
				}}
				onChange={() => {
					passwordResets++
					onEdit("password")
				}} />
		</dd>
	</div>
{/snippet}

{#if info.error}
	<div class="settings-inline-error" role="alert">{info.error}</div>
{:else if (snapshot?.unavailable?.length ?? 0) > 0}
	<div class="settings-inline-error" role="alert">
		Some details could not be loaded. Refresh to retry.
	</div>
{/if}

{#if !snapshot}
	{#if visible.has("password")}
		<section class="settings-section" aria-labelledby="account-info-account">
			<h3 id="account-info-account">Account</h3>
			<dl class="profile-facts">{@render passwordFact()}</dl>
		</section>
	{/if}
	{#if info.loading}
		<div class="account-settings-loading" role="status">
			<CircleNotchIcon class="spinner" size={16} aria-hidden="true" />
			Loading account info
		</div>
	{/if}
{:else}
	{#if any("displayName", "username", "password", "pin", "gender", "ageGroup", "birthday", "firstAccount")}
		<section class="settings-section" aria-labelledby="account-info-account">
			<h3 id="account-info-account">Account</h3>
			<dl class="profile-facts">
				{#if visible.has("displayName")}
					<div>
						<dt>
							<IdentificationCardIcon
								size={16}
								aria-hidden="true" />Display name
						</dt>
						<dd>
							<span>{account.displayName}</span>
							{@render edit("displayName", "Change display name")}
						</dd>
					</div>
				{/if}
				{#if visible.has("username")}
					<div>
						<dt><AtIcon size={16} aria-hidden="true" />Username</dt>
						<dd><span>{account.username}</span></dd>
					</div>
				{/if}
				{#if visible.has("password")}
					{@render passwordFact()}
				{/if}
				{#if visible.has("pin")}
					<div>
						<dt><LockKeyIcon size={16} aria-hidden="true" />Account PIN</dt>
						<dd>
							{#if missing("pin") || !pinLabels[snapshot.pin]}
								<span class="account-info-missing">Unavailable</span>
							{:else}
								<span>{pinLabels[snapshot.pin]}</span>
								{#if snapshot.pin === PinState.PinLocked}
									<button
										class="settings-row-action profile-fact-action"
										type="button"
										onclick={() => onEdit("pin")}>Unlock</button>
								{/if}
							{/if}
						</dd>
					</div>
				{/if}
				{#if visible.has("gender")}
					<div>
						<dt><UserCircleIcon size={16} aria-hidden="true" />Gender</dt>
						<dd>
							{#if missing("gender")}
								<span class="account-info-missing">Unavailable</span>
							{:else if genders[snapshot.gender]}
								<span>{genders[snapshot.gender]}</span>
							{:else}
								<span class="account-info-missing">Not set</span>
							{/if}
						</dd>
					</div>
				{/if}
				{#if visible.has("ageGroup")}
					<div>
						<dt><CalendarIcon size={16} aria-hidden="true" />Age group</dt>
						<dd>
							{#if snapshot.ageGroup}
								<span>{snapshot.ageGroup}</span>
								{#if ageVerificationLabels[snapshot.ageVerification]}<small
										>{ageVerificationLabels[
											snapshot.ageVerification
										]}</small
									>{/if}
							{:else}
								<span class="account-info-missing">Unavailable</span>
							{/if}
						</dd>
					</div>
				{/if}
				{#if visible.has("birthday")}
					<div>
						<dt><CakeIcon size={16} aria-hidden="true" />Birthday</dt>
						<dd>
							{#if birthday}
								<SensitiveValue label="Birthday"
									>{birthday}</SensitiveValue>
								{#if snapshot.ageVerification === AgeVerification.AgeVerifiedID}<small
										>{ageVerificationLabels[
											snapshot.ageVerification
										]}</small
									>{/if}
							{:else}
								<span class="account-info-missing">Unavailable</span>
							{/if}
						</dd>
					</div>
				{/if}
				{#if visible.has("firstAccount")}
					<div>
						<dt><FlagIcon size={16} aria-hidden="true" />First account</dt>
						<dd>
							{#if snapshot.firstAccount == null}
								<span class="account-info-missing">Unavailable</span>
							{:else}
								<button
									class="hover-value"
									type="button"
									aria-label={`${snapshot.firstAccount ? "Yes" : "No"}. ${firstAccountHint}`}
									data-tooltip={firstAccountHint}
									>{snapshot.firstAccount ? "Yes" : "No"}</button>
							{/if}
						</dd>
					</div>
				{/if}
			</dl>
		</section>
	{/if}

	{#if any("email", "phone")}
		<section class="settings-section" aria-labelledby="account-info-contact">
			<h3 id="account-info-contact">Contact</h3>
			<dl class="profile-facts">
				{#if visible.has("email")}
					<div>
						<dt><EnvelopeIcon size={16} aria-hidden="true" />Email</dt>
						<dd>
							{#if missing("email")}
								<span class="account-info-missing">Unavailable</span>
							{:else if snapshot.email}
								<SensitiveValue label="Email"
									>{snapshot.email}</SensitiveValue>
								<small
									>{snapshot.emailVerified
										? "Verified"
										: "Not verified"}</small>
							{:else}
								<span class="account-info-missing">Not set</span>
							{/if}
							{#if !missing("email")}
								{@render edit("email", "Change email")}
							{/if}
						</dd>
					</div>
				{/if}
				{#if visible.has("phone")}
					<div>
						<dt><PhoneIcon size={16} aria-hidden="true" />Phone number</dt>
						<dd>
							{#if missing("phone")}
								<span class="account-info-missing">Unavailable</span>
							{:else if snapshot.phone}
								<SensitiveValue label="Phone number"
									>{snapshot.phone}</SensitiveValue>
								<small
									>{snapshot.phoneVerified
										? "Verified"
										: "Not verified"}</small>
							{:else}
								<span class="account-info-missing">Not set</span>
							{/if}
						</dd>
					</div>
				{/if}
			</dl>
		</section>
	{/if}

	{#if any("language", "translations", "location")}
		<section class="settings-section" aria-labelledby="account-info-region">
			<h3 id="account-info-region">Language &amp; region</h3>
			<dl class="profile-facts">
				{#if visible.has("language")}
					<div>
						<dt><TranslateIcon size={16} aria-hidden="true" />Language</dt>
						<dd>
							{#if snapshot.language}
								<span>{snapshot.language}</span>
							{:else}
								<span class="account-info-missing">Unavailable</span>
							{/if}
						</dd>
					</div>
				{/if}
				{#if visible.has("translations")}
					<div>
						<dt>
							<GlobeHemisphereWestIcon
								size={16}
								aria-hidden="true" />Automatic translations
						</dt>
						<dd>
							{#if snapshot.autoTranslations == null}
								<span class="account-info-missing">Unavailable</span>
							{:else}
								<span>{snapshot.autoTranslations ? "On" : "Off"}</span>
							{/if}
						</dd>
					</div>
				{/if}
				{#if visible.has("location")}
					<div>
						<dt>
							<MapPinIcon size={16} aria-hidden="true" />Account location
						</dt>
						<dd>
							{#if missing("location")}
								<span class="account-info-missing">Unavailable</span>
							{:else if snapshot.location}
								<SensitiveValue label="Account location"
									>{snapshot.location}</SensitiveValue>
							{:else}
								<span class="account-info-missing">Not set</span>
							{/if}
						</dd>
					</div>
				{/if}
			</dl>
		</section>
	{/if}

	{#if any("voiceChat", "voiceOptIn", "voiceVerification")}
		{@const voice = snapshot.voiceChat}
		<section class="settings-section" aria-labelledby="account-info-voice">
			<h3 id="account-info-voice">Voice chat</h3>
			<dl class="profile-facts">
				{#if visible.has("voiceChat")}
					<AccountVoiceChat voiceChat={voice} label="Status" />
					{#if voice?.state === VoiceChatState.VoiceChatSuspended && voice.suspendedUntilMs}
						<div>
							<dt>
								<HourglassIcon size={16} aria-hidden="true" />Suspension
								ends
							</dt>
							<dd><Timestamp value={voice.suspendedUntilMs} /></dd>
						</div>
					{/if}
				{/if}
				{#if visible.has("voiceOptIn")}
					<div>
						<dt>
							<ToggleRightIcon size={16} aria-hidden="true" />Opted in
						</dt>
						<dd>
							{#if voice}
								<span>{voice.optedIn ? "Yes" : "No"}</span>
								{#if voice.enrollmentDisabled}<small
										>Enrollment disabled</small
									>{/if}
							{:else}
								<span class="account-info-missing">Unavailable</span>
							{/if}
						</dd>
					</div>
				{/if}
				{#if visible.has("voiceVerification")}
					<div>
						<dt>
							<UserCheckIcon size={16} aria-hidden="true" />Voice
							verification
						</dt>
						<dd>
							{#if voice}
								<span
									>{voice.verified
										? "Verified"
										: "Not verified"}</span>
								{#if !voice.verified && voice.canVerifyAge}<small
										>Age check available</small
									>{/if}
							{:else}
								<span class="account-info-missing">Unavailable</span>
							{/if}
						</dd>
					</div>
				{/if}
			</dl>
		</section>
	{/if}
{/if}
