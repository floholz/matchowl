<!-- The tip capsule: the user's own numbers in the app's mono, in an
     outlined orange capsule that sits in the same column on every screen.
     Dashed + plus = open and untipped; solid = tipped; after full time it
     stays orange on a hit and goes grey on a miss. The LED dot marks the
     user's advancer pick on a knockout (a drawn tip + dot = penalty pick). -->
<script lang="ts">
	import { Plus, Lock, ShieldCheck, Ban } from '@lucide/svelte';
	import type { H2HPickMark } from '$lib/api';
	let {
		home = null,
		away = null,
		state = 'open',
		advancer = '',
		active = false,
		onclick = undefined,
		marks = null
	}: {
		/** Head-to-head marks: a shield on the top corner for a Save Call
		 *  (mine in orange, the rival's revealed one in grey), a ban on the
		 *  bottom corner (outlined = mine on the rival, filled red = the
		 *  rival's on me, revealed at kick-off). */
		marks?: H2HPickMark | null;
		home?: number | null;
		away?: number | null;
		/** open: editable (digits, or a plus when empty) · frozen: locked with
		 *  a tip, not scored yet · locked: locked without a tip · hit / miss:
		 *  scored · unavailable: not tippable yet (pairing undecided). */
		state?: 'open' | 'frozen' | 'locked' | 'hit' | 'miss' | 'unavailable';
		advancer?: '' | 'home' | 'away';
		/** The row's editor is open on this tip. */
		active?: boolean;
		onclick?: () => void;
	} = $props();
	let has = $derived(home !== null && away !== null);
	let markText = $derived(
		[
			marks?.saved ? 'your Save Call' : marks?.rivalSaved ? "your rival's Save Call" : '',
			marks?.rivalBanned ? 'banned for you' : marks?.banned ? 'your ban on your rival' : ''
		]
			.filter(Boolean)
			.join(', ')
	);
	let label = $derived(
		(state === 'unavailable'
			? 'Not tippable yet'
			: !has
				? state === 'open'
					? 'Place a tip'
					: 'No tip'
				: `Your tip ${home}–${away}${state === 'open' ? ', tap to change' : ''}`) +
			(markText ? ` · ${markText}` : '')
	);
</script>

{#snippet inner()}
	{#if marks?.saved || marks?.rivalSaved}
		<span class="bd save" class:rival={!marks.saved}><ShieldCheck size={9} /></span>
	{/if}
	{#if marks?.banned || marks?.rivalBanned}
		<span class="bd ban" class:onme={marks.rivalBanned}><Ban size={9} /></span>
	{/if}
	{#if state === 'unavailable'}
		<Lock size={14} />
	{:else if !has}
		{#if state === 'open'}<Plus size={18} />{:else}<span class="dash">—</span>{/if}
	{:else}
		{#if advancer === 'home'}<span class="mk h"></span>{/if}
		{#if advancer === 'away'}<span class="mk a"></span>{/if}
		<span class="n">{home}</span>
		<span class="n">{away}</span>
	{/if}
{/snippet}

{#if onclick && state === 'open'}
	<button type="button" class="cap {state}" class:empty={!has} class:active {onclick} aria-label={label}>
		{@render inner()}
	</button>
{:else}
	<span class="cap {state}" class:empty={!has} role="img" aria-label={label}>{@render inner()}</span>
{/if}

<style>
	.cap {
		position: relative;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 1px;
		flex: none;
		width: 40px;
		height: 46px;
		padding: 0;
		margin: 0;
		border: 1.5px solid var(--accent);
		border-radius: 12px;
		background: color-mix(in srgb, var(--accent) 10%, transparent);
		color: var(--text);
		font: inherit;
		cursor: default;
	}
	button.cap {
		cursor: pointer;
		transition:
			background 0.15s ease,
			box-shadow 0.15s ease;
	}
	button.cap:hover,
	.cap.active {
		box-shadow: var(--glow);
	}
	.n {
		height: 19px;
		display: flex;
		align-items: center;
		font-family: var(--font-mono);
		font-weight: 700;
		font-size: 14px;
		font-variant-numeric: tabular-nums;
	}
	.cap.empty.open {
		border-style: dashed;
		border-color: color-mix(in srgb, var(--accent) 60%, transparent);
		background: transparent;
		color: var(--accent);
	}
	.cap.locked,
	.cap.unavailable {
		border-style: dashed;
		border-color: var(--border);
		background: transparent;
		color: var(--muted);
	}
	.cap.miss {
		border-color: var(--border);
		background: var(--surface-2);
	}
	.cap.miss .n {
		color: var(--muted);
	}
	.cap.hit {
		background: color-mix(in srgb, var(--accent) 14%, transparent);
	}
	.cap.frozen {
		border-color: color-mix(in srgb, var(--accent) 70%, var(--border));
	}
	.dash {
		font-weight: 700;
	}
	.mk {
		position: absolute;
		left: 4px;
		width: 4px;
		height: 4px;
		border-radius: 50%;
		background: var(--accent);
		box-shadow: 0 0 5px var(--accent);
	}
	.miss .mk {
		background: var(--muted);
		box-shadow: none;
	}
	.mk.h {
		top: 11px;
	}
	.mk.a {
		top: 31px;
	}
	/* Head-to-head badges on the capsule's right corners. */
	.bd {
		position: absolute;
		right: -6px;
		width: 15px;
		height: 15px;
		border-radius: 50%;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		box-sizing: border-box;
		border: 1.5px solid var(--surface);
		z-index: 1;
	}
	.bd.save {
		top: -6px;
		background: var(--accent);
		color: var(--accent-fg);
	}
	.bd.save.rival {
		background: var(--muted);
		color: var(--surface);
	}
	.bd.ban {
		bottom: -6px;
		background: var(--surface);
		border-color: var(--live, #e0443e);
		color: var(--live, #e0443e);
	}
	.bd.ban.onme {
		background: var(--live, #e0443e);
		border-color: var(--surface);
		color: #fff;
	}
</style>
