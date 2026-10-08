<script lang="ts">
	// Classic user-circle: the user's picked raven (lib/avatars.ts), else
	// their uploaded or Google photo, else the raven in a palette chosen
	// from their id so users stay visually distinguishable.
	import { knownPreset, resolvePreset } from '$lib/avatars';

	let {
		name,
		src = null,
		id = '',
		preset = '',
		size = 36
	}: {
		name: string;
		src?: string | null;
		/** The user id: seeds the default palette. */
		id?: string;
		/** users.avatarPreset, "<template>:<palette>". */
		preset?: string | null;
		size?: number;
	} = $props();

	let drawn = $derived(resolvePreset(preset, id || name));
</script>

{#if src && !knownPreset(preset)}
	<img
		class="avatar"
		{src}
		alt={name}
		style="width:{size}px;height:{size}px"
		referrerpolicy="no-referrer"
	/>
{:else}
	<svg
		class="avatar"
		viewBox="0 0 256 256"
		width={size}
		height={size}
		role="img"
		aria-label={name}
		style="--pp-h:{drawn.palette.h};--pp-a:{drawn.palette.a};--pp-d:{drawn.palette.d}"
	>
		<!-- eslint-disable-next-line svelte/no-at-html-tags — static art from lib/avatars.ts -->
		{@html drawn.template.svg}
	</svg>
{/if}

<style>
	.avatar {
		display: inline-grid;
		place-items: center;
		border-radius: 50%;
		object-fit: cover;
		border: 2px solid var(--border);
		flex: none;
	}
	svg.avatar {
		overflow: hidden;
	}
</style>
