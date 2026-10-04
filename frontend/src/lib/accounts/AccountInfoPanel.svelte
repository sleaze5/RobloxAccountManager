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
	import AtSign from "@lucide/svelte/icons/at-sign"
	import Cake from "@lucide/svelte/icons/cake"
	import CalendarRange from "@lucide/svelte/icons/calendar-range"
	import Flag from "@lucide/svelte/icons/flag"
	import Languages from "@lucide/svelte/icons/languages"
	import LoaderCircle from "@lucide/svelte/icons/loader-circle"
	import Mail from "@lucide/svelte/icons/mail"
	import MapPin from "@lucide/svelte/icons/map-pin"
	import Phone from "@lucide/svelte/icons/phone"
	import UserRound from "@lucide/svelte/icons/user-round"
	import Globe2 from "@lucide/svelte/icons/earth"
	import moment from "moment"
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
		genders: Record<string, string> = { male: "Male", female: "Female" }

	const snapshot = $derived(info.snapshot),
		birthday = $derived.by(() => {
			const date = moment(snapshot?.birthdate ?? "", "YYYY-MM-DD", true)
			return date.isValid()
				? `${date.format("MMMM DD, YYYY")} (${moment().diff(date, "years")}y)`
				: ""
		})

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
			<LoaderCircle class="spinner" size={14} aria-hidden="true" />
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
						<dt><AtSign size={14} aria-hidden="true" />Username</dt>
						<dd><span>{username}</span></dd>
					</div>
				{/if}
				{#if visible.has("gender")}
					<div>
						<dt><UserRound size={14} aria-hidden="true" />Gender</dt>
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
						<dt><CalendarRange size={14} aria-hidden="true" />Age group</dt>
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
						<dt><Cake size={14} aria-hidden="true" />Birthday</dt>
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
						<dt><Flag size={14} aria-hidden="true" />First account</dt>
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
						<dt><Mail size={14} aria-hidden="true" />Email</dt>
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
						<dt><Phone size={14} aria-hidden="true" />Phone number</dt>
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
						<dt><Languages size={14} aria-hidden="true" />Language</dt>
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
							<Globe2 size={14} aria-hidden="true" />Automatic
							translations
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
						<dt><MapPin size={14} aria-hidden="true" />Account location</dt>
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
