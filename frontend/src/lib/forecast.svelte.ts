import { pb } from './pb';
import { auth } from './auth.svelte';
import type { Team } from './tips.svelte';
import { tournamentStore, type Structure } from './tournament.svelte';

/** One headline pick of a calls-mode forecast (mirrors the Go spec). */
export interface ForecastCall {
	key: string;
	name: string;
	type: 'team' | 'teamset';
	points: number;
	zone?: string;
	stage?: string;
	count?: number;
}

export interface ForecastSpec {
	mode: 'full' | 'calls' | 'none';
	calls?: ForecastCall[];
}

export interface KOMatch {
	num: number;
	stage: string;
	round: string;
	homeLabel: string;
	awayLabel: string;
}
export interface ThirdSlot {
	matchNum: number;
	winner: string;
	allowed: string[];
}
export interface GroupDef {
	letter: string;
	teams: string[];
}

/** Stable key for a KO match: its number, or the stage for the number-less
 *  Final / third-place matches. */
export function koKey(m: { num: number; stage: string }): string {
	return m.num > 0 ? String(m.num) : m.stage;
}

export class ForecastStore {
	loaded = $state(false);
	locked = $state(false);
	tournamentStart = $state<string>('');
	structure = $state<Structure>({ stages: [] });
	spec = $state<ForecastSpec>({ mode: 'full' });
	teams = $state<Record<string, Team>>({});
	groups = $state<GroupDef[]>([]);
	knockout = $state<KOMatch[]>([]);
	thirdTable: Record<string, Record<string, string>> = {};
	thirdSlots = $state<ThirdSlot[]>([]);

	// Editable forecast.
	recId: string | undefined;
	readOnly = $state(false); // true when viewing a friend's forecast
	viewName = $state(''); // friend's display name (read-only mode)
	groupOrder = $state<Record<string, string[]>>({}); // letter -> [id x4]
	thirds = $state<Record<string, string>>({}); // matchNum -> teamId
	bracket = $state<Record<string, string>>({}); // koKey -> winner teamId
	calls = $state<Record<string, string | string[]>>({}); // callKey -> pick(s)

	// Actual results, for post-stage correctness indicators.
	results = $state<
		{
			stage: string;
			groupLetter: string;
			num: number;
			homeTeam: string;
			awayTeam: string;
			ftHome: number;
			ftAway: number;
			advancer: string;
			finished: boolean;
		}[]
	>([]);

	// Loads structure/teams/results (shared by the editor and the read-only
	// friend viewer).
	private async loadBase() {
		await tournamentStore.ready();
		const tid = tournamentStore.current?.id ?? '';
		const [structure, teams, matches] = await Promise.all([
			pb.send('/api/forecast/structure', { method: 'GET' }),
			pb.collection('teams').getFullList({ sort: 'name', filter: `tournament = "${tid}"` }),
			pb.collection('matches').getFullList({ sort: 'kickoff', filter: `tournament = "${tid}"` })
		]);
		this.structure = structure.structure ?? tournamentStore.structure;
		this.spec = structure.forecastSpec ?? { mode: 'full' };
		this.results = (matches as unknown[]).map((m) => {
			const r = m as Record<string, unknown>;
			return {
				stage: r.stage as string,
				groupLetter: r.groupLetter as string,
				num: r.num as number,
				homeTeam: r.homeTeam as string,
				awayTeam: r.awayTeam as string,
				ftHome: r.ftHome as number,
				ftAway: r.ftAway as number,
				advancer: r.advancer as string,
				finished:
					r.status === 'finished' || !!(r.finalizedAt as string)
			};
		});
		const tmap: Record<string, Team> = {};
		for (const t of teams)
			tmap[t.id] = {
				id: t.id,
				name: t.name,
				iso2: t.iso2,
				fifaCode: t.fifaCode
			};
		this.teams = tmap;
		this.groups = structure.groups;
		this.knockout = structure.knockout;
		this.thirdSlots = structure.thirdSlots ?? [];
		this.thirdTable = structure.thirdTable ?? {};
		this.tournamentStart = structure.tournamentStart;
		this.locked = structure.locked;
	}

	// Sets the editable prediction from a forecast-like record (or undefined),
	// defaulting each group's order to its team list.
	private applyForecast(f?: {
		groupOrder?: Record<string, string[]>;
		thirdQualifiers?: Record<string, string>;
		bracket?: Record<string, string>;
		calls?: Record<string, string | string[]>;
	}) {
		const order: Record<string, string[]> = {};
		for (const g of this.groups)
			order[g.letter] = f?.groupOrder?.[g.letter]?.length
				? [...f.groupOrder[g.letter]]
				: [...g.teams];
		this.groupOrder = order;
		this.thirds = f?.thirdQualifiers ?? {};
		this.bracket = f?.bracket ?? {};
		this.calls = f?.calls ?? {};
	}

	private loadedFor = '';

	async load() {
		await tournamentStore.ready();
		const wantTid = tournamentStore.current?.id ?? '';
		if (this.loaded && !this.readOnly && this.loadedFor === wantTid) return;
		this.loaded = false;
		await this.loadBase();
		const tid = tournamentStore.current?.id ?? '';
		this.loadedFor = tid;
		const mine = await pb
			.collection('forecasts')
			.getFullList({ filter: `user = "${auth.user?.id}" && tournament = "${tid}"` });
		this.recId = mine[0]?.id;
		this.readOnly = false;
		this.applyForecast(mine[0] as never);
		this.loaded = true;
	}

	// Read-only: load a friend's forecast (shared-league gated server-side).
	async loadView(userId: string) {
		await this.loadBase();
		const r = await pb.send(`/api/forecast/of/${userId}`, {
			method: 'GET'
		});
		this.readOnly = true;
		this.viewName = r.name ?? '';
		this.recId = undefined;
		this.applyForecast(r.forecast ?? undefined);
		this.loaded = true;
	}

	team(id: string) {
		return this.teams[id];
	}

	/** The group stage's code per the structure ('group' for WC2026). */
	get groupStageCode(): string {
		return this.structure.stages.find((s) => s.kind === 'group')?.code ?? 'group';
	}

	get extraQualifiers() {
		return this.structure.extraQualifiers ?? null;
	}

	get groupSize(): number {
		return this.structure.groupSize ?? 4;
	}

	get gamesPerTeam(): number {
		return this.structure.gamesPerTeam ?? Math.max(1, this.groupSize - 1);
	}

	/** True once every group match is finished. */
	get groupStageDone(): boolean {
		const g = this.results.filter((r) => r.stage === this.groupStageCode);
		return g.length > 0 && g.every((r) => r.finished);
	}

	// Standings (pts, gd, gf) for one group's finished matches.
	private standing(letter: string) {
		const t: Record<
			string,
			{ id: string; pts: number; gd: number; gf: number; p: number }
		> = {};
		for (const id of this.groups.find((x) => x.letter === letter)?.teams ??
			[])
			t[id] = { id, pts: 0, gd: 0, gf: 0, p: 0 };
		for (const m of this.results) {
			if (m.stage !== this.groupStageCode || m.groupLetter !== letter || !m.finished)
				continue;
			const H = t[m.homeTeam],
				A = t[m.awayTeam];
			if (!H || !A) continue;
			H.p++;
			A.p++;
			H.gf += m.ftHome;
			A.gf += m.ftAway;
			H.gd += m.ftHome - m.ftAway;
			A.gd += m.ftAway - m.ftHome;
			const w = this.structure.pointsWin ?? 3;
			const d = this.structure.pointsDraw ?? 1;
			if (m.ftHome > m.ftAway) H.pts += w;
			else if (m.ftHome < m.ftAway) A.pts += w;
			else {
				H.pts += d;
				A.pts += d;
			}
		}
		return Object.values(t);
	}

	/** Actual final ordering of a group, or null until it's complete. */
	actualOrder(letter: string): string[] | null {
		const rows = this.standing(letter);
		if (rows.length < this.groupSize || rows.some((r) => r.p < this.gamesPerTeam))
			return null;
		rows.sort((a, b) => b.pts - a.pts || b.gd - a.gd || b.gf - a.gf);
		return rows.map((r) => r.id);
	}

	/** The teams that actually qualify as extra qualifiers (WC2026: the 8 best
	 *  thirds), or null until the whole group stage is done / for tournaments
	 *  without extra qualifiers. */
	actualBestThirds(): Set<string> | null {
		const eq = this.extraQualifiers;
		if (!eq || !this.groupStageDone) return null;
		const thirds: { id: string; pts: number; gd: number; gf: number }[] =
			[];
		for (const g of this.groups) {
			const rows = this.standing(g.letter).sort(
				(a, b) => b.pts - a.pts || b.gd - a.gd || b.gf - a.gf
			);
			if (rows[eq.fromPosition - 1]) thirds.push(rows[eq.fromPosition - 1]);
		}
		thirds.sort((a, b) => b.pts - a.pts || b.gd - a.gd || b.gf - a.gf);
		return new Set(thirds.slice(0, eq.count).map((t) => t.id));
	}

	/** Actual advancer of a knockout match number, '' if not finished. */
	advancerOf(num: number): string {
		const m = this.results.find((r) => r.num === num);
		return m && m.finished ? m.advancer : '';
	}

	move(letter: string, idx: number, dir: -1 | 1) {
		const arr = [...this.groupOrder[letter]];
		const j = idx + dir;
		if (j < 0 || j >= arr.length) return;
		[arr[idx], arr[j]] = [arr[j], arr[idx]];
		this.groupOrder[letter] = arr;
	}

	/** True when a label is an extra-qualifier slot ("3A/B/C/D/F"). */
	private isSlotLabel(label: string): boolean {
		const eq = this.extraQualifiers;
		return (
			!!eq &&
			label.startsWith(String(eq.fromPosition)) &&
			label.includes('/')
		);
	}

	/** Resolve a placeholder label ("1A","2B","3A/B/..","W74","L101") to a
	 *  team id given the current predictions, or '' if undecidable. The digit
	 *  is a group position per the tournament structure. */
	resolve(label: string, forMatchNum: number, seen = new Set<number>()): string {
		if (!label) return '';
		const c = label[0];
		if (c >= '1' && c <= '9') {
			if (this.isSlotLabel(label))
				return this.thirdAssignment()[forMatchNum] ?? '';
			const letter = label.slice(1);
			return this.groupOrder[letter]?.[Number(c) - 1] ?? '';
		}
		if (c === 'W' || c === 'L') {
			const n = parseInt(label.slice(1), 10);
			if (seen.has(n)) return '';
			seen.add(n);
			const w = this.bracket[String(n)] ?? '';
			if (c === 'W') return w;
			const src = this.knockout.find((m) => m.num === n);
			if (!src || !w) return '';
			const h = this.resolve(src.homeLabel, n, seen);
			const a = this.resolve(src.awayLabel, n, seen);
			return w === h ? a : w === a ? h : '';
		}
		return '';
	}

	sides(m: KOMatch): [string, string] {
		return [
			this.resolve(m.homeLabel, m.num),
			this.resolve(m.awayLabel, m.num)
		];
	}

	pick(m: KOMatch, teamId: string) {
		if (!teamId) return;
		this.bracket[koKey(m)] = teamId;
	}

	/** How many extra qualifiers the user must tick (0 = feature disabled). */
	get maxThirds(): number {
		return this.extraQualifiers?.count ?? 0;
	}

	/** The predicted extra-qualifier-position team of a group (from the
	 *  current order). */
	groupThird(letter: string): string {
		const pos = this.extraQualifiers?.fromPosition ?? 3;
		return this.groupOrder[letter]?.[pos - 1] ?? '';
	}

	/** Letters the user ticked to advance as a best third. */
	get chosenThirdLetters(): string[] {
		return Object.keys(this.thirds);
	}

	toggleThird(letter: string) {
		if (this.thirds[letter]) {
			delete this.thirds[letter];
			this.thirds = { ...this.thirds };
		} else if (this.chosenThirdLetters.length < this.maxThirds) {
			this.thirds = { ...this.thirds, [letter]: this.groupThird(letter) };
		}
	}

	/** Toggle a team inside a call's pick (single for team calls, capped
	 *  multi for teamsets). */
	toggleCall(call: ForecastCall, teamId: string) {
		if (call.type === 'team') {
			this.calls = {
				...this.calls,
				[call.key]: this.calls[call.key] === teamId ? '' : teamId
			};
			return;
		}
		const cur = Array.isArray(this.calls[call.key])
			? [...(this.calls[call.key] as string[])]
			: [];
		const i = cur.indexOf(teamId);
		if (i >= 0) cur.splice(i, 1);
		else if (cur.length < (call.count ?? 99)) cur.push(teamId);
		this.calls = { ...this.calls, [call.key]: cur };
	}

	/** Whether a team is part of a call's current pick. */
	inCall(call: ForecastCall, teamId: string): boolean {
		const v = this.calls[call.key];
		return Array.isArray(v) ? v.includes(teamId) : v === teamId;
	}

	/** Slot the chosen thirds into the 8 R32 third-slots. Uses FIFA's official
	 *  Annex C table (served from the backend) for the chosen combination of 8
	 *  groups; falls back to a deterministic backtracking matching otherwise.
	 *  Mirrors the Go scorer exactly so the Forecast bracket + scoring agree. */
	thirdAssignment(): Record<number, string> {
		const slots = [...this.thirdSlots].sort(
			(a, b) => a.matchNum - b.matchNum
		);
		const chosen = this.chosenThirdLetters.sort();

		// Official table for this exact set of qualifying groups.
		if (chosen.length === this.maxThirds && this.maxThirds > 0) {
			const key = [...chosen].sort().join('');
			const map = this.thirdTable[key];
			if (map) {
				const out: Record<number, string> = {};
				for (const s of slots) {
					const g = map[s.winner];
					if (g) out[s.matchNum] = this.groupThird(g);
				}
				return out;
			}
		}

		// Fallback: deterministic backtracking perfect matching.
		const assign: (string | null)[] = new Array(slots.length).fill(null);

		const solve = (i: number): boolean => {
			if (i === slots.length) return true;
			for (const letter of chosen) {
				if (assign.includes(letter)) continue;
				if (!slots[i].allowed.includes(letter)) continue;
				assign[i] = letter;
				if (solve(i + 1)) return true;
				assign[i] = null;
			}
			return false;
		};
		solve(0);

		const out: Record<number, string> = {};
		slots.forEach((s, i) => {
			if (assign[i]) out[s.matchNum] = this.groupThird(assign[i] as string);
		});
		return out;
	}

	async save() {
		// Persist thirds as {groupLetter: currentThirdTeamId} so the value
		// stays correct even if the group order changed after ticking.
		const thirdQualifiers: Record<string, string> = {};
		for (const letter of this.chosenThirdLetters)
			thirdQualifiers[letter] = this.groupThird(letter);
		const data = {
			user: auth.user?.id,
			tournament: tournamentStore.current?.id,
			groupOrder: this.groupOrder,
			thirdQualifiers,
			bracket: this.bracket,
			calls: this.calls
		};
		const rec = this.recId
			? await pb.collection('forecasts').update(this.recId, data)
			: await pb.collection('forecasts').create(data);
		this.recId = rec.id;
	}
}

export const forecastStore = new ForecastStore();
