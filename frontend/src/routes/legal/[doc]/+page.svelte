<script lang="ts">
	// Terms of use, privacy notice, imprint. Public (reachable signed out,
	// linked from register and the sign-in footer). The final texts belong on
	// the marketing site once it exists; these are the in-app copies and are
	// DRAFTS until floholz has signed them off.
	import { page } from '$app/stores';
	import { appConfig } from '$lib/appconfig.svelte';
	import { pageChrome } from '$lib/shell.svelte';
	import { TERMS_VERSION } from '$lib/legal';
	import { onMount } from 'svelte';
	import Logo from '$lib/components/Logo.svelte';
	import { auth } from '$lib/auth.svelte';

	onMount(() => appConfig.load());

	const titles: Record<string, string> = {
		terms: 'Terms of use',
		privacy: 'Privacy notice',
		imprint: 'Imprint'
	};
	let doc = $derived($page.params.doc ?? '');
	let title = $derived(titles[doc] ?? 'Legal');
	pageChrome(() => ({ title, back: '/settings' }));
</script>

<div class="legal">
	{#if !auth.isAuthed}
		<div class="brand"><Logo size={36} /></div>
	{/if}
	<nav class="tabs" aria-label="Legal pages">
		<a class:on={doc === 'terms'} href="/legal/terms">Terms</a>
		<a class:on={doc === 'privacy'} href="/legal/privacy">Privacy</a>
		<a class:on={doc === 'imprint'} href="/legal/imprint">Imprint</a>
	</nav>

	{#if doc === 'terms'}
		<h1>Terms of use</h1>
		<p class="muted small">Version {TERMS_VERSION} · draft</p>
		<h2>What Matchowl is</h2>
		<p>
			Matchowl is a free football prediction game. You tip match scores, place a
			forecast per season and compare yourself with friends. It is a hobby
			project run by a private person, without any commercial intent, betting
			or prizes.
		</p>
		<h2>Your account</h2>
		<p>
			You need an account to play. Keep your sign-in details to yourself and
			tell us if you think someone else is using them. One person, one account.
			You can delete your account at any time under Settings; this removes your
			tips, forecasts and memberships.
		</p>
		<h2>Fair play</h2>
		<p>
			Pick a display name that does not impersonate someone else and is not
			offensive. Pool chats are private among their members; still, no
			harassment, no illegal content. We may remove content or accounts that
			break this.
		</p>
		<h2>Availability</h2>
		<p>
			Matchowl is offered as it is. Match data comes from third-party sources
			and may be late or wrong; points are recomputed when results are
			corrected. There is no guarantee of uptime, and the service may change or
			end at any time.
		</p>
		<h2>Changes</h2>
		<p>
			When these terms change in a way that matters, you will be asked to
			accept them again on your next visit.
		</p>
	{:else if doc === 'privacy'}
		<h1>Privacy notice</h1>
		<p class="muted small">Version {TERMS_VERSION} · draft</p>
		<h2>What we store</h2>
		<ul>
			<li><b>Account:</b> email address, display name, optional avatar, password hash or the Google account link, and which terms version you accepted.</li>
			<li><b>Play:</b> your tips, forecasts, the competitions you play, points and placings.</li>
			<li><b>Social:</b> friendships, pool memberships, pool invites and the messages you post in pool chats.</li>
			<li><b>Notifications:</b> your notification preferences and, if you enable push, the push subscription of that device.</li>
			<li><b>Technical:</b> server logs with IP address and requested URL, kept for a short time for security and debugging.</li>
		</ul>
		<h2>What we do with it</h2>
		<p>
			Only run the game: show boards to the people in your pools and to your
			friends, send the emails and pushes you have switched on, and keep the
			service working. We do not sell data, show ads or track you across other
			sites.
		</p>
		<h2>Email</h2>
		<p>
			We send account mails you request (verification, password reset, email
			change) and, only to verified addresses, the notifications you enable
			under Settings. Mail is delivered through a transactional mail provider
			acting on our behalf.
		</p>
		<h2>Who sees what</h2>
		<p>
			Your display name, avatar and points are visible to members of your pools
			and to your friends, and on the everyone board once your address is
			verified. Your email address is never shown to other players.
		</p>
		<h2>Your rights</h2>
		<p>
			You can view and change your data under Settings and delete your account
			there. For anything else, or to exercise your rights under the GDPR,
			write to <a href={`mailto:${appConfig.contactEmail}`}>{appConfig.contactEmail}</a>.
		</p>
	{:else if doc === 'imprint'}
		<h1>Imprint</h1>
		<p class="muted small">Draft — the operator's name and postal address go here (required for a German-facing service).</p>
		<p>
			Matchowl is run by <b>[Name]</b><br />
			[Street and number]<br />
			[Postcode and city]<br />
			[Country]
		</p>
		<p>
			Contact: <a href={`mailto:${appConfig.contactEmail}`}>{appConfig.contactEmail}</a>
		</p>
		<p class="muted small">
			Match data from third-party sources. Team names and crests belong to
			their owners. Matchowl is not affiliated with any league or federation.
		</p>
	{:else}
		<h1>Not found</h1>
		<p class="muted">Pick a page above.</p>
	{/if}
</div>

<style>
	.legal {
		max-width: 640px;
		margin: 0 auto;
		padding: 1rem 1rem 3rem;
	}
	.brand {
		margin: 1rem 0 1.25rem;
	}
	.tabs {
		display: flex;
		gap: 0.4rem;
		margin-bottom: 1.25rem;
	}
	.tabs a {
		padding: 0.35rem 0.75rem;
		border-radius: 999px;
		border: 1px solid var(--border);
		color: var(--muted);
		font-size: 0.85rem;
		text-decoration: none;
	}
	.tabs a.on {
		color: var(--accent-fg);
		background: var(--accent);
		border-color: var(--accent);
	}
	h1 {
		margin: 0 0 0.25rem;
		font-size: 1.6rem;
	}
	h2 {
		margin: 1.4rem 0 0.35rem;
		font-size: 1.05rem;
	}
	p,
	li {
		line-height: 1.5;
		font-size: 0.95rem;
	}
	p {
		margin: 0 0 0.6rem;
	}
	ul {
		padding-left: 1.2rem;
		margin: 0 0 0.6rem;
	}
	.small {
		font-size: 0.82rem;
	}
</style>
