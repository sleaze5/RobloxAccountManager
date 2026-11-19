<script lang="ts">
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import ProfileImage from "../shared/ProfileImage.svelte"

	type Size = "sm" | "md" | "xl"
	let { url, size = "md" }: { url: string; size?: Size } = $props()

	const sizes: Record<Size, string> = {
		sm: "size-8 rounded-md",
		md: "size-9 rounded-md",
		xl: "size-24 rounded-md",
	}
</script>

<span
	class={[
		"grid shrink-0 place-items-center overflow-hidden border border-line-strong bg-raised",
		sizes[size],
	]}
	aria-hidden="true">
	<ProfileImage {url} fallbackIcon={usersThreeIcon} />
</span>
