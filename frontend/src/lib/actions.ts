/** Collapse a sticky page header on scroll (hide the big title + intro, keep
 *  the kicker line + tabs), expanding again only near the very top.
 *
 *  Uses hysteresis — collapse past `collapseAt`, expand only back under
 *  `expandAt` — so slow scrolling or sitting exactly on a single threshold
 *  can't make the header oscillate (the collapse itself shifts layout height,
 *  which would otherwise re-cross a single cutoff and vibrate). The class is
 *  only touched on an actual state change, coalesced via rAF. */
export function collapseOnScroll(
	node: HTMLElement,
	opts: { collapseAt?: number; expandAt?: number } = {}
) {
	const collapseAt = opts.collapseAt ?? 64;
	const expandAt = opts.expandAt ?? 6;

	let collapsed = window.scrollY > collapseAt;
	let ticking = false;
	node.classList.toggle('scrolled', collapsed);

	const apply = () => {
		ticking = false;
		const y = window.scrollY;
		if (!collapsed && y > collapseAt) {
			collapsed = true;
			node.classList.add('scrolled');
		} else if (collapsed && y < expandAt) {
			collapsed = false;
			node.classList.remove('scrolled');
			// Expanding grows the header, which would push us off the top and
			// re-cross the threshold (jump/oscillation). We only get here
			// within `expandAt` px of the top, so pin to 0 and let the header
			// grow from a stable anchor.
			window.scrollTo(0, 0);
		}
	};
	const onScroll = () => {
		if (ticking) return;
		ticking = true;
		requestAnimationFrame(apply);
	};

	window.addEventListener('scroll', onScroll, { passive: true });
	return {
		destroy() {
			window.removeEventListener('scroll', onScroll);
		}
	};
}

/** Render a node at the end of <body>. Bottom sheets need it: the app shell
 *  is its own stacking context (z-index 2), so anything inside it — however
 *  high its own z-index — sits below the tab bar (z-index 50), which then
 *  covers the sheet's bottom and catches its touches. */
export function portal(node: HTMLElement) {
	document.body.appendChild(node);
	return {
		destroy() {
			node.remove();
		}
	};
}

/** Hold the page still while a modal sheet is open, so a swipe scrolls the
 *  sheet and not the list behind it (iOS hands a fixed element's touches to
 *  the page otherwise). Counted, so stacked sheets unlock once. */
let scrollLocks = 0;
export function lockScroll(_node: HTMLElement) {
	if (scrollLocks++ === 0) document.documentElement.classList.add('scroll-locked');
	return {
		destroy() {
			if (--scrollLocks === 0) document.documentElement.classList.remove('scroll-locked');
		}
	};
}
