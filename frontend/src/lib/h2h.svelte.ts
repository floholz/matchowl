import { api, type H2HMePool, type H2HPickMark } from './api';
import { auth } from './auth.svelte';

// My head-to-head pools at a glance, loaded once (GET /api/h2h/me) and
// shared by the match rows (pick marks on the capsule), the competition
// hub (the rival strip) and Home (the duel cards). Refreshed after a pick
// changes and after a short while on the next read; the tip drawer keeps
// its own per-match request for the full detail.
const TTL_MS = 60_000;

class H2HStore {
	pools = $state<H2HMePool[]>([]);
	loaded = $state(false);
	private at = 0;
	private inflight: Promise<void> | null = null;

	/** Load (or refresh when stale). Safe to call from any read. */
	load(force = false): Promise<void> {
		if (!auth.isAuthed) {
			this.pools = [];
			this.loaded = true;
			return Promise.resolve();
		}
		if (this.inflight) return this.inflight;
		if (!force && this.loaded && Date.now() - this.at < TTL_MS) return Promise.resolve();
		this.inflight = api
			.h2hMe()
			.then((r) => {
				this.pools = r.pools;
				this.at = Date.now();
			})
			.catch(() => {
				/* keep what we have */
			})
			.finally(() => {
				this.loaded = true;
				this.inflight = null;
			});
		return this.inflight;
	}

	/** A pick changed somewhere: re-read on the next tick. */
	invalidate() {
		this.at = 0;
		this.load(true).catch(() => {});
	}

	/** The pools that play a season. */
	forSeason(tournamentId: string): H2HMePool[] {
		return this.pools.filter((p) => p.season === tournamentId);
	}

	/** What the rows show for a match, merged over my pools (any pool's pick
	 *  marks the row; the drawer lists them per pool). Null when none. */
	marks(matchId: string): (H2HPickMark & { pools: number }) | null {
		let out: (H2HPickMark & { pools: number }) | null = null;
		for (const p of this.pools) {
			const m = p.picks[matchId];
			if (!m) continue;
			out ??= { saved: false, banned: false, rivalSaved: false, rivalBanned: false, pools: 0 };
			out.saved ||= m.saved;
			out.banned ||= m.banned;
			out.rivalSaved ||= m.rivalSaved;
			out.rivalBanned ||= m.rivalBanned;
			out.pools++;
		}
		return out;
	}
}

export const h2hStore = new H2HStore();
