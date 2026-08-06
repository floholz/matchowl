<script lang="ts">
	import { page } from '$app/stores';
	import { navItems, isActive } from '$lib/nav';

	let { variant = 'tab' as 'tab' | 'top' } = $props();
	let path = $derived($page.url.pathname);
</script>

<div class="links {variant}">
	{#each navItems as it (it.href)}
		{@const Icon = it.icon}
		<a href={it.href} class:active={isActive(it.href, path)}>
			<Icon size={variant === 'top' ? 18 : 22} />
			<span>{it.label}</span>
		</a>
	{/each}
</div>

<style>
	.links {
		display: flex;
	}
	.links a {
		display: flex;
		align-items: center;
		color: var(--muted);
		position: relative;
		transition: color 0.15s ease;
	}
	.links a span {
		font-weight: 700;
		letter-spacing: 0.04em;
		text-transform: uppercase;
	}
	.links a.active {
		color: var(--accent);
	}

	/* Mobile bottom tab bar */
	.tab {
		flex: 1;
	}
	.tab a {
		flex: 1;
		flex-direction: column;
		justify-content: center;
		gap: 4px;
		font-size: 0.6rem;
		padding: 0.4rem 0;
	}
	.tab a.active::before {
		content: '';
		position: absolute;
		top: 0;
		left: 50%;
		transform: translateX(-50%);
		width: 26px;
		height: 3px;
		border-radius: 0 0 3px 3px;
		background: var(--accent);
		box-shadow: 0 0 12px 1px var(--accent);
	}

	/* Desktop top navbar */
	.top {
		gap: 0.4rem;
	}
	.top a {
		gap: 0.5rem;
		padding: 0.45rem 0.9rem;
		font-size: 0.9rem;
		border-radius: var(--radius-pill);
	}
	.top a span {
		font-size: 0.8rem;
	}
	.top a:hover {
		color: var(--text);
	}
	.top a.active {
		color: var(--accent);
		background: color-mix(in srgb, var(--accent) 12%, transparent);
	}
</style>
