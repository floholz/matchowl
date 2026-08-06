<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { feedStore, type FeedMatch } from '$lib/feed.svelte';
	import { serverClock } from '$lib/serverclock.svelte';
	import Landing from '$lib/components/Landing.svelte';
	import TipCard from '$lib/components/TipCard.svelte';
	import SupportCard from '$lib/components/SupportCard.svelte';
	import { tick } from 'svelte';
	import { Telescope, Plus, ChevronUp, ChevronDown } from '@lucide/svelte';

	let openId = $state('');
	let scrolled = $state(false);

	$effect(() => {
		if (auth.isAuthed && !feedStore.loaded && !feedStore.loading) {
			feedStore.load().then(scrollToToday).catch(() => {});
		}
	});

	// Land on today (or the first upcoming day) once, after first render.
	async function scrollToToday() {
		if (scrolled) return;
		scrolled = true;
		await tick();
		document
			.getElementById('feed-today')
			?.scrollIntoView({ block: 'start', behavior: 'instant' as ScrollBehavior });
	}

	/** The section today's anchor lands on: today itself, else the first
	 *  future day (there may be no matches today). */
	let anchorKey = $derived.by(() => {
		const days = feedStore.days;
		const today = days.find((d) => d.isToday);
		if (today) return today.key;
		return days.find((d) => d.key > feedStore.todayKey)?.key ?? '';
	});

	function tipFor(m: FeedMatch) {
		return m.myTip ? { match: m.id, ...m.myTip } : null;
	}

	function locksIn(iso: string): string {
		const ms = new Date(iso).getTime() - serverClock.now();
		const days = Math.floor(ms / 86400_000);
		if (days >= 2) return `locks in ${days} days`;
		const hours = Math.floor(ms / 3600_000);
		if (hours >= 2) return `locks in ${hours} hours`;
		return 'locks soon';
	}
</script>

{#if !auth.isAuthed}
	<Landing />
{:else}
	<div class="feed stagger">
		<header class="fhead">
			<p class="kicker">Your matchday</p>
			<h1>Feed</h1>
		</header>

		{#each feedStore.deadlines as d (d.tournament.id)}
			<a class="card deadline" href={`/forecast?t=${d.tournament.slug}`}>
				<span class="dl-ic"><Telescope size={20} /></span>
				<span class="dl-txt">
					<b>{d.hasForecast ? 'Finish your' : 'Make your'} {d.tournament.shortName || d.tournament.name} forecast</b>
					<span class="muted">One shot before kickoff — {locksIn(d.locksAt)}.</span>
				</span>
			</a>
		{/each}

		{#each feedStore.suggestions.filter((s) => s.leagueMates > 0) as s (s.id)}
			<div class="card suggest">
				<span class="dl-txt">
					<b>{s.name}</b>
					<span class="muted"
						>{s.leagueMates}
						{s.leagueMates === 1 ? 'league mate plays' : 'league mates play'} this</span
					>
				</span>
				<span class="spacer"></span>
				<button class="btn slim" onclick={() => feedStore.play(s.slug)}>
					<Plus size={16} /> Play
				</button>
			</div>
		{/each}

		{#if feedStore.loaded && feedStore.days.length === 0}
			<div class="card empty">
				<p><b>Nothing in your feed yet.</b></p>
				<p class="muted">
					Play a tournament and its matches show up here, day by day.
				</p>
				<a class="btn" href="/tournaments">Browse tournaments</a>
			</div>
		{:else if feedStore.loaded}
			<button class="btn ghost more" onclick={() => feedStore.earlier()}>
				<ChevronUp size={16} /> Earlier results
			</button>

			{#each feedStore.days as day (day.key)}
				<section class="day" id={day.key === anchorKey ? 'feed-today' : undefined}>
					<h2 class="day-h" class:today={day.isToday}>
						{day.isToday ? 'Today' : day.label}
					</h2>
					{#each day.matches as m (m.id)}
						<div class="fm">
							<span class="src kicker"
								>{m.tournament.shortName || m.tournament.name}</span
							>
							<TipCard
								match={m}
								team={(id) => feedStore.team(id)}
								tip={tipFor(m)}
								knockout={m.knockout}
								points={m.myTip?.points}
								onSave={(t) => feedStore.saveTip(m, t)}
								open={openId === m.id}
								onToggle={() => (openId = openId === m.id ? '' : m.id)}
							/>
						</div>
					{/each}
				</section>
			{/each}

			<button class="btn ghost more" onclick={() => feedStore.later()}>
				<ChevronDown size={16} /> Later fixtures
			</button>
		{/if}

		<SupportCard />
		<footer class="foot muted">Matchowl · made by floholz</footer>
	</div>
{/if}

<style>
	.fhead {
		margin: 0.4rem 0 1.1rem;
	}
	.deadline,
	.suggest {
		display: flex;
		align-items: center;
		gap: 0.9rem;
		margin-bottom: 0.85rem;
		color: var(--text);
	}
	.deadline {
		border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
		background:
			linear-gradient(
				135deg,
				color-mix(in srgb, var(--accent) 10%, transparent),
				transparent 55%
			),
			var(--surface);
	}
	.dl-ic {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 40px;
		height: 40px;
		border-radius: 50%;
		flex: none;
		color: var(--accent-fg);
		background: var(--accent);
	}
	.dl-txt {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}
	.dl-txt .muted {
		font-size: 0.85rem;
	}
	.btn.slim {
		width: auto;
		padding: 0.5rem 1rem;
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
	}
	.day {
		margin-top: 1.4rem;
		/* room for the fixed header when the today-anchor scrolls here */
		scroll-margin-top: calc(var(--topbar-h) + 0.8rem);
	}
	.day-h {
		font-size: 1.05rem;
		margin-bottom: 0.6rem;
		color: var(--muted);
	}
	.day-h.today {
		color: var(--accent);
	}
	.fm {
		margin-bottom: 0.85rem;
	}
	.fm .src {
		display: block;
		font-size: 0.62rem;
		margin: 0 0 0.25rem 0.25rem;
		color: var(--muted);
	}
	.btn.ghost.more {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.35rem;
		margin: 1rem 0;
	}
	.empty {
		text-align: center;
		padding: 2rem 1.2rem;
	}
	.empty .btn {
		margin-top: 1rem;
	}
	.foot {
		text-align: center;
		font-size: 0.75rem;
		padding: 2rem 0 1rem;
	}
</style>
