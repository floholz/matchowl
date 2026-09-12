/** Reactive viewport class: `desktop` is the ≥ 900px rule from the layout
 *  decisions (tabs in the top bar, Matches = rail · list · panel). */
import { browser } from '$app/environment';

class Media {
	desktop = $state(false);
	constructor() {
		if (!browser) return;
		const mq = window.matchMedia('(min-width: 900px)');
		this.desktop = mq.matches;
		mq.addEventListener('change', (e) => (this.desktop = e.matches));
	}
}

export const media = new Media();
