<script lang="ts">
	// "Update ready" toast. A new build ships a new service worker that
	// activates at once (skipWaiting + claim); this page keeps running the
	// old JavaScript until it reloads, which can break against a changed API.
	// When a new worker takes control of an already-controlled page, offer
	// the reload. First installs (no controller before) stay silent.
	import { onMount } from 'svelte';
	import { RefreshCw, X } from '@lucide/svelte';

	let show = $state(false);

	onMount(() => {
		if (!('serviceWorker' in navigator)) return;
		const sw = navigator.serviceWorker;
		if (!sw.controller) return;
		const onChange = () => {
			show = true;
		};
		sw.addEventListener('controllerchange', onChange);
		return () => sw.removeEventListener('controllerchange', onChange);
	});
</script>

{#if show}
	<div class="toast" role="status">
		<RefreshCw size={16} class="ico" />
		<span class="msg">A new version of Matchowl is ready.</span>
		<button class="btn reload" onclick={() => location.reload()}>Reload</button>
		<button class="x" aria-label="Later" onclick={() => (show = false)}><X size={16} /></button>
	</div>
{/if}

<style>
	.toast {
		position: fixed;
		left: 50%;
		bottom: calc(env(safe-area-inset-bottom, 0px) + 76px);
		transform: translateX(-50%);
		z-index: 60;
		display: flex;
		align-items: center;
		gap: 0.6rem;
		max-width: min(92vw, 440px);
		padding: 0.55rem 0.6rem 0.55rem 0.85rem;
		background: var(--surface);
		border: 1px solid color-mix(in srgb, var(--accent) 35%, var(--border));
		border-radius: 999px;
		box-shadow: 0 10px 30px rgba(0, 0, 0, 0.35);
		font-size: 0.9rem;
	}
	:global(.toast .ico) {
		color: var(--accent);
		flex: none;
	}
	.msg {
		flex: 1;
		min-width: 0;
	}
	.btn.reload {
		width: auto;
		padding: 0.4rem 0.8rem;
		font-size: 0.82rem;
	}
	.x {
		display: inline-grid;
		place-items: center;
		width: 30px;
		height: 30px;
		border-radius: 999px;
		background: transparent;
		color: var(--muted);
		border: 0;
		cursor: pointer;
	}
	@media (min-width: 900px) {
		.toast {
			bottom: 24px;
		}
	}
</style>
