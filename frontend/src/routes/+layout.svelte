<script lang="ts">
	import '../app.css';
	import { auth } from '$lib/auth.svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import Logo from '$lib/components/Logo.svelte';
	import UserMenu from '$lib/components/UserMenu.svelte';
	import NavLinks from '$lib/components/NavLinks.svelte';
	import PwaInstallButton from '$lib/components/PwaInstallButton.svelte';
	import PwaInstallBanner from '$lib/components/PwaInstallBanner.svelte';
	import NotifyAnnounce from '$lib/components/NotifyAnnounce.svelte';
	import VerifyEmailAnnounce from '$lib/components/VerifyEmailAnnounce.svelte';
	import AnnounceBanner from '$lib/components/AnnounceBanner.svelte';
	import { serverClock } from '$lib/serverclock.svelte';
	import { theme } from '$lib/theme.svelte';
	import { shell } from '$lib/shell.svelte';
	import '$lib/keyboard'; // tracks the on-screen keyboard → `kb-open` class + `--kb` var
	import { CircleHelp, ChevronLeft } from '@lucide/svelte';

	let { children } = $props();

	// Apply the saved (or device) theme before anything renders.
	theme.init();

	// Pull the (possibly simulated) server clock once so lock checks and the
	// dev-tools link are correct app-wide.
	$effect(() => {
		if (auth.isAuthed && !serverClock.loaded) serverClock.refresh();
	});

	// Signed-out-only pages — visible to anonymous users; signed-in users
	// bounce away to home (or /join if an invite is attached).
	const authPages = ['/login', '/register', '/forgot-password'];
	let path = $derived($page.url.pathname);
	let isAuthPage = $derived(authPages.includes(path));
	// Public routes — anyone can land here regardless of auth state:
	//   /join/<code>                 invite landing
	//   /confirm-password-reset/<t>  email reset target (must work even for
	//                                a still-signed-in user whose token was
	//                                requested by someone with their email)
	//   /confirm-verification/<t>    email verification target (same reasoning)
	//   /confirm-email-change/<t>    email-change target (lands on the NEW
	//                                address, possibly on a signed-out device)
	//   /welcome                     chrome-less landing/help page (any auth state)
	let isPublic = $derived(
		path.startsWith('/join') ||
			path.startsWith('/confirm-password-reset/') ||
			path.startsWith('/confirm-verification/') ||
			path.startsWith('/confirm-email-change/') ||
			path === '/welcome'
	);
	// The home route doubles as the public landing page for signed-out
	// visitors (app home once authed) — never bounce anon users away from it.
	let isLanding = $derived(path === '/');
	// No app chrome on the standalone auth / invite / reset screens.
	let chrome = $derived(auth.isAuthed && !isAuthPage && !isPublic);

	// SPA auth guard.
	$effect(() => {
		const invite = $page.url.searchParams.get('invite');
		if (!auth.isAuthed && !isAuthPage && !isPublic && !isLanding) {
			goto('/login', { replaceState: true });
		}
		// Already signed in: skip the auth pages. If they arrived via an
		// invite, send them to the join flow so it auto-joins.
		if (auth.isAuthed && isAuthPage) {
			goto(invite ? `/join/${invite}` : '/', { replaceState: true });
		}
	});
</script>

{#if chrome}
	<!-- Top header. Mobile: contextual — the page's title (or back +
	     title + context) left, avatar right; Home shows the wordmark.
	     Desktop: wordmark + nav links, always. -->
	<header class="topbar" class:titled={!!shell.title || !!shell.back}>
		<div class="topbar-brand"><Logo /></div>
		<div class="topbar-ctx" class:withback={!!shell.back}>
			{#if shell.back}
				<a class="topbar-back" href={shell.back} aria-label="Back"><ChevronLeft size={22} /></a>
			{/if}
			<div class="topbar-ttl">
				<span class="ttl">{shell.title}</span>
				{#if shell.context}<span class="ctx muted">{shell.context}</span>{/if}
			</div>
		</div>
		<nav class="topbar-links"><NavLinks variant="top" /></nav>
		<div class="spacer"></div>
		<a class="topbar-help" href="/welcome" aria-label="What is Matchowl?">
			<CircleHelp size={20} />
		</a>
		<PwaInstallButton />
		<UserMenu align="right" />
	</header>

	<!-- Mobile: bottom tab bar -->
	<nav class="tabbar"><NavLinks variant="tab" /></nav>
{/if}

<div class="app-shell" class:with-chrome={chrome} class:wide={shell.wide}>
	{#if chrome}
		<PwaInstallBanner />
		<VerifyEmailAnnounce />
		<NotifyAnnounce />
		<AnnounceBanner />
	{/if}
	{@render children()}
</div>
