import { pb } from './pb';
import { auth } from './auth.svelte';
import { serverClock } from './serverclock.svelte';
import type { Match, Team, Tip } from './tips.svelte';

/** One row of the feed: a match (same field names as the matches
 *  collection, so the shared TipCard renders it directly) denormalized
 *  with its tournament and the caller's tip. Mirrors /api/feed. */
export interface FeedMatch extends Match {
	tournament: { id: string; slug: string; name: string; shortName: string; status: string };
	stageName: string;
	knockout: boolean;
	myTip?: {
		ftHome: number;
		ftAway: number;
		etHome: number;
		etAway: number;
		penWinner: string;
		advancer: string;
		points?: number;
	};
}

export interface FeedDeadline {
	type: 'forecast';
	tournament: FeedMatch['tournament'];
	locksAt: string;
	hasForecast: boolean;
}

export interface FeedSuggestion {
	id: string;
	slug: string;
	name: string;
	status: string;
	startsAt: string;
	leagueMates: number;
}

export interface FeedDay {
	/** Local date key, e.g. "2026-06-25". */
	key: string;
	label: string;
	isToday: boolean;
	matches: FeedMatch[];
}

/** Deadline cards only surface this close to their lock. */
const DEADLINE_LEAD_DAYS = 14;

function localDayKey(iso: string): string {
	const d = new Date(iso);
	return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

/** The feed store: a today-centered window of matches across every
 *  tournament the user plays. Scrolling extends the window in either
 *  direction (one refetch with the grown window — payloads are small). */
class FeedStore {
	matches = $state<FeedMatch[]>([]);
	teams = $state<Record<string, Team>>({});
	deadlines = $state<FeedDeadline[]>([]);
	suggestions = $state<FeedSuggestion[]>([]);
	playing = $state<FeedMatch['tournament'][]>([]);
	loaded = $state(false);
	loading = $state(false);

	past = $state(2);
	ahead = $state(7);

	async load() {
		if (this.loading) return;
		this.loading = true;
		try {
			await serverClock.refresh();
			const [r, s] = await Promise.all([
				pb.send(`/api/feed?past=${this.past}&days=${this.ahead}`, { method: 'GET' }),
				pb
					.send('/api/tournaments/suggestions', { method: 'GET' })
					.catch(() => ({ suggestions: [] }))
			]);
			this.matches = r.matches ?? [];
			this.playing = r.playing ?? [];
			this.deadlines = (r.deadlines ?? []).filter((d: FeedDeadline) => {
				const ms = new Date(d.locksAt).getTime() - serverClock.now();
				return ms > 0 && ms < DEADLINE_LEAD_DAYS * 86400_000;
			});
			const tmap: Record<string, Team> = {};
			for (const [id, t] of Object.entries(r.teams ?? {})) {
				const tt = t as Record<string, string>;
				tmap[id] = { id, name: tt.name, iso2: tt.iso2, fifaCode: tt.fifaCode };
			}
			this.teams = tmap;
			this.suggestions = s.suggestions ?? [];
			this.loaded = true;
		} finally {
			this.loading = false;
		}
	}

	/** Extend the window backwards (older results) and refetch. */
	async earlier(days = 5) {
		this.past = Math.min(this.past + days, 30);
		await this.load();
	}

	/** Extend the window forwards (further fixtures) and refetch. */
	async later(days = 7) {
		this.ahead = Math.min(this.ahead + days, 30);
		await this.load();
	}

	team(id: string): Team | undefined {
		return this.teams[id];
	}

	/** Whether a feed match is finished. */
	finished(m: FeedMatch): boolean {
		return m.status === 'finished' || !!m.finalizedAt;
	}

	/** Matches grouped by the user's LOCAL day, oldest first. */
	get days(): FeedDay[] {
		const todayKey = localDayKey(new Date(serverClock.now()).toISOString());
		const byDay = new Map<string, FeedMatch[]>();
		for (const m of this.matches) {
			const k = localDayKey(m.kickoff);
			if (!byDay.has(k)) byDay.set(k, []);
			byDay.get(k)!.push(m);
		}
		const out: FeedDay[] = [];
		for (const [key, matches] of byDay) {
			const d = new Date(matches[0].kickoff);
			out.push({
				key,
				isToday: key === todayKey,
				label: d.toLocaleDateString(undefined, {
					weekday: 'short',
					day: 'numeric',
					month: 'short'
				}),
				matches
			});
		}
		out.sort((a, b) => (a.key < b.key ? -1 : 1));
		return out;
	}

	/** True when today has no section (no matches today). */
	get todayKey(): string {
		return localDayKey(new Date(serverClock.now()).toISOString());
	}

	/** Save (create or update) a tip for a feed match and mirror it back
	 *  into the row. Server hooks validate the lock + derive the advancer
	 *  and auto-subscribe the user to the tournament. */
	async saveTip(m: FeedMatch, t: Omit<Tip, 'id' | 'match'>): Promise<void> {
		const data = {
			user: auth.user?.id,
			match: m.id,
			ftHome: t.ftHome,
			ftAway: t.ftAway,
			etHome: t.etHome,
			etAway: t.etAway,
			penWinner: t.penWinner || null
		};
		const existing = await pb
			.collection('tips')
			.getFirstListItem(`user = "${auth.user?.id}" && match = "${m.id}"`)
			.catch(() => null);
		const rec = existing
			? await pb.collection('tips').update(existing.id, data)
			: await pb.collection('tips').create(data);
		const i = this.matches.findIndex((x) => x.id === m.id);
		if (i >= 0) {
			this.matches[i] = {
				...this.matches[i],
				myTip: {
					ftHome: rec.ftHome,
					ftAway: rec.ftAway,
					etHome: rec.etHome,
					etAway: rec.etAway,
					penWinner: rec.penWinner,
					advancer: rec.advancer
				}
			};
		}
	}

	/** Play a tournament from a suggestion card, then refresh. */
	async play(slug: string) {
		await pb.send(`/api/tournaments/${slug}/play`, { method: 'POST' });
		await this.load();
	}
}

export const feedStore = new FeedStore();
