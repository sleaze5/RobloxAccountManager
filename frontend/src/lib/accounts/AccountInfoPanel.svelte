<script lang="ts" module>
	export const accountInfoFields = [
		{ id: "email", search: "email address contact verified" },
		{ id: "phone", search: "phone number contact verified" },
		{ id: "username", search: "username name" },
		{ id: "gender", search: "gender" },
		{ id: "ageGroup", search: "age group checked" },
		{ id: "birthday", search: "birthday birthdate date of birth verified" },
		{ id: "firstAccount", search: "first account original alt" },
		{ id: "language", search: "language locale" },
		{ id: "translations", search: "automatic translations experience" },
		{ id: "location", search: "account location country region" },
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
	import MapPinIcon from "phosphor-svelte/lib/MapPinIcon"
	import PhoneIcon from "phosphor-svelte/lib/PhoneIcon"
	import UserCircleIcon from "phosphor-svelte/lib/UserCircleIcon"
	import GlobeHemisphereWestIcon from "phosphor-svelte/lib/GlobeHemisphereWestIcon"
	import SensitiveValue from "../shared/SensitiveValue.svelte"
	import { AgeVerification } from "../backend/bridge"
	import type { AccountInfoState } from "./account-info-state.svelte"
	import { ageVerificationLabels } from "./account-model"

	let {
		info,
		username,
		visible,
	}: {
		info: AccountInfoState
		username: string
		visible: ReadonlySet<AccountInfoFieldId>
	} = $props()

	const firstAccountHint =
			"Whether Roblox thinks this is the owner's first account. It may not be accurate.",
		genders: Record<string, string> = { male: "Male", female: "Female" },
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

{#if info.error}
	<div class="settings-inline-error" role="alert">{info.error}</div>
{:else if (snapshot?.unavailable?.length ?? 0) > 0}
	<div class="settings-inline-error" role="alert">
		Some details could not be loaded. Refresh to retry.
	</div>
{/if}

{#if !snapshot}
	{#if info.loading}
		<div class="account-settings-loading" role="status">
			<CircleNotchIcon class="spinner" size={16} aria-hidden="true" />
			Loading account info
		</div>
	{/if}
{:else}
	{#if any("username", "gender", "ageGroup", "birthday", "firstAccount")}
		<section class="settings-section" aria-labelledby="account-info-account">
			<h3 id="account-info-account">Account</h3>
			<dl class="profile-facts">
				{#if visible.has("username")}
					<div>
						<dt><AtIcon size={16} aria-hidden="true" />Username</dt>
						<dd><span>{username}</span></dd>
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
{/if}
