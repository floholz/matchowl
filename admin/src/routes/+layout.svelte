<script lang="ts">
	import '../app.css';
	import { auth } from '$lib/auth.svelte';
	import { appConfig } from '$lib/appconfig.svelte';
	import { serverClock } from '$lib/serverclock.svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import Avatar from '$lib/components/Avatar.svelte';
	import {
		LayoutDashboard,
		Trophy,
		Shield,
		RefreshCw,
		Users,
		Megaphone,
		Bell,
		Mail,
		FlaskConical,
		ExternalLink,
		LogOut,
		Menu,
		X
	} from '@lucide/svelte';

	let { children } = $props();

	let path = $derived($page.url.pathname);
	let isLogin = $derived(path === '/login');

	// Guard: the admin app is for admins (owner inherits). Everyone else
	// gets the sign-in, or a plain "not an admin" note with a way out.
	$effect(() => {
		if (!auth.isAuthed && !isLogin) goto('/login', { replaceState: true });
		if (auth.isAuthed && isLogin) goto('/', { replaceState: true });
	});
	$effect(() => {
		if (auth.isAuthed) {
			appConfig.load();
			if (!serverClock.loaded) serverClock.refresh();
		}
	});

	const nav = $derived([
		{ href: '/', label: 'Dashboard', icon: LayoutDashboard },
		{ href: '/competitions', label: 'Competitions', icon: Trophy },
		{ href: '/teams', label: 'Teams', icon: Shield },
		{ href: '/sync', label: 'Sync', icon: RefreshCw },
		{ href: '/people', label: 'People', icon: Users },
		{ href: '/announcements', label: 'Announcements', icon: Megaphone },
		{ href: '/mailings', label: 'Mailings', icon: Mail },
		{ href: '/notifications', label: 'Notifications', icon: Bell },
		...(serverClock.dev ? [{ href: '/dev', label: 'Dev', icon: FlaskConical }] : [])
	]);
	const active = (href: string) => (href === '/' ? path === '/' : path.startsWith(href));

	let menuOpen = $state(false);
	function logout() {
		auth.logout();
		goto('/login', { replaceState: true });
	}
</script>

{#if !auth.isAuthed}
	{@render children()}
{:else if !auth.isAdmin}
	<div class="gate">
		<div class="card">
			<h2>Not an admin</h2>
			<p class="muted">You are signed in as {auth.user?.email}, which is not an admin account.</p>
			<button class="btn secondary" onclick={logout}><LogOut size={16} /> Sign out</button>
		</div>
	</div>
{:else}
	<div class="shell" class:open={menuOpen}>
		<aside class="side">
			<a class="brand" href="/">
				<img src="/favicon.svg" alt="" width="28" height="28" />
				<span>Matchowl <b>Admin</b></span>
			</a>
			<nav class="nav">
				{#each nav as n (n.href)}
					<a class="ni" class:on={active(n.href)} href={n.href} onclick={() => (menuOpen = false)}>
						<n.icon size={18} />
						<span>{n.label}</span>
					</a>
				{/each}
			</nav>
			<div class="side-foot">
				{#if appConfig.appUrl}
					<a class="ni" href={appConfig.appUrl} target="_blank" rel="noopener"><ExternalLink size={18} /><span>Open the app</span></a>
				{/if}
				<a class="ni" href="/_/" target="_blank" rel="noopener"><ExternalLink size={18} /><span>PocketBase</span></a>
				<div class="me">
					<Avatar name={auth.user?.name ?? '?'} id={auth.user?.id} preset={auth.user?.avatarPreset} src={auth.user?.avatarUrl} size={30} />
					<span class="who"><b>{auth.user?.name}</b><small class="muted">{auth.user?.role}</small></span>
					<button class="iconb" title="Sign out" aria-label="Sign out" onclick={logout}><LogOut size={16} /></button>
				</div>
				{#if appConfig.version}<div class="ver muted">v{appConfig.version}</div>{/if}
			</div>
		</aside>
		<header class="mobbar">
			<button class="iconb" aria-label={menuOpen ? 'Close menu' : 'Menu'} onclick={() => (menuOpen = !menuOpen)}>
				{#if menuOpen}<X size={20} />{:else}<Menu size={20} />{/if}
			</button>
			<span class="mtitle">Matchowl <b>Admin</b></span>
		</header>
		<main class="main">
			{@render children()}
		</main>
	</div>
{/if}

<style>
	.gate {
		min-height: 100dvh;
		display: grid;
		place-items: center;
		padding: 1rem;
	}
	.gate .card {
		max-width: 420px;
		display: flex;
		flex-direction: column;
		gap: 0.8rem;
	}
	.shell {
		display: grid;
		grid-template-columns: 240px 1fr;
		min-height: 100dvh;
	}
	.side {
		position: sticky;
		top: 0;
		height: 100dvh;
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
		padding: 1rem 0.8rem;
		border-right: 1px solid var(--border);
		background: var(--surface);
	}
	.brand {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.3rem 0.5rem 0.8rem;
		color: var(--text);
		font-family: var(--font-display);
		font-size: 1.15rem;
	}
	.brand b {
		color: var(--accent);
	}
	.nav {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}
	.ni {
		display: flex;
		align-items: center;
		gap: 0.65rem;
		padding: 0.6rem 0.7rem;
		border-radius: var(--radius-sm);
		color: var(--muted);
		font-weight: 600;
		font-size: 0.92rem;
	}
	.ni:hover {
		background: var(--surface-2);
		color: var(--text);
	}
	.ni.on {
		background: color-mix(in srgb, var(--accent) 14%, transparent);
		color: var(--accent);
	}
	.side-foot {
		margin-top: auto;
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}
	.me {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.7rem 0.5rem 0.2rem;
		border-top: 1px solid var(--border);
		margin-top: 0.4rem;
	}
	.who {
		display: flex;
		flex-direction: column;
		min-width: 0;
		font-size: 0.85rem;
		line-height: 1.15;
	}
	.who b {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.who small {
		font-size: 0.7rem;
		text-transform: uppercase;
		letter-spacing: 0.08em;
	}
	.iconb {
		margin-left: auto;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 34px;
		height: 34px;
		border-radius: 50%;
		border: 1px solid var(--border);
		background: transparent;
		color: var(--muted);
		cursor: pointer;
	}
	.iconb:hover {
		color: var(--text);
		background: var(--surface-2);
	}
	.ver {
		font-size: 0.7rem;
		padding: 0.3rem 0.5rem 0;
	}
	.mobbar {
		display: none;
	}
	.main {
		min-width: 0;
		padding: 1.5rem 2rem 4rem;
		max-width: 1280px;
	}
	@media (max-width: 899px) {
		.shell {
			/* One column: the bar, then the page. Rows must not stretch to
			   fill the viewport, or the bar grows on short pages. */
			grid-template-columns: 1fr;
			grid-template-rows: auto 1fr;
			align-content: start;
		}
		.side {
			position: fixed;
			inset: 0 auto 0 0;
			width: 260px;
			z-index: 30;
			transform: translateX(-100%);
			transition: transform 0.2s ease;
		}
		.shell.open .side {
			transform: none;
			box-shadow: var(--shadow-pop);
		}
		.mobbar {
			display: flex;
			align-items: center;
			gap: 0.7rem;
			padding: 0.6rem 0.9rem;
			border-bottom: 1px solid var(--border);
			background: var(--surface);
			position: sticky;
			top: 0;
			z-index: 20;
		}
		.mobbar .iconb {
			margin-left: 0;
		}
		.mtitle {
			font-family: var(--font-display);
			font-size: 1.05rem;
		}
		.mtitle b {
			color: var(--accent);
		}
		.main {
			padding: 1rem 1rem 3rem;
		}
	}
</style>
