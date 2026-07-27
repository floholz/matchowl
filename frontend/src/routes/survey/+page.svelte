<script lang="ts">
	// One-time end-of-tournament feedback survey. Questions are hard-coded here
	// (single survey, single run); the answer shape is validated server-side in
	// internal/survey. One response per user, no edits after submit.
	import { onMount } from 'svelte';
	import { api, type SurveyAnswers } from '$lib/api';
	import { appConfig } from '$lib/appconfig.svelte';
	import SupportCard from '$lib/components/SupportCard.svelte';
	import { Check, Send, MessageSquareHeart } from '@lucide/svelte';

	let checked = $state(false);
	let submitted = $state(false);
	let justSubmitted = $state(false);
	let editing = $state(false); // re-opened the form after an earlier submit
	let submitting = $state(false);
	let error = $state('');

	// --- answers ---
	let enjoy = $state(0);
	let playedOther = $state<boolean | null>(null);
	let comparison = $state('');
	let liked = $state<string[]>([]);
	let likedOther = $state('');
	let annoyed = $state('');
	let playAgain = $state('');
	let playSeason = $state('');
	let competitions = $state<string[]>([]);
	let competitionsOther = $state('');
	let fairPrice = $state('');
	let comments = $state('');

	// Short labels, roughly length-paired so the growing chips pack into even
	// rows on a phone screen.
	const likedOpts = [
		{ key: 'tips', label: 'Match tips' },
		{ key: 'chat', label: 'League chat' },
		{ key: 'forecast', label: 'Forecast & bracket' },
		{ key: 'notifications', label: 'Notifications' },
		{ key: 'leagues', label: 'Leagues with friends' },
		{ key: 'design', label: 'Look & feel' },
		{ key: 'other', label: 'Something else…' }
	];
	const compOpts = [
		{ key: 'championsLeague', label: 'Champions League' },
		{ key: 'premierLeague', label: 'Premier League' },
		{ key: 'bundesliga', label: 'Bundesliga' },
		{ key: 'laLiga', label: 'La Liga' },
		{ key: 'serieA', label: 'Serie A' },
		{ key: 'ligue1', label: 'Ligue 1' },
		{ key: 'other', label: 'Other…' }
	];

	function toggle(list: string[], key: string): string[] {
		return list.includes(key) ? list.filter((k) => k !== key) : [...list, key];
	}

	let valid = $derived(
		enjoy > 0 &&
			playedOther !== null &&
			(!playedOther || comparison !== '') &&
			playAgain !== '' &&
			playSeason !== '' &&
			fairPrice !== ''
	);

	onMount(() => {
		appConfig.load();
		api
			.surveyStatus()
			.then((r) => {
				submitted = r.submitted;
				if (r.answers) prefill(r.answers);
			})
			.catch(() => {})
			.finally(() => (checked = true));
	});

	// Load an earlier submission back into the form so it can be refined.
	function prefill(a: SurveyAnswers) {
		enjoy = a.enjoy ?? 0;
		playedOther = a.playedOther ?? null;
		comparison = a.comparison ?? '';
		liked = a.liked ?? [];
		likedOther = a.likedOther ?? '';
		annoyed = a.annoyed ?? '';
		playAgain = a.playAgain ?? '';
		playSeason = a.playSeason ?? '';
		competitions = a.competitions ?? [];
		competitionsOther = a.competitionsOther ?? '';
		comments = a.comments ?? '';
	}

	async function submit() {
		if (!valid || submitting) return;
		submitting = true;
		error = '';
		const answers: SurveyAnswers = {
			v: 1,
			enjoy,
			playedOther: playedOther!,
			comparison: playedOther ? (comparison as SurveyAnswers['comparison']) : undefined,
			liked,
			likedOther: liked.includes('other') ? likedOther.trim() : undefined,
			annoyed: annoyed.trim() || undefined,
			playAgain: playAgain as SurveyAnswers['playAgain'],
			playSeason: playSeason as SurveyAnswers['playSeason'],
			competitions: playSeason !== 'no' ? competitions : undefined,
			competitionsOther:
				playSeason !== 'no' && competitions.includes('other')
					? competitionsOther.trim()
					: undefined,
			fairPrice: fairPrice as SurveyAnswers['fairPrice'],
			comments: comments.trim() || undefined
		};
		try {
			await api.submitSurvey(answers);
			submitted = true;
			justSubmitted = true;
			editing = false;
		} catch {
			error = 'Could not send your answers — please try again.';
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head><title>Survey · WM Tips</title></svelte:head>

<header>
	<p class="kicker">Feedback</p>
	<h1>The full-time whistle survey</h1>
	<p class="muted sd">8 quick questions · about 2 minutes · shapes what comes next.</p>
</header>

{#if !checked}
	<section class="card"><p class="muted">Loading…</p></section>
{:else if submitted && !editing}
	<div class="stagger">
		<section class="card done">
			<span class="di"><Check size={22} /></span>
			<h3>{justSubmitted ? 'Thank you!' : 'Already answered — thank you!'}</h3>
			<p class="muted">
				{justSubmitted
					? 'Your answers are in. They directly decide whether WM Tips returns for future tournaments and seasons.'
					: 'Your answers are in — and you can still refine them if something new comes to mind.'}
			</p>
			<div class="done-actions">
				<button class="btn secondary" onclick={() => (editing = true)}>
					Review or change answers
				</button>
				<a class="btn ghost" href="/">Back to home</a>
			</div>
		</section>
		<SupportCard />
	</div>
{:else}
	<div class="stagger">
		<!-- 1 · overall -->
		<section class="card q">
			<h3><span class="qn">1</span> Overall, how much did you enjoy WM Tips?</h3>
			<div class="scale" role="radiogroup" aria-label="Enjoyment from 1 to 5">
				{#each [1, 2, 3, 4, 5] as n (n)}
					<button
						type="button"
						class="sc"
						class:on={enjoy === n}
						role="radio"
						aria-checked={enjoy === n}
						onclick={() => (enjoy = n)}>{n}</button
					>
				{/each}
			</div>
			<div class="scale-lbl muted"><span>not really</span><span>loved it</span></div>
		</section>

		<!-- 2 · other games -->
		<section class="card q">
			<h3><span class="qn">2</span> Have you played other football prediction games before?</h3>
			<div class="seg">
				<button type="button" class:on={playedOther === true} onclick={() => (playedOther = true)}
					>Yes</button
				>
				<button
					type="button"
					class:on={playedOther === false}
					onclick={() => (playedOther = false)}>No</button
				>
			</div>
			{#if playedOther}
				<p class="follow">Compared to those, WM Tips was…</p>
				<div class="seg">
					<button type="button" class:on={comparison === 'better'} onclick={() => (comparison = 'better')}
						>Better</button
					>
					<button type="button" class:on={comparison === 'same'} onclick={() => (comparison = 'same')}
						>About the same</button
					>
					<button type="button" class:on={comparison === 'worse'} onclick={() => (comparison = 'worse')}
						>Worse</button
					>
				</div>
			{/if}
		</section>

		<!-- 3 · liked most -->
		<section class="card q">
			<h3><span class="qn">3</span> What did you like most? <span class="muted opt">pick any</span></h3>
			<div class="chips">
				{#each likedOpts as o (o.key)}
					<button
						type="button"
						class="chip"
						class:on={liked.includes(o.key)}
						aria-pressed={liked.includes(o.key)}
						onclick={() => (liked = toggle(liked, o.key))}
					>
						<span class="ck">{#if liked.includes(o.key)}<Check size={13} />{/if}</span>
						{o.label}
						<span class="ck"></span>
					</button>
				{/each}
			</div>
			{#if liked.includes('other')}
				<input
					class="input follow"
					type="text"
					maxlength="300"
					placeholder="Tell me — what else did you like?"
					bind:value={likedOther}
				/>
			{/if}
		</section>

		<!-- 4 · annoyed -->
		<section class="card q">
			<h3>
				<span class="qn">4</span> What annoyed you, or what was missing?
				<span class="muted opt">optional</span>
			</h3>
			<textarea
				class="input"
				rows="3"
				maxlength="2000"
				placeholder="Be honest — this is the most useful answer in here."
				bind:value={annoyed}
			></textarea>
		</section>

		<!-- 5 · play again -->
		<section class="card q">
			<h3><span class="qn">5</span> Would you play again at the next Euro or World Cup?</h3>
			<div class="seg">
				<button type="button" class:on={playAgain === 'definitely'} onclick={() => (playAgain = 'definitely')}
					>Definitely</button
				>
				<button type="button" class:on={playAgain === 'probably'} onclick={() => (playAgain = 'probably')}
					>Probably</button
				>
				<button type="button" class:on={playAgain === 'no'} onclick={() => (playAgain = 'no')}
					>Probably not</button
				>
			</div>
		</section>

		<!-- 6 · season version -->
		<section class="card q">
			<h3>
				<span class="qn">6</span> Would you also play a season-long version — club football, week
				after week?
			</h3>
			<div class="seg">
				<button type="button" class:on={playSeason === 'definitely'} onclick={() => (playSeason = 'definitely')}
					>Definitely</button
				>
				<button type="button" class:on={playSeason === 'maybe'} onclick={() => (playSeason = 'maybe')}
					>Maybe</button
				>
				<button type="button" class:on={playSeason === 'no'} onclick={() => (playSeason = 'no')}
					>No</button
				>
			</div>
			{#if playSeason && playSeason !== 'no'}
				<p class="follow">Which competitions would you want? <span class="muted opt">pick any</span></p>
				<div class="chips">
					{#each compOpts as o (o.key)}
						<button
							type="button"
							class="chip"
							class:on={competitions.includes(o.key)}
							aria-pressed={competitions.includes(o.key)}
							onclick={() => (competitions = toggle(competitions, o.key))}
						>
							<span class="ck">{#if competitions.includes(o.key)}<Check size={13} />{/if}</span>
							{o.label}
							<span class="ck"></span>
						</button>
					{/each}
				</div>
				{#if competitions.includes('other')}
					<input
						class="input follow"
						type="text"
						maxlength="300"
						placeholder="Which ones? e.g. Eredivisie, Allsvenskan, Brasileirão…"
						bind:value={competitionsOther}
					/>
				{/if}
			{/if}
		</section>

		<!-- 7 · fair price -->
		<section class="card q">
			<h3>
				<span class="qn">7</span> WM Tips was free this time. If it continued, what would feel
				fair?
			</h3>
			<div class="opts">
				<button
					type="button"
					class="optbtn"
					class:on={fairPrice === 'donations'}
					onclick={() => (fairPrice = 'donations')}
				>
					<span class="ot">Free, with optional donations</span>
					<span class="muted os">Like now — chip in if you feel like it</span>
				</button>
				<button
					type="button"
					class="optbtn"
					class:on={fairPrice === 'fee'}
					onclick={() => (fairPrice = 'fee')}
				>
					<span class="ot">A small one-time fee per season</span>
					<span class="muted os">A few euros to cover servers &amp; data</span>
				</button>
				<button
					type="button"
					class="optbtn"
					class:on={fairPrice === 'nopay'}
					onclick={() => (fairPrice = 'nopay')}
				>
					<span class="ot">I wouldn't pay</span>
					<span class="muted os">Honest answer — that's useful too</span>
				</button>
			</div>
		</section>

		<!-- 8 · anything else -->
		<section class="card q">
			<h3>
				<span class="qn">8</span> Anything else you want to say?
				<span class="muted opt">optional</span>
			</h3>
			<textarea
				class="input"
				rows="3"
				maxlength="2000"
				placeholder="Shout-outs, ideas, complaints — all welcome."
				bind:value={comments}
			></textarea>
		</section>

		<section class="card submit-card">
			<div class="sr">
				<span class="si"><MessageSquareHeart size={20} /></span>
				<p class="muted sn">
					Answers are tied to your account and never shown to other players — you can
					come back and change them anytime.
				</p>
			</div>
			<button class="btn" disabled={!valid || submitting} onclick={submit}>
				<Send size={16} />
				{submitting ? 'Sending…' : submitted ? 'Update my answers' : 'Send my answers'}
			</button>
			{#if !valid}
				<p class="muted fine">Questions 1, 2, 5, 6 and 7 still need an answer.</p>
			{/if}
			{#if error}<p class="error">{error}</p>{/if}
		</section>
	</div>
{/if}

<style>
	header {
		margin: 0.25rem 0 1.25rem;
	}
	h1 {
		margin: 0;
		font-size: 1.6rem;
	}
	header .muted {
		margin: 0.2rem 0 0;
	}

	.q h3 {
		margin: 0 0 0.75rem;
		font-size: 1rem;
		line-height: 1.45;
	}
	.qn {
		display: inline-grid;
		place-items: center;
		width: 1.5rem;
		height: 1.5rem;
		margin-right: 0.35rem;
		border-radius: var(--radius-pill);
		background: color-mix(in srgb, var(--accent) 14%, transparent);
		color: var(--accent);
		font-size: 0.8rem;
		font-weight: 800;
		vertical-align: -0.25rem;
	}
	.opt {
		font-size: 0.75rem;
		font-weight: 500;
	}
	.follow {
		margin: 0.75rem 0 0.6rem;
		font-size: 0.9rem;
		font-weight: 600;
	}
	input.follow {
		margin-bottom: 0;
	}

	/* 1–5 scale */
	.scale {
		display: flex;
		gap: 0.4rem;
	}
	.sc {
		flex: 1;
		padding: 0.6rem 0;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--text);
		font-weight: 700;
		font-size: 1rem;
	}
	.sc.on {
		background: var(--accent);
		border-color: var(--accent);
		color: var(--accent-fg);
	}
	.scale-lbl {
		display: flex;
		justify-content: space-between;
		font-size: 0.72rem;
		margin-top: 0.35rem;
	}

	/* multi-select chips */
	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: 0.45rem;
	}
	.chip {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		/* Grow to fill each wrapped row — even mosaic instead of ragged pills. */
		flex: 1 0 auto;
		gap: 0.3rem;
		padding: 0.45rem 0.75rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: var(--surface-2);
		color: var(--text);
		font-size: 0.85rem;
		font-weight: 600;
	}
	.chip.on {
		color: var(--accent);
		border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
		background: color-mix(in srgb, var(--accent) 8%, var(--surface-2));
	}
	/* Fixed checkmark slots on BOTH sides so toggling never shifts the chip's
	   width or the wrap layout; the label stays visually centered. */
	.ck {
		display: grid;
		place-items: center;
		width: 13px;
		height: 13px;
		flex: none;
	}
	.input.follow {
		margin-top: 0.6rem;
		width: 100%;
	}

	textarea.input {
		width: 100%;
		resize: vertical;
		font: inherit;
	}

	/* stacked descriptive options (q7) */
	.opts {
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
	}
	.optbtn {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
		padding: 0.7rem 0.85rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--text);
		text-align: left;
	}
	.optbtn.on {
		border-color: color-mix(in srgb, var(--accent) 55%, var(--border));
		background: color-mix(in srgb, var(--accent) 8%, var(--surface-2));
	}
	.optbtn.on .ot {
		color: var(--accent);
	}
	.ot {
		font-weight: 600;
	}
	.os {
		font-size: 0.78rem;
	}

	/* submit */
	.submit-card .btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.45rem;
		width: 100%;
	}
	.sr {
		display: flex;
		align-items: flex-start;
		gap: 0.6rem;
		margin-bottom: 0.85rem;
	}
	.si {
		display: grid;
		place-items: center;
		width: 34px;
		height: 34px;
		flex: none;
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--accent);
	}
	.sn {
		margin: 0.2rem 0 0;
		font-size: 0.82rem;
		line-height: 1.45;
	}
	.fine {
		font-size: 0.78rem;
		margin: 0.55rem 0 0;
		text-align: center;
	}
	.error {
		margin: 0.55rem 0 0;
		text-align: center;
	}

	/* thank-you */
	.done {
		text-align: center;
		padding: 2rem 1.25rem;
	}
	.di {
		display: grid;
		place-items: center;
		width: 46px;
		height: 46px;
		margin: 0 auto 0.7rem;
		border-radius: var(--radius-pill);
		background: color-mix(in srgb, var(--accent) 18%, transparent);
		color: var(--accent);
	}
	.done h3 {
		margin: 0 0 0.35rem;
	}
	.done .muted {
		margin: 0 auto;
		max-width: 42ch;
		line-height: 1.5;
	}
	.done-actions {
		display: flex;
		justify-content: center;
		gap: 0.6rem;
		margin-top: 1.1rem;
	}
	.done-actions .btn {
		width: auto;
	}
</style>
