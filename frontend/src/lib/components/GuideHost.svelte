<!-- The guide on screen (lib/guide.svelte.ts): the page dimmed with the
     step's element cut out, and a bubble beside it — or, for a step
     without a target, the bubble alone in the middle. A target that does
     not show up within a moment is skipped. Mounted once in the layout. -->
<script lang="ts">
	import { guide } from '$lib/guide.svelte';
	import { X } from '@lucide/svelte';

	const PAD = 6; // space around the cut-out
	const GAP = 12; // between cut-out and bubble
	const EDGE = 16; // the page gutter

	let step = $derived(guide.active?.steps[guide.index] ?? null);
	let total = $derived(guide.active?.steps.length ?? 0);
	let el = $state<HTMLElement | null>(null);
	let rect = $state<DOMRect | null>(null);
	let ready = $state(false);
	let bubble = $state<HTMLElement | null>(null);
	let bubbleH = $state(0);
	let nextBtn = $state<HTMLButtonElement | null>(null);
	let vw = $state(0);
	let vh = $state(0);
	// Which way the last move went, so a missing target skips onwards.
	let dir = 1;

	const shown = (e: Element | null): e is HTMLElement =>
		!!e && e instanceof HTMLElement && e.getClientRects().length > 0;

	// Resolve the step's target: wait a little for it to render (picks and
	// boards load), then point at it or skip it.
	$effect(() => {
		const s = step;
		const g = guide.active;
		const i = guide.index;
		el = null;
		rect = null;
		ready = false;
		if (!s || !g) return;
		if (!s.target) {
			ready = true;
			return;
		}
		let tries = 0;
		const find = () => {
			if (guide.active !== g || guide.index !== i) return;
			const found = document.querySelector(`[data-guide="${s.target}"]`);
			if (shown(found)) {
				el = found;
				found.scrollIntoView({ block: 'center', behavior: 'smooth' });
				ready = true;
				return;
			}
			if (++tries < 15) timer = setTimeout(find, 100);
			else if (dir < 0 && i > 0) guide.back();
			else guide.next();
		};
		let timer = setTimeout(find, 0);
		return () => clearTimeout(timer);
	});

	// Follow the target while it scrolls or the page moves.
	$effect(() => {
		if (!guide.active) return;
		let raf = 0;
		const tick = () => {
			vw = window.innerWidth;
			vh = window.innerHeight;
			rect = el ? el.getBoundingClientRect() : null;
			if (bubble) bubbleH = bubble.offsetHeight;
			raf = requestAnimationFrame(tick);
		};
		tick();
		return () => cancelAnimationFrame(raf);
	});

	$effect(() => {
		if (ready) nextBtn?.focus({ preventScroll: true });
	});

	let hole = $derived(
		rect
			? { x: rect.left - PAD, y: rect.top - PAD, w: rect.width + 2 * PAD, h: rect.height + 2 * PAD }
			: null
	);
	let width = $derived(Math.min(340, vw - 2 * EDGE));
	let place = $derived.by(() => {
		if (!hole) return `left:${(vw - width) / 2}px;top:${Math.max(EDGE, (vh - bubbleH) / 2)}px`;
		const left = Math.min(Math.max(hole.x + hole.w / 2 - width / 2, EDGE), vw - EDGE - width);
		const below = hole.y + hole.h + GAP;
		const top =
			below + bubbleH <= vh - EDGE || hole.y - GAP - bubbleH < EDGE
				? Math.min(below, vh - EDGE - bubbleH)
				: hole.y - GAP - bubbleH;
		return `left:${left}px;top:${Math.max(EDGE, top)}px`;
	});

	function next() {
		dir = 1;
		guide.next();
	}
	function back() {
		dir = -1;
		guide.back();
	}
	function onkey(e: KeyboardEvent) {
		if (!guide.active) return;
		if (e.key === 'Escape') guide.finish();
		else if (e.key === 'ArrowRight') next();
		else if (e.key === 'ArrowLeft') back();
	}
</script>

<svelte:window onkeydown={onkey} />

{#if guide.active && step}
	<!-- Swallows taps on the page while the guide plays. -->
	<div class="catch" class:dim={!hole} aria-hidden="true"></div>
	{#if hole}
		<div class="hole" style="left:{hole.x}px;top:{hole.y}px;width:{hole.w}px;height:{hole.h}px"></div>
	{/if}
	{#if ready}
		<div class="bubble" bind:this={bubble} style="width:{width}px;{place}" role="dialog" aria-modal="true" aria-labelledby="guide-title">
			<button class="close" onclick={() => guide.finish()} aria-label="Close the guide"><X size={16} /></button>
			<p class="count">{guide.index + 1} / {total}</p>
			<h3 id="guide-title">{step.title}</h3>
			<p class="body">{step.body}</p>
			<div class="acts">
				{#if guide.index > 0}
					<button class="btn ghost" onclick={back}>Back</button>
				{:else}
					<button class="btn ghost" onclick={() => guide.finish()}>Skip</button>
				{/if}
				<button class="btn" bind:this={nextBtn} onclick={next}>{guide.index === total - 1 ? 'Got it' : 'Next'}</button>
			</div>
		</div>
	{/if}
{/if}

<style>
	.catch {
		position: fixed;
		inset: 0;
		z-index: 90;
	}
	.catch.dim {
		background: rgb(0 0 0 / 0.62);
	}
	.hole {
		position: fixed;
		z-index: 91;
		border-radius: var(--radius-sm);
		box-shadow:
			0 0 0 2px var(--accent),
			0 0 0 200vmax rgb(0 0 0 / 0.62);
		pointer-events: none;
		transition:
			left 0.2s ease,
			top 0.2s ease,
			width 0.2s ease,
			height 0.2s ease;
	}
	.bubble {
		position: fixed;
		z-index: 92;
		padding: 0.9rem 1rem 0.8rem;
		border: 1px solid color-mix(in srgb, var(--accent) 45%, var(--border));
		border-radius: var(--radius);
		background: var(--surface);
		color: var(--text);
		box-shadow: var(--shadow-pop);
	}
	.close {
		position: absolute;
		top: 0.5rem;
		right: 0.5rem;
		display: inline-flex;
		padding: 0.25rem;
		border: none;
		background: none;
		color: var(--muted);
		cursor: pointer;
	}
	.count {
		margin: 0 0 0.2rem;
		font-size: 0.72rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		color: var(--accent);
	}
	h3 {
		margin: 0 1.5rem 0.35rem 0;
		font-size: 1rem;
	}
	.body {
		margin: 0;
		font-size: 0.88rem;
		line-height: 1.45;
		color: var(--muted);
	}
	.acts {
		display: flex;
		justify-content: flex-end;
		gap: 0.5rem;
		margin-top: 0.85rem;
	}
	.acts .btn {
		width: auto;
		padding-inline: 1rem;
	}
</style>
