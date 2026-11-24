import { pb } from './pb';
import { teamLogoUrl, type Team } from './tips.svelte';
import type { FeedMatch } from './feed.svelte';

/** A club or nation across seasons (a `team_identities` row), as
 *  /api/teams and /api/search show it. */
export interface TeamIdentity {
	id: string;
	/** The team page lives at /teams/{key}. */
	key: string;
	name: string;
	fifaCode: string;
	iso2: string;
	national: boolean;
	/** ISO code of the team's country (the nation itself for national teams). */
	country: string;
	countryName: string;
	/** "teams/<row>/<file>" under /api/files, or "". */
	crest: string;
}

/** A followed team's next (or live) match, for Home's strip. */
export interface NextMatch {
	id: string;
	kickoff: string;
	status: string;
	home: boolean;
	opponent: { teamId: string; key: string; name: string; fifaCode: string; iso2: string; logo: string };
	tournament: { id: string; name: string; shortName: string };
}

export interface TableLine {
	group: string;
	rank: number;
	size: number;
	played: number;
	won: number;
	drawn: number;
	lost: number;
	gf: number;
	ga: number;
	pts: number;
}

export interface TeamSeason {
	/** The season's `teams` row. */
	teamId: string;
	tournament: { id: string; slug: string; name: string; shortName: string; status: string; competition?: string };
	table: TableLine | null;
}

export interface TeamPage {
	team: TeamIdentity;
	following: boolean;
	followers: number;
	seasons: TeamSeason[];
}

export interface SearchResult {
	competitions: { key: string; name: string; shortName: string; country: string; logo: string }[];
	teams: TeamIdentity[];
}

export const crestUrl = (crest: string) => (crest ? `/api/files/${crest}` : '');

/** The matches of one team (/api/feed?team=), with its team map. */
export async function teamMatches(
	key: string,
	past: number,
	days: number
): Promise<{ matches: FeedMatch[]; teams: Record<string, Team> }> {
	const r = await pb.send(`/api/feed?team=${encodeURIComponent(key)}&past=${past}&days=${days}`, { method: 'GET' });
	const teams: Record<string, Team> = {};
	for (const [id, t] of Object.entries(r.teams ?? {})) {
		const tt = t as Record<string, string>;
		teams[id] = { id, name: tt.name, iso2: tt.iso2, fifaCode: tt.fifaCode, logo: teamLogoUrl(id, tt.logo), key: tt.key };
	}
	return { matches: (r.matches ?? []).map((m: FeedMatch) => ({ ...m, reasons: m.reasons ?? [] })), teams };
}

const RECENT_KEY = 'matchowl.recentSearches';
const RECENT_MAX = 8;

/** Followed teams (Home's strip, the follow stars) and recent searches. */
class TeamsStore {
	mine = $state<(TeamIdentity & { next: NextMatch | null })[]>([]);
	loaded = $state(false);
	/** Keys of the teams I follow, kept in step with follow/unfollow. */
	following = $state<Set<string>>(new Set());

	async load() {
		const r = await pb.send('/api/teams/mine', { method: 'GET' });
		this.mine = r.teams ?? [];
		this.following = new Set(this.mine.map((t) => t.key));
		this.loaded = true;
	}

	isFollowing(key: string): boolean {
		return this.following.has(key);
	}

	async setFollowing(key: string, on: boolean) {
		const next = new Set(this.following);
		if (on) next.add(key);
		else next.delete(key);
		this.following = next;
		try {
			await pb.send(`/api/teams/${encodeURIComponent(key)}/follow`, { method: on ? 'POST' : 'DELETE' });
		} catch (e) {
			const back = new Set(this.following);
			if (on) back.delete(key);
			else back.add(key);
			this.following = back;
			throw e;
		}
		// Home's strip and "my matches" follow the change.
		this.load().catch(() => {});
	}

	page(key: string): Promise<TeamPage> {
		return pb.send(`/api/teams/${encodeURIComponent(key)}`, { method: 'GET' });
	}

	search(q: string): Promise<SearchResult> {
		return pb.send(`/api/search?q=${encodeURIComponent(q)}`, { method: 'GET' });
	}

	// ---- recent searches (this device only) ----
	recent = $state<string[]>(readRecent());

	remember(q: string) {
		const s = q.trim();
		if (s.length < 2) return;
		this.recent = [s, ...this.recent.filter((x) => x.toLowerCase() !== s.toLowerCase())].slice(0, RECENT_MAX);
		try {
			localStorage.setItem(RECENT_KEY, JSON.stringify(this.recent));
		} catch {
			/* private mode: recents just don't stick */
		}
	}

	clearRecent() {
		this.recent = [];
		try {
			localStorage.removeItem(RECENT_KEY);
		} catch {
			/* ignore */
		}
	}
}

function readRecent(): string[] {
	try {
		const v = JSON.parse(localStorage.getItem(RECENT_KEY) ?? '[]');
		return Array.isArray(v) ? v.filter((x) => typeof x === 'string') : [];
	} catch {
		return [];
	}
}

export const teamsStore = new TeamsStore();
