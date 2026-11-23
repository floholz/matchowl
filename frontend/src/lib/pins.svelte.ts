// Pinned competitions (PLAN.md, "pools at the center"): the competition
// chips in the Matches filter row. The competitions your pools play are
// always pinned; others you pin yourself in the "More" sheet, stored on the
// account (users.pinnedCompetitions) so they follow you across devices.
import { pb } from './pb';
import { auth } from './auth.svelte';
import { feedStore } from './feed.svelte';
import { tournamentStore, defaultSeason, type Competition, type Tournament } from './tournament.svelte';

export interface PinnedCompetition {
	competition: Competition;
	/** Pinned because a pool of yours plays it (can't be unpinned). */
	pool: boolean;
}

/** A competition with its current season, for the sheet. */
export interface CompetitionEntry {
	competition: Competition;
	season: Tournament;
}

class PinStore {
	/** Optimistic copy while a write is on its way. */
	private pending = $state<string[] | null>(null);

	/** Competition keys you pinned yourself, in order. */
	get mine(): string[] {
		return this.pending ?? auth.user?.pinnedCompetitions ?? [];
	}

	/** Keys of the competitions your pools play. */
	get poolKeys(): string[] {
		const out: string[] = [];
		for (const t of feedStore.poolSeasons) if (t.competition && !out.includes(t.competition)) out.push(t.competition);
		return out;
	}

	/** Every competition with a visible season (not draft, not archived),
	 *  keyed by competition key, with the season Matches would show. */
	get entries(): Map<string, CompetitionEntry> {
		const by = new Map<string, Tournament[]>();
		for (const t of tournamentStore.list) {
			if (!t.competition || t.status === 'draft' || t.status === 'archived') continue;
			const k = t.competition.key;
			if (!by.has(k)) by.set(k, []);
			by.get(k)!.push(t);
		}
		const out = new Map<string, CompetitionEntry>();
		for (const [k, seasons] of by) {
			const season = defaultSeason(seasons);
			if (season) out.set(k, { competition: season.competition, season });
		}
		return out;
	}

	/** The chips: pool competitions first, then your pins. */
	get pinned(): PinnedCompetition[] {
		const entries = this.entries;
		const pools = this.poolKeys;
		const out: PinnedCompetition[] = [];
		for (const k of pools) {
			const e = entries.get(k);
			if (e) out.push({ competition: e.competition, pool: true });
		}
		for (const k of this.mine) {
			if (pools.includes(k)) continue;
			const e = entries.get(k);
			if (e) out.push({ competition: e.competition, pool: false });
		}
		return out;
	}

	isPinned(key: string): boolean {
		return this.poolKeys.includes(key) || this.mine.includes(key);
	}

	/** Pin or unpin a competition (pool competitions stay pinned). */
	async toggle(key: string) {
		const u = auth.user;
		if (!u || this.poolKeys.includes(key)) return;
		const cur = this.mine;
		const next = cur.includes(key) ? cur.filter((k) => k !== key) : [...cur, key];
		this.pending = next;
		try {
			await pb.collection('users').update(u.id, { pinnedCompetitions: next });
			await pb.collection('users').authRefresh();
		} catch {
			/* the next refresh shows what was stored */
		} finally {
			this.pending = null;
		}
	}
}

export const pins = new PinStore();
