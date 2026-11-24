<!-- Challenge a friend on one match: pick a friend (each with your record
     against them), add an optional stake note, send. -->
<script lang="ts">
	import { pb } from '$lib/pb';
	import { api, type Person } from '$lib/api';
	import { challengesStore, recordLine, type Challenge } from '$lib/challenges.svelte';
	import Avatar from './Avatar.svelte';
	import { X } from '@lucide/svelte';

	let {
		matchId,
		title,
		taken = [],
		onClose,
		onSent
	}: {
		matchId: string;
		/** "Benevento – Empoli · Tue 24, 03:00 PM" */
		title: string;
		/** Friends I already have a challenge with on this match. */
		taken?: string[];
		onClose: () => void;
		onSent: (c: Challenge) => void;
	} = $props();

	let friends = $state<Person[]>([]);
	let loaded = $state(false);
	let pick = $state('');
	let stake = $state('');
	let busy = $state(false);
	let error = $state('');

	$effect(() => {
		api
			.friends()
			.then((r) => (friends = r.friends))
			.catch(() => {})
			.finally(() => (loaded = true));
		if (!challengesStore.loaded) challengesStore.load().catch(() => {});
	});
	let free = $derived(friends.filter((f) => !taken.includes(f.userId)));
	$effect(() => {
		if (!pick && free.length === 1) pick = free[0].userId;
	});
	const avatar = (id: string, file?: string) =>
		file ? pb.files.getURL({ id, collectionName: 'users' }, file) : null;

	async function send(e: SubmitEvent) {
		e.preventDefault();
		if (!pick) return;
		busy = true;
		error = '';
		try {
			onSent(await challengesStore.send(matchId, pick, stake.trim()));
		} catch (err) {
			const r = (err as { response?: { error?: string } })?.response;
			error = r?.error || (err instanceof Error ? err.message : 'Could not send the challenge.');
		} finally {
			busy = false;
		}
	}
	function onKey(e: KeyboardEvent) {
		if (e.key === 'Escape') onClose();
	}
</script>

<svelte:window onkeydown={onKey} />

<button class="scrim" onclick={onClose} aria-label="Close"></button>
<div class="sheet" role="dialog" aria-modal="true" aria-label="Challenge a friend">
<form class="sform" onsubmit={send}>
	<div class="grab"></div>
	<div class="shead">
		<h2>Challenge a friend</h2>
		<span class="spacer"></span>
		<button type="button" class="ibtn" onclick={onClose} aria-label="Close"><X size={18} /></button>
	</div>
	<p class="muted lead">{title}</p>

	<h3 class="gh">Friend</h3>
	{#if !loaded}
		<p class="muted lead">Loading…</p>
	{:else if !free.length}
		<p class="muted lead">
			{friends.length ? 'You already challenged all your friends on this match.' : 'Challenges are between friends.'}
			<a href="/friends" onclick={onClose}>Find friends</a>
		</p>
	{:else}
		<div class="group" role="radiogroup" aria-label="Friend">
			{#each free as f (f.userId)}
				{@const line = recordLine(challengesStore.records[f.userId], f.name)}
				<label class="row" class:on={pick === f.userId}>
					<Avatar name={f.name} id={f.userId} preset={f.avatarPreset} src={avatar(f.userId, f.avatar)} size={28} />
					<span class="name"><b>{f.name}</b><span class="muted">{line || 'no challenges yet'}</span></span>
					<input type="radio" name="friend" value={f.userId} bind:group={pick} />
				</label>
			{/each}
		</div>
	{/if}

	<h3 class="gh">Stake · optional</h3>
	<input class="input" maxlength="140" placeholder="Loser buys the next round" bind:value={stake} />
	<p class="muted hint">Just a note: Matchowl shows it, never collects it. Your picks stay hidden from each other until kick-off.</p>

	{#if error}<p class="error">{error}</p>{/if}
	<button class="btn" disabled={busy || !pick}>{busy ? 'Sending…' : 'Send challenge'}</button>
</form>
</div>

<style>
	.scrim {
		position: fixed;
		inset: 0;
		z-index: 60;
		border: none;
		padding: 0;
		background: rgba(0, 0, 0, 0.55);
		cursor: default;
	}
	.sheet {
		position: fixed;
		left: 0;
		right: 0;
		bottom: 0;
		z-index: 61;
		max-width: 560px;
		max-height: calc(100dvh - var(--topbar-h) - 2rem);
		overflow-y: auto;
		margin: 0 auto;
		padding: 0.6rem 1rem calc(1.2rem + env(safe-area-inset-bottom));
		background: var(--surface);
		border: 1px solid var(--border);
		border-bottom: none;
		border-radius: var(--radius) var(--radius) 0 0;
		box-shadow: 0 -18px 44px -12px rgba(0, 0, 0, 0.8);
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
	}
	.sform {
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
	}
	.grab {
		width: 36px;
		height: 4px;
		border-radius: 2px;
		background: var(--border);
		margin: 0 auto 0.2rem;
	}
	.shead {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.shead h2 {
		font-size: 1.25rem;
	}
	.spacer {
		flex: 1;
	}
	.ibtn {
		background: none;
		border: 0;
		color: var(--muted);
		cursor: pointer;
		display: inline-flex;
		padding: 0.3rem;
	}
	.lead {
		font-size: 0.85rem;
		margin: 0 0 0.3rem;
	}
	.gh {
		font-size: 0.68rem;
		font-weight: 700;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--muted);
		margin: 0.5rem 0.1rem 0.1rem;
	}
	.group {
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		overflow: hidden;
	}
	.row {
		display: flex;
		align-items: center;
		gap: 0.7rem;
		padding: 0.6rem 0.8rem;
		border-bottom: 1px solid var(--border);
		cursor: pointer;
	}
	.row:last-child {
		border-bottom: none;
	}
	.row.on {
		background: color-mix(in srgb, var(--accent) 8%, transparent);
	}
	.name {
		display: flex;
		flex-direction: column;
		gap: 0.05rem;
		flex: 1;
		min-width: 0;
		font-size: 0.92rem;
	}
	.name .muted {
		font-size: 0.78rem;
	}
	.row input {
		accent-color: var(--accent);
		width: 18px;
		height: 18px;
	}
	.hint {
		font-size: 0.78rem;
		margin: 0 0.1rem 0.4rem;
	}
	.btn {
		margin-top: 0.3rem;
	}
	@media (min-width: 900px) {
		.sheet {
			inset: 50% auto auto 50%;
			transform: translate(-50%, -50%);
			width: min(520px, calc(100vw - 2rem));
			max-height: calc(100dvh - 4rem);
			padding: 1.25rem 1.4rem 1.4rem;
			border-bottom: 1px solid var(--border);
			border-radius: var(--radius);
			box-shadow: var(--shadow-pop);
		}
		.grab {
			display: none;
		}
	}
</style>
