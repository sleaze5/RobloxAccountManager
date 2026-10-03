<script lang="ts">
	import type { AccountProfileSnapshot } from "../backend/bridge"

	let { snapshot }: { snapshot: AccountProfileSnapshot } = $props()
</script>

{#if snapshot.premium}
	<span class="profile-badge" data-tooltip="Roblox Premium">
		<svg
			viewBox="0 0 32 32"
			width="12"
			height="12"
			fill="currentColor"
			aria-hidden="true">
			<path
				d="M23 4C25.7614 4 28 6.23858 28 9V23C28 25.7614 25.7614 28 23 28H15C14.4477 28 14 27.5523 14 27C14 26.4477 14.4477 26 15 26H23C24.6569 26 26 24.6569 26 23V9C26 7.34315 24.6569 6 23 6H9C7.34315 6 6 7.34315 6 9V27C6 27.5523 5.55228 28 5 28C4.44772 28 4 27.5523 4 27V9C4 6.23858 6.23858 4 9 4H23Z" />
			<path
				d="M20 10C21.1046 10 22 10.8954 22 12V20C22 21.1046 21.1046 22 20 22H15C14.4477 22 14 21.5523 14 21C14 20.4477 14.4477 20 15 20H20V12H12V27C12 27.5523 11.5523 28 11 28C10.4477 28 10 27.5523 10 27V12C10 10.8954 10.8954 10 12 10H20Z" />
		</svg>
		Premium
	</span>
{/if}
{#if snapshot.plus}
	<span class="profile-badge" data-tooltip="Roblox Plus">
		<svg
			viewBox="0 0 32 32"
			width="12"
			height="12"
			fill="currentColor"
			aria-hidden="true">
			<path
				d="M14.5 26V17.5H6C5.17157 17.5 4.5 16.8284 4.5 16C4.5 15.1716 5.17157 14.5 6 14.5H14.5V6C14.5 5.17157 15.1716 4.5 16 4.5C16.8284 4.5 17.5 5.17157 17.5 6V14.5H26L26.1533 14.5078C26.9097 14.5846 27.5 15.2233 27.5 16C27.5 16.7767 26.9097 17.4154 26.1533 17.4922L26 17.5H17.5V26C17.5 26.8284 16.8284 27.5 16 27.5C15.1716 27.5 14.5 26.8284 14.5 26Z" />
		</svg>
		Plus
	</span>
{/if}
