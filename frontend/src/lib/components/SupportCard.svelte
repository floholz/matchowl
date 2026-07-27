<script lang="ts">
	// "Buy me a coffee" card — only renders when a Ko-Fi URL is configured
	// (KOFI_URL env → /api/appconfig). Deliberately quiet: the app is a free
	// gift, this is a door left open, not a paywall pitch.
	import { onMount } from 'svelte';
	import { appConfig } from '$lib/appconfig.svelte';
	import { Coffee } from '@lucide/svelte';

	onMount(() => appConfig.load());
</script>

{#if appConfig.kofiUrl}
	<section class="card support">
		<a class="wrap" href={appConfig.kofiUrl} target="_blank" rel="noopener">
			<span class="ci"><Coffee size={20} /></span>
			<span class="txt">
				<span class="t">Enjoying WM Tips?</span>
				<span class="muted s">It's free with no ads — say thanks with a coffee.</span>
			</span>
			<!-- Official Ko-fi symbol (storage.ko-fi.com/cdn/brandasset); its cup
			     body is white-filled, so it reads on the dark pill as-is. -->
			<span class="pill go">
				Support
				<img class="kfi" src="/assets/kofi_symbol.png" alt="Ko-fi" loading="lazy" />
			</span>
		</a>
	</section>
{/if}

<style>
	.support {
		background: color-mix(in srgb, var(--accent-2) 2%, var(--surface));
		border-color: color-mix(in srgb, var(--accent-2) 15%, var(--border));
	}
	.wrap {
		display: flex;
		align-items: center;
		gap: 0.85rem;
		color: var(--text);
		text-decoration: none;
	}
	.ci {
		display: grid;
		place-items: center;
		width: 38px;
		height: 38px;
		flex: none;
		border-radius: var(--radius-sm);
		background: color-mix(in srgb, var(--accent-2) 12%, var(--surface-2));
		color: var(--accent-2);
	}
	.kfi {
		width: 16px;
		height: 13px;
		flex: none;
		object-fit: contain;
	}
	.txt {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		min-width: 0;
	}
	.t {
		font-weight: 600;
	}
	.s {
		font-size: 0.82rem;
		line-height: 1.4;
	}
	.pill.go {
		margin-left: auto;
		flex: none;
		color: var(--accent-2);
		border-color: color-mix(in srgb, var(--accent-2) 40%, var(--border));
		white-space: nowrap;
	}
</style>
