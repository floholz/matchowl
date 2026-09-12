/** Reactive viewport class: `desktop` is the ≥ 900px rule from the layout
 *  decisions (tabs in the top bar, Matches = rail · list · panel). */
import { browser } from '$app/environment';

class Media {
	desktop = $state(false);
	/** Room for the Matches detail panel beside the list (≥ 1100px);
	 *  below that a match opens as its route. */
	panel = $state(false);
	constructor() {
		if (!browser) return;
		const watch = (q: string, set: (v: boolean) => void) => {
			const mq = window.matchMedia(q);
			set(mq.matches);
			mq.addEventListener('change', (e) => set(e.matches));
		};
		watch('(min-width: 900px)', (v) => (this.desktop = v));
		watch('(min-width: 1100px)', (v) => (this.panel = v));
	}
}

export const media = new Media();
