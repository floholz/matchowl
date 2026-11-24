import { pb } from './pb';
import type { Person } from './api';

/** A tip as a challenge shows it. */
export interface ChallengeTip {
	ftHome: number;
	ftAway: number;
	etHome: number;
	etAway: number;
	penWinner: string;
}

export interface ChallengeTeam {
	teamId: string;
	name: string;
	fifaCode: string;
	iso2: string;
	logo: string;
}

/** One challenge from my side (/api/challenges). */
export interface Challenge {
	id: string;
	/** pending | accepted | declined | cancelled | expired (pending past kick-off). */
	status: string;
	stake: string;
	created: string;
	/** I sent it. */
	mine: boolean;
	them: Person;
	match: {
		id: string;
		kickoff: string;
		status: string;
		finalizedAt: string;
		ftHome: number;
		ftAway: number;
		home: ChallengeTeam;
		away: ChallengeTeam;
		tournament?: { id: string; name: string; shortName: string };
	};
	myTip: ChallengeTip | null;
	theirTipped: boolean;
	/** Only once the match has kicked off. */
	theirTip?: ChallengeTip | null;
	settled: boolean;
	myPts?: number;
	theirPts?: number;
	outcome?: 'won' | 'lost' | 'draw';
}

export interface ChallengeRecord {
	won: number;
	drawn: number;
	lost: number;
}

/** "you lead 4–2", "Max leads 1–0", "level 1–1", "" (none yet). */
export function recordLine(r: ChallengeRecord | undefined, name: string): string {
	if (!r || r.won + r.drawn + r.lost === 0) return '';
	const draws = r.drawn ? ` · ${r.drawn} draw${r.drawn === 1 ? '' : 's'}` : '';
	if (r.won > r.lost) return `you lead ${r.won}–${r.lost}${draws}`;
	if (r.lost > r.won) return `${name} leads ${r.lost}–${r.won}${draws}`;
	return `level ${r.won}–${r.lost}${draws}`;
}

export const teamLogo = (t: ChallengeTeam) => (t.logo && t.teamId ? `/api/files/teams/${t.teamId}/${t.logo}` : '');

/** Incoming, open and recent challenges plus the record per friend. */
class ChallengesStore {
	incoming = $state<Challenge[]>([]);
	open = $state<Challenge[]>([]);
	recent = $state<Challenge[]>([]);
	records = $state<Record<string, ChallengeRecord>>({});
	loaded = $state(false);

	async load() {
		const [r, rec] = await Promise.all([
			pb.send('/api/challenges', { method: 'GET' }),
			pb.send('/api/challenges/records', { method: 'GET' })
		]);
		this.incoming = r.incoming ?? [];
		this.open = r.open ?? [];
		this.recent = r.recent ?? [];
		this.records = rec.records ?? {};
		this.loaded = true;
	}

	forMatch(matchId: string): Promise<Challenge[]> {
		return pb
			.send(`/api/challenges/match/${encodeURIComponent(matchId)}`, { method: 'GET' })
			.then((r) => r.challenges ?? []);
	}

	async send(match: string, opponent: string, stake: string): Promise<Challenge> {
		const c = await pb.send('/api/challenges', { method: 'POST', body: { match, opponent, stake } });
		this.load().catch(() => {});
		return c;
	}

	async answer(id: string, action: 'accept' | 'decline' | 'cancel'): Promise<Challenge> {
		const c = await pb.send(`/api/challenges/${encodeURIComponent(id)}/${action}`, { method: 'POST' });
		this.load().catch(() => {});
		return c;
	}
}

export const challengesStore = new ChallengesStore();
