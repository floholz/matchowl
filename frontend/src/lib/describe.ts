import type { Tournament } from './tournament.svelte';

/** One-line "what is this" for a season, derived from its structure:
 *    "48 national teams · 12 groups of 4, then knockout from the round of 32 · 11 Jun – 19 Jul 2026"
 *    "18 clubs · 34 matchdays · Aug 2025 – May 2026"
 *  Used when the competition has no admin-written description, and as
 *  the placeholder in the admin editor. `teams` may be 0 (unknown). */
export function describeSeason(t: Tournament, teams = 0): string {
	const st = t.structure ?? { stages: [] };
	const stages = st.stages ?? [];
	const group = stages.find((s) => s.kind === 'group');
	const ko = stages.filter((s) => s.kind === 'knockout' && !s.consolation);
	const groupSize = st.groupSize ?? 4;
	const league = !!group && groupSize > 8;
	const kind = t.competition?.teamKind === 'club' ? 'clubs' : 'national teams';

	const parts: string[] = [];
	if (teams > 0) parts.push(`${teams} ${kind}`);

	if (league) {
		const md = st.gamesPerTeam ?? (teams > 0 ? (teams - 1) * 2 : 0);
		parts.push(md > 0 ? `${md} matchdays` : 'round-robin');
		if (ko.length) parts.push(`then knockout from the ${ko[0].name.toLowerCase()}`);
	} else if (group) {
		const n = teams > 0 && groupSize > 0 ? Math.round(teams / groupSize) : 0;
		let g = n > 0 ? `${n} groups of ${groupSize}` : `groups of ${groupSize}`;
		if (ko.length) g += `, then knockout from the ${ko[0].name.toLowerCase()}`;
		parts.push(g);
	} else if (ko.length) {
		parts.push(`straight knockout from the ${ko[0].name.toLowerCase()}`);
	}

	const span = dateSpan(t.startsAt, t.endsAt);
	if (span) parts.push(span);
	return parts.join(' · ');
}

/** "11 Jun – 19 Jul 2026" within a year, "Aug 2025 – May 2026" across years. */
export function dateSpan(startsAt: string, endsAt: string): string {
	if (!startsAt) return '';
	const a = new Date(startsAt);
	if (!endsAt) return a.toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' });
	const b = new Date(endsAt);
	if (a.getFullYear() === b.getFullYear()) {
		const d = (x: Date) => x.toLocaleDateString(undefined, { day: 'numeric', month: 'short' });
		return `${d(a)} – ${d(b)} ${b.getFullYear()}`;
	}
	const my = (x: Date) => x.toLocaleDateString(undefined, { month: 'short', year: 'numeric' });
	return `${my(a)} – ${my(b)}`;
}
