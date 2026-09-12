/** The two-legged-tie strip: one line under a row or hero saying where the
 *  tie stands. Shared by MatchRow and the match page. */
import { legScore, type Match, type OtherLeg } from './tips.svelte';

export interface StripPart {
	t: string;
	/** Team name (bold). */
	b?: boolean;
	/** Score (mono). */
	n?: boolean;
}

export function tieStrip(o: {
	match: Match;
	leg: OtherLeg;
	played: boolean;
	live: boolean;
	homeName: string;
	awayName: string;
	teamName: (id: string) => string;
}): StripPart[] {
	const { match, leg: l, played, live } = o;
	const otherOrd = l.first ? '2nd' : '1st';
	const legDay = new Date(l.kickoff).toLocaleDateString(undefined, {
		weekday: 'short',
		day: 'numeric',
		month: 'short'
	});
	const leadOf = (h: number, a: number): StripPart[] =>
		h === a ? [{ t: 'level' }] : [{ t: h > a ? o.homeName : o.awayName, b: true }, { t: ' lead' }];
	if (!l.played) {
		const parts: StripPart[] = [{ t: `${otherOrd} leg ` }, { t: legDay, b: true }];
		if (played) {
			const [h, a] = legScore(match);
			parts.push({ t: ' · ' }, ...leadOf(h, a), { t: ' ' }, { t: `${h}–${a}`, n: true });
		}
		return parts;
	}
	if (!played && !live) {
		return [
			{ t: `${otherOrd} leg ` },
			{ t: `${l.forHome}–${l.forAway}`, n: true },
			{ t: ' · ' },
			...leadOf(l.forHome, l.forAway)
		];
	}
	const [h, a] = legScore(match);
	const agg: [number, number] = [h + l.forHome, a + l.forAway];
	const parts: StripPart[] = [
		{ t: 'Agg ' },
		{ t: `${agg[0]}–${agg[1]}`, n: true },
		{ t: ` · ${otherOrd} leg ` },
		{ t: `${l.forHome}–${l.forAway}`, n: true }
	];
	if (played && match.advancer) {
		const how =
			match.penHome || match.penAway
				? 'on penalties'
				: match.etHome || match.etAway
					? 'after extra time'
					: 'on aggregate';
		parts.push({ t: ' · ' }, { t: o.teamName(match.advancer), b: true }, { t: ` advance ${how}` });
	} else if (played) {
		parts.push({ t: ' · ' }, ...leadOf(agg[0], agg[1]));
	} else if (match.etHome || match.etAway) {
		parts.push({ t: ' · extra time' });
	}
	return parts;
}
