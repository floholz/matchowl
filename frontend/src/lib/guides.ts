// The guides themselves (played by lib/guide.svelte.ts): the words live
// here so they are easy to tune. Targets are data-guide names on the
// screens they explain; a step whose target is not on screen is skipped.
import { guide, type Guide } from './guide.svelte';
import { defaultScoring, type ScoringConfig } from './scoring';
import { api } from './api';

export const H2H_GUIDE = 'h2h';
export const TIP_GUIDE = 'tip';

/** Head-to-head, on the pool page: the duel, then the picks, then the table. */
export function h2hGuide(saveCalls: number): Guide {
	const n = saveCalls === 1 ? 'one' : saveCalls === 2 ? 'two' : String(saveCalls);
	return {
		id: H2H_GUIDE,
		steps: [
			{
				title: 'Head-to-head',
				body: "This pool plays duels. Every matchday you face one pool mate, and whoever scores more tip points over that matchday's matches wins."
			},
			{
				target: 'h2h-strip',
				title: 'Your matchdays',
				body: 'One card per matchday: your result for the ones played, the live score while one is in play, and who you face next. Tap a card to see its duels.'
			},
			{
				target: 'h2h-duels',
				title: 'Every duel',
				body: 'All the pairings of the matchday. With an odd number of players, one of you plays the Ghost — it scores the average of everyone else.'
			},
			{
				target: 'h2h-save',
				title: 'Save Call',
				body: `Your banker: a Save Call makes a match count double for you. You get ${n} per matchday.`
			},
			{
				target: 'h2h-ban',
				title: 'Ban',
				body: "Take one match away from your rival: their points on it don't count. If they made it a Save Call, your ban only cancels the double."
			},
			{
				target: 'h2h-calls',
				title: 'Hidden until kick-off',
				body: 'Your rival only sees your Save Calls and your ban once that match kicks off. Until then you can move them — each one locks at its own kick-off.'
			},
			{
				target: 'h2h-table',
				title: 'The table',
				body: 'A won duel is worth 3 points, a draw 1. Level on points? The better tipper overall goes ahead.'
			}
		]
	};
}

/** Tipping, in a tip editor the first time one opens. The duel step only
 *  when the match counts in one of the member's head-to-head pools. */
export function tipGuide(scoring: ScoringConfig | null, inH2H: boolean): Guide {
	let points =
		'A correct result earns points, and the closer your tip, the more — an exact score collects the most. Help has the full table.';
	if (scoring) {
		const s = scoring.match;
		const extra = [
			s.goalDiff ? `+${s.goalDiff} for the goal difference` : '',
			s.totalGoals ? `+${s.totalGoals} for the total goals` : '',
			s.exact ? `+${s.exact} for the exact score` : ''
		].filter(Boolean);
		points = `The right result (win, draw or loss) earns ${s.tendency}.`;
		if (extra.length) points += ` On top: ${extra.join(', ')} — so an exact tip collects them all.`;
	}
	return {
		id: TIP_GUIDE,
		steps: [
			{
				target: 'tip-enter',
				title: 'Your tip',
				body: 'Set the score with the steppers and tap Save. You can change it until kick-off — then it locks.'
			},
			{ title: 'Points', body: points },
			...(inH2H
				? [
						{
							target: 'tip-h2h',
							title: 'Your duels',
							body: 'This match counts in your head-to-head pool: make it a Save Call or ban it for your rival right here.'
						}
					]
				: []),
			{
				title: 'After kick-off',
				body: "Open a match once it has started to see what your friends tipped — and later, how many points each tip earned."
			}
		]
	};
}

/** Offer the tipping guide for a tip editor that just opened (a no-op
 *  once the account has been through it). */
export async function offerTipGuide(matchId: string) {
	if (guide.seen(TIP_GUIDE)) return;
	const [scoring, h2h] = await Promise.all([
		defaultScoring(),
		api.h2hMatch(matchId).catch(() => ({ pools: [] }))
	]);
	guide.offer(tipGuide(scoring, h2h.pools.length > 0));
}
