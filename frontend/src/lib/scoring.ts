/** Client mirror of the server's tip scorer (internal/scoring: scoreValues)
 *  plus the default config, so the match page can explain a tip's points.
 *  Keep in step with the Go rules. */
import { pb } from './pb';

export interface ScoringConfig {
	match: { tendency: number; exact: number; totalGoals: number; goalDiff: number };
	forecast: {
		groupPosition: number;
		perfectGroupBonus: number;
		advance: number;
		round: Record<string, number>;
	};
}

let cached: Promise<ScoringConfig | null> | null = null;
/** The default scoring config (cached for the session; null when none). */
export function defaultScoring(): Promise<ScoringConfig | null> {
	if (!cached)
		cached = pb
			.send('/api/scoring/default', { method: 'GET' })
			.then((r) => r as ScoringConfig)
			.catch(() => null);
	return cached;
}

export interface MatchResultLike {
	ftHome: number;
	ftAway: number;
	etHome: number;
	etAway: number;
	advancer: string;
}
export interface TipLike {
	ftHome: number;
	ftAway: number;
	etHome: number;
	etAway: number;
	advancer: string;
}
export interface TipPoints {
	tendency: number;
	exact: number;
	totalGoals: number;
	goalDiff: number;
	total: number;
}

const sign = (n: number) => (n > 0 ? 1 : n < 0 ? -1 : 0);

/** Points a tip earns on a finished match (knockout = phased tie/final:
 *  tendency is the advancer; the reference score is after ET when it went
 *  there). First legs pass knockout=false. */
export function scoreTip(
	cfg: ScoringConfig,
	knockout: boolean,
	m: MatchResultLike,
	p: TipLike
): TipPoints {
	let aH = m.ftHome,
		aA = m.ftAway,
		pH = p.ftHome,
		pA = p.ftAway;
	if (knockout && (m.etHome !== 0 || m.etAway !== 0)) {
		aH = m.etHome;
		aA = m.etAway;
		if (p.ftHome === p.ftAway) {
			pH = p.etHome;
			pA = p.etAway;
		}
	}
	const r: TipPoints = { tendency: 0, exact: 0, totalGoals: 0, goalDiff: 0, total: 0 };
	if (!knockout) {
		if (sign(p.ftHome - p.ftAway) === sign(m.ftHome - m.ftAway)) r.tendency = cfg.match.tendency;
	} else if (m.advancer) {
		if (m.advancer === p.advancer) r.tendency = cfg.match.tendency;
	} else if (sign(p.ftHome - p.ftAway) === sign(m.ftHome - m.ftAway)) {
		r.tendency = cfg.match.tendency;
	}
	if (pH === aH && pA === aA) r.exact = cfg.match.exact;
	if (pH + pA === aH + aA) r.totalGoals = cfg.match.totalGoals;
	if (pH - pA === aH - aA) r.goalDiff = cfg.match.goalDiff;
	r.total = r.tendency + r.exact + r.totalGoals + r.goalDiff;
	return r;
}
