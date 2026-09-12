import { pb } from './pb';
import { auth } from './auth.svelte';
import { serverClock } from './serverclock.svelte';
import { tournamentStore } from './tournament.svelte';

export interface Team {
	id: string;
	name: string;
	iso2: string;
	fifaCode: string;
	/** Crest URL (served from our /api/files) — clubs, or nations without a
	 *  bundled flag. Empty when none. */
	logo: string;
}

/** URL of a team's crest file (PocketBase file field), or ''. */
export function teamLogoUrl(id: string, file: string | undefined): string {
	return file ? `/api/files/teams/${id}/${file}` : '';
}

export interface Match {
	id: string;
	stage: string; // a stage code from the tournament's structure
	groupLetter: string;
	roundLabel: string;
	num: number;
	kickoff: string;
	status: string;
	homeTeam: string;
	awayTeam: string;
	homeLabel: string;
	awayLabel: string;
	ftHome: number;
	ftAway: number;
	etHome: number;
	etAway: number;
	penHome: number;
	penAway: number;
	advancer: string;
	finalizedAt: string;
}

/** The raw fields of a tie's other leg (a subset of Match; the feed sends
 *  exactly these for legs outside its window). */
export interface LegSource {
	id: string;
	kickoff: string;
	status: string;
	finalizedAt: string;
	homeTeam: string;
	awayTeam: string;
	ftHome: number;
	ftAway: number;
	etHome: number;
	etAway: number;
}

/** What the TipCard needs to render a two-legged tie: which leg the card's
 *  match is, the other leg's result oriented to the card's sides, and a link
 *  to jump to the other leg. */
export interface OtherLeg {
	matchId: string;
	/** The card's match is the FIRST leg. */
	first: boolean;
	href: string;
	kickoff: string;
	played: boolean;
	/** Other-leg goals for the card's home/away team (sides flipped when the
	 *  return leg swaps home advantage). */
	forHome: number;
	forAway: number;
}

/** A leg's score for aggregate purposes: the cumulative after-120 score when
 *  it went to extra time (et 0/0 means "no ET"), else full time. */
export function legScore(l: {
	ftHome: number;
	ftAway: number;
	etHome: number;
	etAway: number;
}): [number, number] {
	return l.etHome !== 0 || l.etAway !== 0 ? [l.etHome, l.etAway] : [l.ftHome, l.ftAway];
}

/** Builds the card view of the other leg, orienting its score to the card
 *  match's home/away sides. */
export function otherLegView(
	m: { homeTeam: string; awayTeam: string },
	other: LegSource,
	first: boolean,
	href: string
): OtherLeg {
	const [oh, oa] = legScore(other);
	const flip = other.homeTeam === m.awayTeam;
	return {
		matchId: other.id,
		first,
		href,
		kickoff: other.kickoff,
		played: other.status === 'finished' || !!other.finalizedAt,
		forHome: flip ? oa : oh,
		forAway: flip ? oh : oa
	};
}

/** Finds the other leg of a two-legged tie in a full match list (same
 *  knockout stage, same two teams, exactly one partner). Caller ensures m is
 *  a knockout match. */
export function findOtherLeg(
	matches: Match[],
	m: Match
): { other: Match; first: boolean } | null {
	if (!m.homeTeam || !m.awayTeam) return null;
	const legs = matches.filter(
		(x) =>
			x.stage === m.stage &&
			((x.homeTeam === m.homeTeam && x.awayTeam === m.awayTeam) ||
				(x.homeTeam === m.awayTeam && x.awayTeam === m.homeTeam))
	);
	if (legs.length !== 2) return null;
	const other = legs.find((x) => x.id !== m.id);
	if (!other) return null;
	const first =
		m.kickoff < other.kickoff || (m.kickoff === other.kickoff && m.num < other.num);
	return { other, first };
}

export interface Tip {
	id?: string;
	match: string;
	ftHome: number;
	ftAway: number;
	etHome: number;
	etAway: number;
	penWinner: string;
	advancer: string;
}

export interface FriendTip {
	userId: string;
	name: string;
	ftHome: number;
	ftAway: number;
	etHome: number;
	etAway: number;
	penWinner: string;
	advancer: string;
	// Points this tip earned — only present once the match is finished.
	points?: number;
}

// Up to 10 names of players (across the whole app) who scored a perfect tip
// on this match — the maximum `points` (correct result + exact reference
// score). `count` is the true total when more than 10 nailed it.
export interface PerfectScorers {
	count: number;
	names: string[];
	points: number;
}

export interface FriendsResult {
	tips: FriendTip[];
	// Bot tips: global (not league-scoped), shown in their own list.
	bots: FriendTip[];
	perfect: PerfectScorers | null;
}

class TipsStore {
	teams = $state<Record<string, Team>>({});
	matches = $state<Match[]>([]);
	tips = $state<Record<string, Tip>>({}); // keyed by matchId
	scores = $state<Record<string, number>>({}); // matchId -> points (default cfg)
	/** Bumped after every saved tip (the server auto-subscribes you to the
	 *  tournament on the first one — pages showing Play re-read on this). */
	saved = $state(0);
	tournamentGroups = $state<Record<string, string[]>>({}); // letter -> teamIds
	loaded = $state(false);
	private subscribed = false;
	private loadedFor = '';

	async load() {
		await tournamentStore.ready();
		const tid = tournamentStore.current?.id ?? '';
		if (this.loaded && this.loadedFor === tid) return;
		this.loaded = false;
		const [teams, matches, mine, tgroups] = await Promise.all([
			pb.collection('teams').getFullList({ sort: 'name', filter: `tournament = "${tid}"` }),
			pb.collection('matches').getFullList({ sort: 'kickoff', filter: `tournament = "${tid}"` }),
			pb
				.collection('tips')
				.getFullList({ filter: `user = "${auth.user?.id}" && match.tournament = "${tid}"` }),
			pb.collection('tournament_groups').getFullList({ sort: 'letter', filter: `tournament = "${tid}"` }),
			serverClock.refresh(),
			pb
				.send('/api/tips/scores', { method: 'GET' })
				.then((r) => (this.scores = r.scores ?? {}))
				.catch(() => {})
		]);
		const gmap: Record<string, string[]> = {};
		for (const g of tgroups) gmap[g.letter] = g.teams ?? [];
		this.tournamentGroups = gmap;
		const tmap: Record<string, Team> = {};
		for (const t of teams)
			tmap[t.id] = {
				id: t.id,
				name: t.name,
				iso2: t.iso2,
				fifaCode: t.fifaCode,
				logo: teamLogoUrl(t.id, t.logo)
			};
		this.teams = tmap;
		this.matches = matches as unknown as Match[];
		const tip: Record<string, Tip> = {};
		for (const r of mine)
			tip[r.match] = {
				id: r.id,
				match: r.match,
				ftHome: r.ftHome,
				ftAway: r.ftAway,
				etHome: r.etHome,
				etAway: r.etAway,
				penWinner: r.penWinner,
				advancer: r.advancer
			};
		this.tips = tip;
		this.loaded = true;
		this.loadedFor = tid;
		this.subscribeLive();
	}

	/** The results sync writes score/status changes onto matches (including
	 *  in-play scores); mirror them into the store so open pages update
	 *  without a reload. When a match finalizes, points were just recomputed
	 *  server-side — refetch them so the FT pill shows them immediately. */
	private subscribeLive() {
		if (this.subscribed) return;
		this.subscribed = true;
		pb.collection('matches')
			.subscribe('*', (e) => {
				const i = this.matches.findIndex((m) => m.id === e.record.id);
				if (i < 0) return;
				const finalized = !this.matches[i].finalizedAt && !!e.record.finalizedAt;
				this.matches[i] = e.record as unknown as Match;
				if (finalized)
					pb.send('/api/tips/scores', { method: 'GET' })
						.then((r) => (this.scores = r.scores ?? {}))
						.catch(() => {});
			})
			.catch(() => {
				this.subscribed = false;
			});
	}

	team(id: string): Team | undefined {
		return this.teams[id];
	}

	/** Save (create or update) a tip; throws with the server message on a
	 *  rule/validation failure so the UI can show it. */
	async save(t: Tip): Promise<void> {
		const data = {
			user: auth.user?.id,
			match: t.match,
			ftHome: t.ftHome,
			ftAway: t.ftAway,
			etHome: t.etHome,
			etAway: t.etAway,
			penWinner: t.penWinner || null
		};
		let rec;
		if (t.id) {
			rec = await pb.collection('tips').update(t.id, data);
		} else {
			rec = await pb.collection('tips').create(data);
		}
		this.tips[t.match] = {
			id: rec.id,
			match: rec.match,
			ftHome: rec.ftHome,
			ftAway: rec.ftAway,
			etHome: rec.etHome,
			etAway: rec.etAway,
			penWinner: rec.penWinner,
			advancer: rec.advancer
		};
		this.saved++;
	}

	async friends(matchId: string): Promise<FriendsResult> {
		const r = await pb.send(`/api/tips/others/${matchId}`, {
			method: 'GET'
		});
		return { tips: r.tips ?? [], bots: r.bots ?? [], perfect: r.perfect ?? null };
	}
}

export const tipsStore = new TipsStore();

export function isLocked(m: Match): boolean {
	return serverClock.now() >= new Date(m.kickoff).getTime();
}
export function teamsResolved(m: Match): boolean {
	return !!m.homeTeam && !!m.awayTeam;
}
