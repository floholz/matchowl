<!-- One card per competition per day: crest + name + round in the header
     with the "Score / Tip" column captions, match rows inside. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { ChevronRight } from '@lucide/svelte';
	let {
		name,
		round = '',
		logo = '',
		href = '',
		captions = true,
		children
	}: {
		name: string;
		round?: string;
		/** Competition crest URL; falls back to the name's initials. */
		logo?: string;
		href?: string;
		captions?: boolean;
		children: Snippet;
	} = $props();
	let initials = $derived(
		name
			.split(/\s+/)
			.filter((w) => /^[A-Z0-9]/.test(w))
			.map((w) => w[0])
			.join('')
			.slice(0, 3)
	);
</script>

<div class="mg card">
	<svelte:element this={href ? 'a' : 'div'} class="head" href={href || undefined}>
		<span class="crest" class:ph={!logo}>
			{#if logo}<img src={logo} alt="" />{:else}{initials}{/if}
		</span>
		<span class="txt">
			<b>{name}</b>
			{#if round}<span class="round">{round}</span>{/if}
		</span>
		{#if captions}
			<span class="cols" aria-hidden="true"><span>Score</span><span>Tip</span><span></span></span>
		{:else if href}
			<ChevronRight size={16} class="cv" />
		{/if}
	</svelte:element>
	{@render children()}
</div>

<style>
	.mg {
		padding: 0;
		margin-bottom: 12px;
	}
	.head {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 10px 12px 10px 14px;
		border-bottom: 1px solid var(--border);
		color: var(--text);
	}
	a.head:hover b {
		color: var(--accent);
	}
	.crest {
		width: 22px;
		height: 22px;
		flex: none;
		border-radius: 50%;
		overflow: hidden;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		background: #fff;
	}
	.crest img {
		width: 100%;
		height: 100%;
		object-fit: contain;
	}
	.crest.ph {
		background: var(--surface-2);
		color: var(--muted);
		font-size: 8px;
		font-weight: 800;
		letter-spacing: 0.02em;
	}
	.txt {
		display: flex;
		flex-direction: column;
		gap: 1px;
		min-width: 0;
	}
	.txt b {
		font-size: 13px;
		font-weight: 700;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.round {
		font-size: 12px;
		color: var(--muted);
	}
	.cols {
		margin-left: auto;
		display: flex;
		font-size: 9px;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.cols span {
		width: 40px;
		text-align: center;
	}
	.cols span:last-child {
		width: 34px;
	}
	.head :global(.cv) {
		margin-left: auto;
		color: var(--muted);
	}
</style>
