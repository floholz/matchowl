/** Page chrome: what the (mobile) top bar shows for the current page.
 *  Home shows the wordmark; list pages a title; detail pages back + title
 *  (+ a context line). Desktop always shows the wordmark + nav links. */
export interface Chrome {
	title?: string;
	/** Href of the back button; shown with the title when set. */
	back?: string;
	/** Small line under the title (detail pages). */
	context?: string;
}

class Shell {
	title = $state('');
	back = $state('');
	context = $state('');

	set(c: Chrome) {
		this.title = c.title ?? '';
		this.back = c.back ?? '';
		this.context = c.context ?? '';
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
