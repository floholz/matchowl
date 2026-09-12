import { browser } from '$app/environment';
import { pb } from './pb';

/** One phase of a tournament, in play order (mirrors the Go structure). */
export interface Stage {
	code: string;
	name: string;
	kind: 'group' | 'knockout';
	consolation?: boolean;
}

export interface ExtraQualifiers {
	fromPosition: number;
	count: number;
	tableKey?: string;
}

/** The tournament's competition shape — stages, group size, qualifier rules.
 *  Everything the UI used to hardcode for WC2026 comes from here. */
/** Named position range of the final table (league shapes). */
export interface Zone {
	key: string;
	name: string;
	from: number;
	to: number;
}

export interface Structure {
	stages: Stage[];
	groupSize?: number;
	gamesPerTeam?: number;
	directQualifiers?: number;
	extraQualifiers?: ExtraQualifiers | null;
	zones?: Zone[];
	pointsWin?: number;
	pointsDraw?: number;
}

/** A competition (FIFA World Cup, Bundesliga, …); a tournament is one
 *  season of it. Mirrors CompetitionView on the server. */
export interface Competition {
	id: string;
	key: string;
	name: string;
	shortName: string;
	country: string;
	teamKind: 'national' | 'club';
	logo: string; // filename, see competitionLogoUrl()
	description: string; // admin blurb; '' → derive from the season
	apiFootballLeague?: number;
}

export function competitionLogoUrl(c: Pick<Competition, 'id' | 'logo'> | null | undefined): string {
	return c?.logo ? `/api/files/competitions/${c.id}/${c.logo}` : '';
}

/** "UEFA Champions League 2025/26" → "2025/26": the season without the
 *  competition's name in front (the import names seasons that way). */
export function seasonLabel(t: Pick<Tournament, 'name' | 'shortName' | 'competition'>): string {
	const n = t.name.trim();
	const c = t.competition?.name?.trim() ?? '';
	if (c && n.toLowerCase().startsWith(c.toLowerCase())) {
		const rest = n.slice(c.length).replace(/^[\s·–-]+/, '');
		if (rest) return rest;
	}
	return t.shortName || n;
}

export type TournamentStatus = 'draft' | 'upcoming' | 'active' | 'finished' | 'archived';

export interface Tournament {
	id: string;
	slug: string;
	name: string;
	shortName: string;
	status: TournamentStatus;
	startsAt: string;
	endsAt: string;
	structure: Structure;
	competition: Competition;
	/** Only `mode` matters to the hub ('none' → no forecast card). */
	forecastSpec?: { mode?: string } | null;
}

const statusRank: Record<TournamentStatus, number> = {
	active: 0,
	upcoming: 1,
	finished: 2,
	archived: 3,
	draft: 4
};

/** The season to land on by default: a running one, else the next
 *  upcoming, else the most recent (mirrors the server's "current" pick). */
export function defaultSeason(seasons: Tournament[]): Tournament | null {
	let best: Tournament | null = null;
	for (const t of seasons) {
		if (!best) {
			best = t;
			continue;
		}
		const ra = statusRank[t.status] ?? 9;
		const rb = statusRank[best.status] ?? 9;
		if (ra !== rb) {
			if (ra < rb) best = t;
			continue;
		}
		const later = t.startsAt > best.startsAt;
		if (t.status === 'upcoming' ? !later : later) best = t;
	}
	return best;
}

/** Loads the tournament list once and exposes the current tournament (the
 *  server picks it: active > next upcoming > latest finished/archived) plus
 *  structure-derived helpers that replace the old hardcoded stage maps. */
class TournamentStore {
	list = $state<Tournament[]>([]);
	current = $state<Tournament | null>(null);
	loaded = $state(false);
	private inflight: Promise<void> | null = null;

	/** Slug from ?t= that isn't in the list (draft for non-admins, deleted,
	 *  or a typo). Pages show a notice instead of silently falling back. */
	missing = $state('');

	/** Idempotent: concurrent callers share one fetch. */
	ready(): Promise<void> {
		if (this.loaded) return Promise.resolve();
		return this.reload();
	}

	/** Refetch the list (e.g. a tournament was created after the SPA
	 *  loaded). Keeps the current selection when it still exists. */
	reload(): Promise<void> {
		if (!this.inflight) {
			this.inflight = pb
				.send('/api/tournaments', { method: 'GET' })
				.then((r) => {
					this.list = r.tournaments ?? [];
					const keep = this.current && this.list.find((t) => t.id === this.current?.id);
					this.current =
						keep ?? this.list.find((t) => t.id === r.current) ?? this.list[0] ?? null;
					this.loaded = true;
				})
				.finally(() => (this.inflight = null));
		}
		return this.inflight;
	}

	/** Seasons of a competition, newest first. */
	seasonsOf(key: string): Tournament[] {
		return this.list
			.filter((t) => t.competition?.key === key)
			.sort((a, b) => (a.startsAt < b.startsAt ? 1 : -1));
	}

	/** Point the app at a specific tournament (by slug) — used by the
	 *  ?t= param on the per-tournament detail pages. No-op for unknown
	 *  slugs. Returns whether the selection changed. */
	select(slug: string): boolean {
		const t = this.list.find((x) => x.slug === slug);
		if (!t || this.current?.id === t.id) return false;
		this.current = t;
		return true;
	}

	get structure(): Structure {
		return this.current?.structure ?? { stages: [] };
	}

	/** All stages in play order. */
	get stages(): Stage[] {
		return this.structure.stages ?? [];
	}

	get knockoutStages(): Stage[] {
		return this.stages.filter((s) => s.kind === 'knockout');
	}

	/** A league season is modelled as one big group ("A"). Hide the
	 *  "Group A" prefix for those and use the stage name ("League")
	 *  instead. Heuristic: WC/Euro-style groups are ≤ 8 teams. */
	get singleTable(): boolean {
		return this.groupStageCode !== '' && (this.structure.groupSize ?? 4) > 8;
	}

	/** Label for a group letter: "Group B", or the stage name for a
	 *  single-table season. */
	groupLabel(letter: string): string {
		return this.singleTable ? this.stageName(this.groupStageCode) : `Group ${letter}`;
	}

	/** The group stage's code ('' for group-less formats). */
	get groupStageCode(): string {
		return this.stages.find((s) => s.kind === 'group')?.code ?? '';
	}

	isKnockout(stageCode: string): boolean {
		return this.stages.find((s) => s.code === stageCode)?.kind === 'knockout';
	}

	isGroup(stageCode: string): boolean {
		return this.stages.find((s) => s.code === stageCode)?.kind === 'group';
	}

	/** Human label for a stage code (falls back to the code itself). */
	stageName(code: string): string {
		return this.stages.find((s) => s.code === code)?.name ?? code;
	}

	/** The stage whose winner is champion (last non-consolation knockout). */
	get championStageCode(): string {
		for (let i = this.stages.length - 1; i >= 0; i--) {
			const s = this.stages[i];
			if (s.kind === 'knockout' && !s.consolation) return s.code;
		}
		return '';
	}

	get extraQualifiers(): ExtraQualifiers | null {
		return this.structure.extraQualifiers ?? null;
	}

	get groupSize(): number {
		return this.structure.groupSize ?? 4;
	}

	get gamesPerTeam(): number {
		return this.structure.gamesPerTeam ?? Math.max(1, this.groupSize - 1);
	}

	/** Teams per group that advance directly (0 for league seasons, where
	 *  zones describe the table instead). */
	get directQualifiers(): number {
		return this.structure.directQualifiers ?? 0;
	}

	get zones(): Zone[] {
		return this.structure.zones ?? [];
	}

	/** The zone covering a 1-based table position, or null. */
	zoneAt(position: number): Zone | null {
		return this.zones.find((z) => position >= z.from && position <= z.to) ?? null;
	}

	/** Stable colour index (0..n) for a zone key — used to tint table rows
	 *  and the legend consistently. */
	zoneIndex(key: string): number {
		return this.zones.findIndex((z) => z.key === key);
	}

	get pointsWin(): number {
		return this.structure.pointsWin ?? 3;
	}

	get pointsDraw(): number {
		return this.structure.pointsDraw ?? 1;
	}
}

export const tournamentStore = new TournamentStore();

/** Honor a ?t=<slug> query param (the forecast page is reached from the
 *  competition hub with it). */
export async function selectFromUrl(): Promise<void> {
	await tournamentStore.ready();
	if (!browser) return;
	const t = new URLSearchParams(location.search).get('t');
	tournamentStore.missing = '';
	if (!t) return;
	if (tournamentStore.list.some((x) => x.slug === t)) {
		tournamentStore.select(t);
		return;
	}
	// Not in the cached list — it may have been created/published since
	// the SPA loaded. Refetch once before giving up.
	await tournamentStore.reload();
	if (!tournamentStore.select(t) && !tournamentStore.list.some((x) => x.slug === t)) {
		tournamentStore.missing = t;
	}
}
