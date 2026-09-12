/** Page chrome: what the (mobile) top bar shows for the current page.
 *  Home shows the wordmark; list pages a title; detail pages back + title
 *  (+ a context line). Desktop always shows the wordmark + nav links. */
import type { Snippet } from 'svelte';

export interface Chrome {
	title?: string;
	/** Href of the back button; shown with the title when set. */
	back?: string;
	/** Small line under the title (detail pages). */
	context?: string;
	/** Desktop: let the content spread to --maxw-wide (Matches, Home). */
	wide?: boolean;
	/** Mobile: the page draws the whole left part of the top bar itself
	 *  (the competition hub: back, crest, name, season, Playing). */
	bar?: Snippet;
}

class Shell {
	title = $state('');
	back = $state('');
	context = $state('');
	wide = $state(false);
	bar = $state.raw<Snippet | undefined>(undefined);

	set(c: Chrome) {
		this.title = c.title ?? '';
		this.back = c.back ?? '';
		this.context = c.context ?? '';
		this.wide = c.wide ?? false;
		this.bar = c.bar;
	}
	clear() {
		this.set({});
	}
}

export const shell = new Shell();

/** Declare a page's chrome from its script (reactive; cleared on leave). */
export function pageChrome(get: () => Chrome) {
	$effect(() => {
		shell.set(get());
		return () => shell.clear();
	});
}
