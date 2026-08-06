<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { ArrowRight } from '@lucide/svelte';

	// The landing is mounted for signed-out visitors at `/` and for signed-in
	// users revisiting via /welcome — keep the primary CTA honest for both.
	let primaryHref = $derived(auth.isAuthed ? '/' : '/register');
	let primaryLabel = $derived(auth.isAuthed ? 'Back to the app' : 'Create account');

	// How you play — the three real modes, in the order you meet them.
	const steps = [
		{
			title: 'Tips',
			body: 'Predict the score of every match. Edit until kickoff — after that, everyone’s picks go public and the gloating begins.'
		},
		{
			title: 'Forecast',
			body: 'One big call before it all starts: full group standings and the whole knockout bracket. Locks at the first kickoff, pays out all tournament.'
		},
		{
			title: 'Leagues',
			body: 'Private leaderboards with the people you actually know. Join with an invite code or link — and your league carries over to the next tournament.'
		}
	];

	// Demo matchday — neutral codes, plausible scores, real scoring tiers.
	const demoTips = [
		{ home: 'AUT', away: 'JPN', hs: 2, as: 1, tip: '2 : 1', pts: '+6', hit: 'Exact score' },
		{ home: 'FRA', away: 'NOR', hs: 1, as: 0, tip: '2 : 0', pts: '+3', hit: 'Right result' }
	];

	// Demo group table — top two advance.
	const demoTable = [
		{ pos: 1, code: 'AUT', pts: 7, adv: true },
		{ pos: 2, code: 'JPN', pts: 5, adv: true },
		{ pos: 3, code: 'MEX', pts: 3, adv: false },
		{ pos: 4, code: 'CIV', pts: 1, adv: false }
	];

	// The bot bench — AI opponents that play leagues under the same rules.
	const bots = [
		{ name: 'Claude', icon: 'claude-icon.png' },
		{ name: 'ChatGPT', icon: 'gpt-icon.png' },
		{ name: 'Gemini', icon: 'gemini-icon.png' },
		{ name: 'DeepSeek', icon: 'deepseek-icon.png' },
		{ name: 'Grok', icon: 'grok-icon.png' },
		{ name: 'Kimi', icon: 'kimi-icon.png' },
		{ name: 'Qwen', icon: 'qwen-icon.png' }
	];
</script>

<div class="land stagger">
	<!-- ============ HERO ============ -->
	<header class="hero">
		<!-- The Matchowl mark: an owl face built from football pitch markings.
		     Same artwork as Logo.svelte, with the eyes grouped so they can blink. -->
		<svg class="owl" viewBox="0 0 256 256" role="img" aria-label="Matchowl owl mark">
			<rect width="256" height="256" rx="56" fill="#FFD7B4" />
			<path d="M20 201L236 201" stroke="#FF7700" stroke-width="14" />
			<rect
				x="127.971"
				y="131"
				width="24"
				height="24"
				transform="rotate(45 127.971 131)"
				fill="#FF7700"
			/>
			<g class="eye">
				<circle cx="94" cy="120" r="36" fill="#141414" />
				<circle cx="94" cy="120" r="23" fill="#fff" />
				<circle cx="94" cy="119.5" r="12" fill="#FF7700" />
			</g>
			<g class="eye">
				<circle cx="162" cy="120" r="36" fill="#141414" />
				<circle cx="162" cy="120" r="23" fill="#fff" />
				<circle cx="162" cy="119.5" r="12" fill="#FF7700" />
			</g>
			<path d="M198 20V88H58V20" stroke="#141414" stroke-width="14" fill="none" />
			<path d="M161 20V56H95V20" stroke="#141414" stroke-width="14" fill="none" />
		</svg>

		<p class="wordmark">Matchowl</p>

		<h1 class="head">
			You call the tournament.<br />
			<span class="grad">Your friends call it wrong.</span>
		</h1>

		<p class="sub">
			A football prediction game for any tournament — tip every match, forecast the whole
			bracket, and settle it on a leaderboard of people you actually know.
		</p>

		<p class="facts" aria-label="Free, open source, no ads">
			<span>Free</span><span class="fact-dot" aria-hidden="true"></span>
			<span>Open source</span><span class="fact-dot" aria-hidden="true"></span>
			<span>No ads</span>
		</p>

		<div class="cta">
			<a class="btn" href={primaryHref}>{primaryLabel} <ArrowRight size={18} /></a>
			{#if !auth.isAuthed}
				<a class="btn secondary" href="/login">Sign in</a>
			{/if}
		</div>
	</header>

	<!-- ============ MATCHDAY DEMO ============ -->
	<section class="card demo" aria-label="What a matchday looks like">
		<div class="demo-half">
			<p class="kicker">Matchday</p>
			<ul class="scores">
				{#each demoTips as m (m.home)}
					<li class="score-row">
						<span class="pill ft">FT</span>
						<span class="team">{m.home}</span>
						<span class="digits scoreline">{m.hs}&nbsp;:&nbsp;{m.as}</span>
						<span class="team">{m.away}</span>
						<span class="tip-chip"
							>Your tip <b class="digits">{m.tip}</b>
							<b class="pts digits">{m.pts}</b> <i>{m.hit}</i></span
						>
					</li>
				{/each}
			</ul>
			<p class="demo-note muted">
				Tips stay editable until kickoff — then everyone’s picks are revealed.
			</p>
		</div>

		<div class="demo-half">
			<p class="kicker">Group B · live</p>
			<table class="mini-table">
				<thead>
					<tr><th class="num">#</th><th>Team</th><th class="num">Pts</th></tr>
				</thead>
				<tbody>
					{#each demoTable as row (row.code)}
						<tr class:adv={row.adv}>
							<td class="num digits">{row.pos}</td>
							<td>{row.code}</td>
							<td class="num digits">{row.pts}</td>
						</tr>
					{/each}
				</tbody>
			</table>
			<p class="demo-note muted">
				Group tables and the knockout tree fill in live from real results.
			</p>
		</div>
	</section>

	<!-- ============ HOW YOU PLAY ============ -->
	<section class="how" aria-label="How you play">
		<h2>Three ways to be right</h2>
		<div class="how-grid">
			{#each steps as s (s.title)}
				<div class="card how-card">
					<span class="dot" aria-hidden="true"></span>
					<h3>{s.title}</h3>
					<p class="muted">{s.body}</p>
				</div>
			{/each}
		</div>
	</section>

	<!-- ============ AI OPPONENTS ============ -->
	<section class="card bots" aria-label="AI opponents">
		<p class="kicker">AI opponents</p>
		<h2>Short a rival? Add a bot.</h2>
		<p class="muted bots-sub">
			Invite AI players into your league — they tip and forecast under exactly the same rules,
			and they lose just like everyone else.
		</p>
		<ul class="bot-row">
			{#each bots as b (b.name)}
				<li class="bot">
					<img src="/bots/{b.icon}" alt="" loading="lazy" width="44" height="44" />
					<span>{b.name}</span>
				</li>
			{/each}
		</ul>
	</section>

	<!-- ============ CLOSING CTA ============ -->
	<section class="card outro" aria-label="Get started">
		<h2>Kickoff is the deadline.</h2>
		<p class="muted">
			Free forever, open source, no ads — and it installs like an app on your phone.
		</p>
		<div class="cta">
			<a class="btn" href={primaryHref}>{primaryLabel} <ArrowRight size={18} /></a>
			{#if !auth.isAuthed}
				<a class="btn secondary" href="/login">I already have an account</a>
			{/if}
		</div>
	</section>

	<footer class="foot">
		<p>
			Matchowl ·
			<a href="https://github.com/floholz/matchowl" target="_blank" rel="noopener">open source</a>
			· made for the love of the game · by floholz
		</p>
		<p class="muted foot-note">Previously WM Tips — now built for any tournament.</p>
	</footer>
</div>

<style>
	.land {
		display: flex;
		flex-direction: column;
		gap: 2.6rem;
		padding: 1.5rem 0 2rem;
	}

	/* ---- Hero ---- */
	.hero {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		gap: 0.9rem;
		padding-top: 1.5rem;
	}
	.owl {
		width: clamp(96px, 22vw, 140px);
		height: auto;
		display: block;
		border-radius: calc(clamp(96px, 22vw, 140px) * 0.22);
		box-shadow: var(--glow);
	}
	/* The signature: the owl blinks. Rarely, like a real one. */
	.eye {
		transform-box: fill-box;
		transform-origin: 50% 55%;
		animation: blink 6.4s ease-in-out infinite;
	}
	@keyframes blink {
		0%,
		93.5%,
		100% {
			transform: scaleY(1);
		}
		96.5% {
			transform: scaleY(0.08);
		}
	}
	.wordmark {
		margin: 0;
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 1.5rem;
		letter-spacing: -0.015em;
	}
	.head {
		font-size: clamp(2.1rem, 6.5vw, 3.3rem);
		max-width: 21ch;
	}
	.grad {
		color: var(--accent);
	}
	.sub {
		margin: 0;
		max-width: 44ch;
		font-size: 1.05rem;
		line-height: 1.55;
		color: var(--muted);
	}
	.facts {
		display: flex;
		align-items: center;
		gap: 0.65rem;
		margin: 0;
		font-weight: 700;
		font-size: 0.82rem;
		letter-spacing: 0.14em;
		text-transform: uppercase;
		color: var(--accent);
	}
	.fact-dot {
		width: 5px;
		height: 5px;
		border-radius: 50%;
		background: var(--muted);
	}

	/* ---- CTA rows (hero + outro) ---- */
	.cta {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 0.7rem;
		margin-top: 0.5rem;
		width: 100%;
	}
	.cta .btn {
		width: auto;
		min-width: 200px;
		flex: 0 1 auto;
	}
	@media (max-width: 480px) {
		.cta .btn {
			width: 100%;
		}
	}

	/* ---- Matchday demo ---- */
	.demo {
		display: grid;
		gap: 1.5rem;
		padding: 1.4rem;
	}
	@media (min-width: 720px) {
		.demo {
			grid-template-columns: 1.25fr 1fr;
			gap: 2rem;
		}
	}
	.demo .kicker {
		margin: 0 0 0.8rem;
	}
	.scores {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
	}
	.score-row {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.55rem;
		padding: 0.65rem 0.8rem;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
	}
	.ft {
		font-size: 0.62rem;
	}
	.team {
		font-weight: 800;
		letter-spacing: 0.04em;
	}
	.scoreline {
		font-size: 1.15rem;
	}
	.tip-chip {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		margin-left: auto;
		font-size: 0.78rem;
		color: var(--muted);
	}
	.tip-chip b {
		font-style: normal;
		color: var(--text);
	}
	.tip-chip .pts {
		color: var(--accent);
	}
	.tip-chip i {
		font-style: normal;
		font-weight: 700;
		font-size: 0.68rem;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--gold);
	}
	.demo-note {
		margin: 0.8rem 0 0;
		font-size: 0.85rem;
	}

	.mini-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.95rem;
	}
	.mini-table th {
		text-align: left;
		font-size: 0.68rem;
		font-weight: 700;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--muted);
		padding: 0.25rem 0.6rem;
	}
	.mini-table td {
		padding: 0.5rem 0.6rem;
		border-top: 1px solid var(--border);
		font-weight: 700;
	}
	.mini-table .num {
		text-align: right;
		width: 2.2rem;
	}
	.mini-table tr.adv td {
		background: color-mix(in srgb, var(--accent) 10%, transparent);
	}
	.mini-table tr.adv td:first-child {
		color: var(--accent);
	}

	/* ---- How you play ---- */
	.how {
		display: flex;
		flex-direction: column;
		gap: 1.1rem;
	}
	.how h2 {
		text-align: center;
	}
	.how-grid {
		display: grid;
		gap: 0.85rem;
	}
	@media (min-width: 720px) {
		.how-grid {
			grid-template-columns: repeat(3, 1fr);
		}
	}
	.how-card {
		display: flex;
		flex-direction: column;
		gap: 0.55rem;
		padding: 1.3rem;
	}
	.how-card + .how-card {
		margin-top: 0; /* .card + .card default doesn't apply in the grid */
	}
	.how-card h3 {
		font-family: var(--font-display);
		font-size: 1.2rem;
	}
	.how-card p {
		margin: 0;
		line-height: 1.5;
		font-size: 0.94rem;
	}
	/* Section bullet = the owl's pupil: orange core, peach iris, dark ring. */
	.dot {
		width: 13px;
		height: 13px;
		border-radius: 50%;
		background: var(--accent);
		box-shadow:
			0 0 0 5px var(--peach),
			0 0 0 7px var(--border);
		margin: 4px 0 8px 7px;
	}

	/* ---- AI opponents ---- */
	.bots {
		text-align: center;
		padding: 1.6rem 1.4rem;
	}
	.bots .kicker {
		margin: 0 0 0.5rem;
	}
	.bots-sub {
		max-width: 46ch;
		margin: 0.6rem auto 1.2rem;
		line-height: 1.5;
		font-size: 0.95rem;
	}
	.bot-row {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 0.6rem;
	}
	.bot {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.4rem;
		width: 78px;
		padding: 0.7rem 0.3rem;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
	}
	.bot img {
		width: 44px;
		height: 44px;
		border-radius: 12px;
		object-fit: contain;
	}
	.bot span {
		font-size: 0.72rem;
		font-weight: 700;
		color: var(--muted);
	}

	/* ---- Closing CTA ---- */
	.outro {
		text-align: center;
		padding: 2rem 1.4rem;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.7rem;
	}
	.outro p {
		margin: 0;
		max-width: 42ch;
		line-height: 1.5;
	}

	/* ---- Footer ---- */
	.foot {
		text-align: center;
		font-size: 0.85rem;
		color: var(--muted);
		padding-bottom: 0.5rem;
	}
	.foot p {
		margin: 0;
	}
	.foot-note {
		margin-top: 0.35rem;
		font-size: 0.78rem;
	}

	@media (prefers-reduced-motion: reduce) {
		.eye {
			animation: none;
		}
	}
</style>
