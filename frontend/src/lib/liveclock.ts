// The minute of a live match. The server stores what the provider said at
// the last sync (phase, minute, stoppage, read at liveAt); between syncs we
// count on from there, held at each phase's end (45+n', 90+n', 120+n') so a
// slow sync never runs the clock into the next half.

/** The live fields a match carries (see migrations/0053_match_clock.go). */
export interface LiveClock {
	kickoff: string;
	livePhase?: string;
	liveMinute?: number;
	liveExtra?: number;
	liveAt?: string;
}

/** Where each running phase ends, and how far past it we keep counting
 *  without a fresh reading. */
const PHASE_END: Record<string, number> = { '1H': 45, '2H': 90, ET: 120 };
const MAX_STOPPAGE = 15;

/** The clock text for a live match: "67'", "45+2'", "HT", "Pens" — or ''
 *  when there is nothing sensible to show. */
export function liveMinute(m: LiveClock, now: number): string {
	switch (m.livePhase) {
		case 'HT':
			return 'HT';
		case 'BT':
			return 'Break';
		case 'P':
			return 'Pens';
		case 'INT':
			return 'Paused';
	}
	const end = m.livePhase ? PHASE_END[m.livePhase] : undefined;
	if (end && m.liveAt) {
		const since = Math.max(0, Math.floor((now - new Date(m.liveAt).getTime()) / 60_000));
		const min = (m.liveMinute ?? 0) + (m.liveExtra ?? 0) + since;
		if (min <= end) return `${Math.max(1, min)}'`;
		return `${end}+${Math.min(min - end, MAX_STOPPAGE)}'`;
	}
	// No reading yet (just kicked off, or a source without a clock): the
	// first half can still be told from the kick-off; past it, no guess.
	if (!m.livePhase) {
		const min = Math.floor((now - new Date(m.kickoff).getTime()) / 60_000) + 1;
		if (min >= 1 && min <= 45) return `${min}'`;
	}
	return '';
}
