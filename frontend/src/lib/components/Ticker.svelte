<!-- The ticker of one match (match_events, written by the live sync):
     newest first, home events left and away right of the minute, goals as
     running-score chips, cards as small cards, substitutions, VAR. HT / FT
     dividers carry the score. Realtime keeps it current during play. Not a
     ticker app: a layer on the match page. -->
<script lang="ts">
	import { pb } from '$lib/pb';
	import type { Match } from '$lib/tips.svelte';
	import { ArrowRightLeft } from '@lucide/svelte';

	interface MatchEvent {
		id: string;
		seq: number;
		minute: number;
		extra: number;
		type: string;
		detail: string;
		side: '' | 'home' | 'away';
		player: string;
		assist: string;
		comments: string;
	}

	let { match }: { match: Match } = $props();

	let events = $state<MatchEvent[]>([]);
	let loaded = $state(false);

	async function load(id: string) {
		events = (await pb
			.collection('match_events')
			.getFullList({ filter: `match = "${id}"`, sort: 'seq' })
			.catch(() => [])) as unknown as MatchEvent[];
		loaded = true;
	}
	$effect(() => {
		const id = match.id;
		load(id);
		let timer: ReturnType<typeof setTimeout> | null = null;
		let unsub: (() => void) | null = null;
		pb.collection('match_events')
			.subscribe(
				'*',
				() => {
					// A pass writes several rows at once: refetch once.
					if (timer) return;
					timer = setTimeout(() => {
						timer = null;
						load(id);
					}, 400);
				},
				{ filter: `match = "${id}"` }
			)
			.then((u) => (unsub = u))
			.catch(() => {});
		return () => {
			if (timer) clearTimeout(timer);
			unsub?.();
		};
	});

	const isGoal = (e: MatchEvent) => e.type.toLowerCase() === 'goal' && !/missed/i.test(e.detail);
	const isOwnGoal = (e: MatchEvent) => /own goal/i.test(e.detail);
	const other = (s: string) => (s === 'home' ? 'away' : 'home');

	/** Running score after each event. Own goals: the provider's team on
	 *  the event isn't documented either way, so use the reading whose
	 *  final tally matches the match score; null when neither does. */
	let running = $derived.by((): ([number, number] | null)[] | null => {
		const tally = (ownForOther: boolean) => {
			let h = 0;
			let a = 0;
			const out: [number, number][] = [];
			for (const e of events) {
				if (isGoal(e) && e.side) {
					const s = isOwnGoal(e) && ownForOther ? other(e.side) : e.side;
					if (s === 'home') h++;
					else a++;
				}
				out.push([h, a]);
			}
			return out;
		};
		const live = match.status === 'live' || match.status === 'finished' || !!match.finalizedAt;
		if (!live) return null;
		const target = [match.ftHome ?? 0, match.ftAway ?? 0];
		for (const flip of [true, false]) {
			const t = tally(flip);
			const last = t[t.length - 1] ?? [0, 0];
			if (last[0] === target[0] && last[1] === target[1]) return t;
		}
		return null;
	});

	type Row =
		| { kind: 'phase'; key: string; label: string; score: string }
		| { kind: 'event'; key: string; e: MatchEvent; score: [number, number] | null };

	/** Newest first, with HT / FT dividers. Built in play order, then
	 *  reversed. */
	let rows = $derived.by((): Row[] => {
		const finished = match.status === 'finished' || !!match.finalizedAt;
		const phase = match.livePhase ?? '';
		const pastHalf = finished || ['2H', 'ET', 'BT', 'P'].includes(phase);
		const scoreAt = (i: number) => {
			if (i < 0) return '0–0';
			const r = running?.[i];
			return r ? `${r[0]}–${r[1]}` : '';
		};
		const out: Row[] = [];
		let ht = false;
		events.forEach((e, i) => {
			if (pastHalf && !ht && e.minute > 45) {
				out.push({ kind: 'phase', key: 'ht', label: 'HT', score: scoreAt(i - 1) });
				ht = true;
			}
			out.push({ kind: 'event', key: e.id, e, score: running?.[i] ?? null });
		});
		if (pastHalf && !ht) out.push({ kind: 'phase', key: 'ht', label: 'HT', score: scoreAt(events.length - 1) });
		if (phase === 'HT' && !finished) out.push({ kind: 'phase', key: 'ht', label: 'Half-time', score: scoreAt(events.length - 1) });
		if (finished) out.push({ kind: 'phase', key: 'ft', label: 'FT', score: `${match.ftHome ?? 0}–${match.ftAway ?? 0}` });
		return out.reverse();
	});

	const minute = (e: MatchEvent) => `${e.minute}${e.extra ? `+${e.extra}` : ''}'`;

	/** What a row says: the main line and the muted second line. */
	function lines(e: MatchEvent): { main: string; sub: string; muted: boolean } {
		const t = e.type.toLowerCase();
		const d = e.detail.toLowerCase();
		if (t === 'goal') {
			if (d.includes('missed')) return { main: e.player || 'Penalty', sub: 'missed penalty', muted: true };
			const sub = d.includes('own') ? 'own goal' : d.includes('penalty') ? 'penalty' : e.assist ? `assist ${e.assist}` : '';
			return { main: e.player || 'Goal', sub, muted: false };
		}
		if (t === 'card') return { main: e.player || e.detail, sub: d.includes('red') && d.includes('yellow') ? 'second yellow' : '', muted: false };
		if (t === 'subst') return { main: e.player || 'Substitution', sub: e.assist ? `⇄ ${e.assist}` : 'substitution', muted: false };
		if (t === 'var') return { main: e.detail || 'VAR', sub: e.player || e.comments, muted: true };
		return { main: e.detail || e.type, sub: e.player, muted: true };
	}
	const cardClass = (e: MatchEvent) => {
		const d = e.detail.toLowerCase();
		return d.includes('red') ? (d.includes('yellow') ? 'second' : 'red') : 'yellow';
	};
</script>

{#if loaded && events.length}
	<div class="sec">
		<h2>Ticker</h2>
		{#if match.status === 'live'}<span class="pill live">Live</span>{/if}
	</div>
	<div class="card ticker">
		{#each rows as r (r.key)}
			{#if r.kind === 'phase'}
				<div class="phase">{r.label}{#if r.score} · <span class="digits">{r.score}</span>{/if}</div>
			{:else}
				{@const e = r.e}
				{@const l = lines(e)}
				{@const t = e.type.toLowerCase()}
				<div class="ev {e.side || 'home'}">
					<span class="txt" class:muted={l.muted}>
						<b>{l.main}</b>
						{#if l.sub}<small>{l.sub}</small>{/if}
					</span>
					<span class="ic">
						{#if t === 'goal' && !/missed/i.test(e.detail)}
							<span class="goal digits">{r.score ? `${r.score[0]}–${r.score[1]}` : 'Goal'}</span>
						{:else if t === 'card'}
							<span class="cardic {cardClass(e)}"></span>
						{:else if t === 'subst'}
							<ArrowRightLeft size={15} class="sub" />
						{:else if t === 'var'}
							<span class="var">VAR</span>
						{:else}
							<span class="dot"></span>
						{/if}
					</span>
					<span class="min digits">{minute(e)}</span>
				</div>
			{/if}
		{/each}
	</div>
{/if}

<style>
	.sec {
		display: flex;
		align-items: baseline;
		gap: 0.5rem;
		margin: 0.6rem 0.15rem 0;
	}
	.sec h2 {
		font-size: 1.1rem;
	}
	.ticker {
		padding: 0;
		overflow: hidden;
	}
	.phase {
		text-align: center;
		font-size: 0.74rem;
		font-weight: 700;
		letter-spacing: 0.06em;
		color: var(--muted);
		padding: 0.4rem;
		background: color-mix(in srgb, var(--text) 3%, transparent);
		border-bottom: 1px solid var(--border);
	}
	/* Grid: home text · icon · minute · icon · away text. Home rows use the
	   left half, away rows the right half; the minute stays centred. */
	.ev {
		display: grid;
		grid-template-columns: 1fr auto 3.2rem auto 1fr;
		align-items: center;
		gap: 0.45rem;
		padding: 0.45rem 0.75rem;
		border-bottom: 1px solid color-mix(in srgb, var(--border) 60%, transparent);
		min-height: 2.6rem;
	}
	.ev:last-child {
		border-bottom: none;
	}
	.min {
		grid-column: 3;
		grid-row: 1;
		text-align: center;
		font-size: 0.78rem;
		font-weight: 700;
		color: var(--muted);
	}
	.ev.home .txt {
		grid-column: 1;
		text-align: right;
		align-items: flex-end;
	}
	.ev.home .ic {
		grid-column: 2;
	}
	.ev.away .ic {
		grid-column: 4;
	}
	.ev.away .txt {
		grid-column: 5;
		text-align: left;
		align-items: flex-start;
	}
	.txt,
	.ic {
		grid-row: 1;
	}
	.txt {
		display: flex;
		flex-direction: column;
		min-width: 0;
		font-size: 0.86rem;
		line-height: 1.2;
	}
	.txt b {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		max-width: 100%;
	}
	.txt small {
		color: var(--muted);
		font-size: 0.74rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		max-width: 100%;
	}
	.txt.muted b {
		color: var(--muted);
	}
	.ic {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 1.4rem;
	}
	.goal {
		font-size: 0.74rem;
		font-weight: 700;
		color: var(--accent);
		background: color-mix(in srgb, var(--accent) 13%, transparent);
		border-radius: 999px;
		padding: 0.12rem 0.45rem;
		white-space: nowrap;
	}
	.cardic {
		width: 10px;
		height: 14px;
		border-radius: 2px;
		background: #f5c518;
	}
	.cardic.red {
		background: var(--danger);
	}
	.cardic.second {
		background: linear-gradient(135deg, #f5c518 50%, var(--danger) 50%);
	}
	.ic :global(.sub) {
		color: var(--success);
	}
	.var {
		font-size: 0.62rem;
		font-weight: 700;
		color: var(--muted);
		border: 1px solid var(--muted);
		border-radius: 4px;
		padding: 0.05rem 0.25rem;
	}
	.dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--muted);
	}
</style>
