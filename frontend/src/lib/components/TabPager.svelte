<!-- TabPager: the pane of a tabbed page that swipes like a native pager on
     touch. The pane follows the finger and the neighbouring tab slides in
     beside it, then settles (past 28% of the width, or a quick flick) or
     springs back. Only the current tab and, while dragging, its neighbour
     are rendered — `pane` renders one tab by id. The transform is applied
     only during a drag or its settle, so nothing fixed inside a pane loses
     its containing block. A touch that starts on something that scrolls
     sideways itself (round pills, a wide table) belongs to that element; a
     vertical-ish gesture is the page's scroll and never starts a drag. -->
<script lang="ts" generics="T extends string">
	import type { Snippet } from 'svelte';

	let {
		tabs,
		current,
		onchange,
		pane
	}: {
		tabs: readonly T[];
		current: T;
		onchange: (id: T) => void;
		pane: Snippet<[T]>;
	} = $props();

	let sx = 0;
	let sy = 0;
	let st = 0;
	let swipeOwned = false;
	let decided = false;
	let dragging = false;
	let dragX = $state(0);
	let settling = $state(false);
	let neighbour = $state<T | ''>('');
	let neighbourSide = $state<'left' | 'right'>('right');
	let paneEl = $state<HTMLElement | null>(null);

	function insideHorizontalScroller(el: Element | null): boolean {
		for (let n = el; n && n !== paneEl; n = n.parentElement) {
			const h = n as HTMLElement;
			if (h.scrollWidth > h.clientWidth + 1) {
				const ox = getComputedStyle(h).overflowX;
				if (ox === 'auto' || ox === 'scroll') return true;
			}
		}
		return false;
	}
	function touchStart(e: TouchEvent) {
		if (settling) return;
		sx = e.touches[0].clientX;
		sy = e.touches[0].clientY;
		st = Date.now();
		decided = false;
		dragging = false;
		swipeOwned = insideHorizontalScroller(e.target as Element | null);
	}
	function touchMove(e: TouchEvent) {
		if (swipeOwned || settling) return;
		const dx = e.touches[0].clientX - sx;
		const dy = e.touches[0].clientY - sy;
		if (!decided) {
			if (Math.abs(dx) < 10 && Math.abs(dy) < 10) return;
			decided = true;
			if (Math.abs(dx) < Math.abs(dy) * 1.2) return; // a scroll, not a swipe
			dragging = true;
			const i = tabs.indexOf(current);
			neighbour = tabs[i + (dx < 0 ? 1 : -1)] ?? '';
			neighbourSide = dx < 0 ? 'right' : 'left';
		}
		if (!dragging) return;
		// No tab that way: give a little, with resistance.
		dragX = neighbour ? dx : dx / 4;
	}
	function touchEnd() {
		if (!dragging) return;
		dragging = false;
		const w = paneEl?.clientWidth ?? 1;
		const dx = dragX;
		const flick = Date.now() - st < 300 && Math.abs(dx) > 40;
		const target = neighbour && (Math.abs(dx) > w * 0.28 || flick) ? neighbour : '';
		settling = true;
		if (target) {
			dragX = dx < 0 ? -w : w;
			setTimeout(() => {
				settling = false;
				dragX = 0;
				neighbour = '';
				onchange(target);
			}, 230);
		} else {
			dragX = 0;
			setTimeout(() => {
				settling = false;
				neighbour = '';
			}, 230);
		}
	}
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="pane"
	bind:this={paneEl}
	ontouchstart={touchStart}
	ontouchmove={touchMove}
	ontouchend={touchEnd}
	ontouchcancel={touchEnd}
>
	<div
		class="track"
		class:anim={settling}
		style:transform={dragX || settling ? `translateX(${dragX}px)` : undefined}
	>
		{#if neighbour && neighbourSide === 'left'}
			<div class="slide side left">{@render pane(neighbour)}</div>
		{/if}
		<div class="slide">{@render pane(current)}</div>
		{#if neighbour && neighbourSide === 'right'}
			<div class="slide side right">{@render pane(neighbour)}</div>
		{/if}
	</div>
</div>

<style>
	.pane {
		min-height: 40vh;
		position: relative;
		overflow: clip; /* no scroll container: sticky rows inside keep working */
	}
	.track {
		position: relative;
	}
	.track.anim {
		transition: transform 220ms cubic-bezier(0.2, 0.8, 0.2, 1);
	}
	.slide.side {
		position: absolute;
		top: 0;
		width: 100%;
	}
	.slide.left {
		left: -100%;
	}
	.slide.right {
		left: 100%;
	}
</style>
