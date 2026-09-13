<script lang="ts">
	// In-app help: how Matchowl works, spot by spot. Public (the sign-in page
	// links here), with the normal chrome when signed in. Replaces the old
	// /welcome, which showed the marketing landing.
	import { onMount } from 'svelte';
	import { auth } from '$lib/auth.svelte';
	import { appConfig } from '$lib/appconfig.svelte';
	import { pageChrome } from '$lib/shell.svelte';
	import { defaultScoring, type ScoringConfig } from '$lib/scoring';
	import Logo from '$lib/components/Logo.svelte';
	import {
		Trophy,
		Target,
		Telescope,
		Users,
		MessagesSquare,
		Bell,
		Smartphone,
		UserCheck,
		Mail
	} from '@lucide/svelte';

	pageChrome(() => ({ title: 'Help' }));

	let scoring = $state<ScoringConfig | null>(null);
	onMount(() => {
		appConfig.load();
		defaultScoring().then((s) => (scoring = s));
	});

	const sections = [
		{ id: 'play', label: 'Play' },
		{ id: 'tip', label: 'Tip' },
		{ id: 'forecast', label: 'Forecast' },
		{ id: 'friends', label: 'Friends' },
		{ id: 'pools', label: 'Pools' },
		{ id: 'notify', label: 'Notifications' },
		{ id: 'account', label: 'Account' }
	];
</script>

<div class="help">
	{#if !auth.isAuthed}
		<div class="brand"><Logo size={36} /></div>
	{/if}
	<h1>How Matchowl works</h1>
	<p class="lead muted">
		Play the competitions you follow, tip every match, place one forecast per
		season, and compare with your friends. Free, no betting, no prizes — only
		bragging rights.
	</p>

	<nav class="jump" aria-label="Sections">
		{#each sections as s (s.id)}
			<a href={`#${s.id}`}>{s.label}</a>
		{/each}
	</nav>

	<section id="play">
		<h2><Trophy size={18} /> Play a competition</h2>
		<p>
			Under <b>Competitions</b> you find every league, cup and tournament
			Matchowl runs. Tap <b>Play</b> on the season you want to follow — its
			matches join your Matches feed and Home. Tipping a match of a competition
			you don't play yet joins it for you. You can leave a competition again at
			any time; your tips stay.
		</p>
	</section>

	<section id="tip">
		<h2><Target size={18} /> Tip a match</h2>
		<p>
			Every match row shows the score board on the left and your tip capsule
			on the right. Tap the capsule to set a score with the steppers; it saves
			when you close it. Tips lock at kick-off — after that the capsule shows
			what you had, and once the match is over it turns grey on a miss and
			stays orange on a hit, with the points next to it.
		</p>
		<p>
			In a knockout match a draw is a draw after 90 minutes; the small dot on
			the capsule is your pick for who goes through. In a two-legged tie both
			legs are tipped like normal matches, and the strip under the row shows
			the aggregate and who advances.
		</p>
		{#if scoring}
			<div class="card pts">
				<p class="kicker">Points per match</p>
				<table>
					<tbody>
						<tr><td>Correct result (home win, draw, away win)</td><td class="digits">{scoring.match.tendency}</td></tr>
						{#if scoring.match.goalDiff}<tr><td>Correct goal difference</td><td class="digits">+{scoring.match.goalDiff}</td></tr>{/if}
						{#if scoring.match.totalGoals}<tr><td>Correct total goals</td><td class="digits">+{scoring.match.totalGoals}</td></tr>{/if}
						{#if scoring.match.exact}<tr><td>Exact score</td><td class="digits">+{scoring.match.exact}</td></tr>{/if}
					</tbody>
				</table>
				<p class="muted small">
					The parts add up: an exact score collects all of them. A pool can run
					its own weights; the match page shows the breakdown of every scored
					tip.
				</p>
			</div>
		{/if}
	</section>

	<section id="forecast">
		<h2><Telescope size={18} /> Forecast a season</h2>
		<p>
			Each season you play has one forecast, placed before its first match and
			locked from then on. For a league it is a handful of calls — champion,
			the European places, relegation. For a tournament it is the full thing:
			group standings, the bracket, the winner. Forecast points come in as the
			season resolves, and they count on every board next to your tip points.
		</p>
	</section>

	<section id="friends">
		<h2><Users size={18} /> Friends</h2>
		<p>
			Friends are mutual: one side asks, the other accepts. The <b>Friends</b>
			tab shows a board of you and your friends for the competition you pick,
			and the everyone board next to it. Find people by name — search only finds
			verified accounts.
		</p>
	</section>

	<section id="pools">
		<h2><MessagesSquare size={18} /> Pools</h2>
		<p>
			A pool is your private table: a name, the seasons it counts, an invite
			code or link, a chat. Start one under Friends, share the code, and the
			board sums everyone's points over the pool's seasons. Chips on the board
			narrow it to a single season. Whoever starts a pool runs it: rename,
			change the seasons, regenerate the code, remove members, add a bot or
			two.
		</p>
		<p>
			When every season a pool counts is over, the pool is finished: the board
			stays as history, the chat stays open for a month, and the owner can set
			the pool up again for the next season with one tap — same members,
			same settings, fresh invites.
		</p>
	</section>

	<section id="notify">
		<h2><Bell size={18} /> Notifications</h2>
		<p>
			Kick-off reminders for untipped matches, forecast deadlines, results
			recaps, pool chat and lead changes — each one can go by email, by push, or
			not at all, under <b>Settings</b>. Email only reaches verified addresses.
			Push works in the installed app and in supported browsers.
		</p>
		<p class="withicon">
			<Smartphone size={16} /> Install Matchowl on your phone from the browser
			menu ("Add to Home Screen") for the app feeling and push.
		</p>
	</section>

	<section id="account">
		<h2><UserCheck size={18} /> Account and verification</h2>
		<p>
			You can register with any email and start tipping right away. Friends,
			pools and every email wait until the address is verified — we send the
			link when you sign up, and Settings can resend it. Signing in with Google
			verifies you on the spot. Unverified accounts are removed after six
			months of silence.
		</p>
		<p>
			Change name, avatar, email or password under Settings, and delete the
			account there too — that removes your tips, forecasts and memberships.
		</p>
	</section>

	<p class="contact muted">
		<Mail size={16} /> Questions, a wrong result, an idea?
		<a href={`mailto:${appConfig.contactEmail}`}>{appConfig.contactEmail}</a>
		· <a href="/legal/terms">Terms</a> · <a href="/legal/privacy">Privacy</a> ·
		<a href="/legal/about">About</a>
	</p>
	{#if !auth.isAuthed}
		<a class="btn cta" href="/login">Sign in</a>
	{/if}
</div>

<style>
	.help {
		max-width: 640px;
		margin: 0 auto;
		padding: 1rem 1rem 3rem;
	}
	.brand {
		margin: 1rem 0 1.25rem;
	}
	h1 {
		margin: 0 0 0.35rem;
		font-size: 1.6rem;
	}
	.lead {
		margin: 0 0 1rem;
		line-height: 1.45;
	}
	.jump {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
		margin-bottom: 0.5rem;
	}
	.jump a {
		padding: 0.3rem 0.7rem;
		border-radius: 999px;
		border: 1px solid var(--border);
		color: var(--muted);
		font-size: 0.82rem;
		text-decoration: none;
	}
	.jump a:hover {
		color: var(--text);
		border-color: var(--accent);
	}
	section {
		margin-top: 1.5rem;
		scroll-margin-top: 4.5rem;
	}
	h2 {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin: 0 0 0.4rem;
		font-size: 1.1rem;
	}
	h2 :global(svg) {
		color: var(--accent);
		flex: none;
	}
	p {
		margin: 0 0 0.6rem;
		line-height: 1.5;
		font-size: 0.95rem;
	}
	.withicon {
		display: flex;
		gap: 0.5rem;
		align-items: flex-start;
	}
	.withicon :global(svg) {
		color: var(--accent);
		flex: none;
		margin-top: 0.2rem;
	}
	.pts {
		margin-top: 0.75rem;
	}
	.pts .kicker {
		margin: 0 0 0.4rem;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.92rem;
	}
	td {
		padding: 0.35rem 0;
		border-bottom: 1px solid var(--border);
	}
	td.digits {
		text-align: right;
		font-weight: 700;
		color: var(--accent);
		width: 3rem;
	}
	tr:last-child td {
		border-bottom: 0;
	}
	.small {
		font-size: 0.82rem;
		margin: 0.6rem 0 0;
	}
	.contact {
		margin-top: 2rem;
		font-size: 0.88rem;
		line-height: 1.6;
	}
	.contact :global(svg) {
		vertical-align: -3px;
		margin-right: 0.25rem;
	}
	.cta {
		display: block;
		margin-top: 1rem;
		text-align: center;
		text-decoration: none;
	}
</style>
