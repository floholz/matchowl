// In-app guides: a short run of steps, each a bubble pointing at a real
// element ([data-guide="…"]) or, without a target, a card on its own. A
// guide plays once per account (users.guides), from the screen it is
// about, the first time that screen is used; /help can replay it.
//
// Only one overlay at a time: a sheet that is open (the notifications or
// verify-email prompt) holds guides back, and a guide waiting to start
// begins once nothing holds it. Rendered by components/GuideHost.svelte.
import { pb } from './pb';
import { auth } from './auth.svelte';

export interface GuideStep {
	/** data-guide name of the element to point at; none = a card alone. A
	 *  step whose element is not on screen is skipped. */
	target?: string;
	title: string;
	body: string;
}

export interface Guide {
	id: string;
	steps: GuideStep[];
}

class GuideStore {
	/** The guide on screen and the step shown. */
	active = $state<Guide | null>(null);
	index = $state(0);
	/** Open overlays that hold guides back, by name. */
	private holds = $state<string[]>([]);
	private queued: Guide | null = null;
	/** Seen this session before the account write lands. */
	private local = new Set<string>();

	seen(id: string): boolean {
		return this.local.has(id) || !!auth.user?.guides?.[id];
	}

	/** Play a guide unless this account has been through it. */
	offer(g: Guide) {
		if (!auth.user || this.seen(g.id) || this.active?.id === g.id) return;
		this.play(g);
	}

	/** Play a guide now (or once nothing holds it), seen or not. */
	play(g: Guide) {
		if (this.active) return;
		if (this.holds.length) {
			this.queued = g;
			return;
		}
		this.active = g;
		this.index = 0;
	}

	/** An overlay opened (true) or closed (false). */
	hold(name: string, on: boolean) {
		if (on && !this.holds.includes(name)) this.holds = [...this.holds, name];
		if (!on && this.holds.includes(name)) {
			this.holds = this.holds.filter((h) => h !== name);
			if (!this.holds.length && this.queued) {
				const g = this.queued;
				this.queued = null;
				this.play(g);
			}
		}
	}

	next() {
		if (!this.active) return;
		if (this.index < this.active.steps.length - 1) this.index++;
		else this.finish();
	}

	back() {
		if (this.index > 0) this.index--;
	}

	/** Done or skipped: either way it counts as seen. */
	finish() {
		const g = this.active;
		this.active = null;
		this.index = 0;
		if (g) void this.markSeen(g.id);
	}

	/** Forget a guide so it plays again where it belongs. */
	async forget(id: string) {
		this.local.delete(id);
		const u = auth.user;
		if (!u?.guides?.[id]) return;
		const guides = { ...u.guides };
		delete guides[id];
		await this.write(guides);
	}

	private async markSeen(id: string) {
		this.local.add(id);
		const u = auth.user;
		if (!u || u.guides?.[id]) return;
		await this.write({ ...(u.guides ?? {}), [id]: new Date().toISOString() });
	}

	private async write(guides: Record<string, string>) {
		const u = auth.user;
		if (!u) return;
		try {
			await pb.collection('users').update(u.id, { guides });
			await pb.collection('users').authRefresh();
		} catch {
			/* the session set keeps it seen until the next try */
		}
	}
}

export const guide = new GuideStore();
