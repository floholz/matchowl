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
export interface Structure {
	stages: Stage[];
	groupSize?: number;
	gamesPerTeam?: number;
	directQualifiers?: number;
	extraQualifiers?: ExtraQualifiers | null;
	pointsWin?: number;
	pointsDraw?: number;
}

export interface Tournament {
	id: string;
	slug: string;
	name: string;
	shortName: string;
	status: 'draft' | 'upcoming' | 'active' | 'finished' | 'archived';
	startsAt: string;
	endsAt: string;
	structure: Structure;
}

/** Loads the tournament list once and exposes the current tournament (the
 *  server picks it: active > next upcoming > latest finished/archived) plus
 *  structure-derived helpers that replace the old hardcoded stage maps. */
class TournamentStore {
	list = $state<Tournament[]>([]);
	current = $state<Tournament | null>(null);
	loaded = $state(false);
	private inflight: Promise<void> | null = null;

	/** Idempotent: concurrent callers share one fetch. */
	ready(): Promise<void> {
		if (this.loaded) return Promise.resolve();
		if (!this.inflight) {
			this.inflight = pb
				.send('/api/tournaments', { method: 'GET' })
				.then((r) => {
					this.list = r.tournaments ?? [];
					this.current =
						this.list.find((t) => t.id === r.current) ?? this.list[0] ?? null;
					this.loaded = true;
				})
				.finally(() => (this.inflight = null));
		}
		return this.inflight;
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

	get directQualifiers(): number {
		return this.structure.directQualifiers ?? 2;
	}

	get pointsWin(): number {
		return this.structure.pointsWin ?? 3;
	}

	get pointsDraw(): number {
		return this.structure.pointsDraw ?? 1;
	}
}

export const tournamentStore = new TournamentStore();
