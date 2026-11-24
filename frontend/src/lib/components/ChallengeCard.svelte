<!-- One challenge from my side: who, the match (unless the page is the
     match), the stake, and what's next — accept or decline an invite,
     withdraw my own, who has tipped (picks stay hidden until kick-off),
     both tips once it's on, and the result once it's final. -->
<script lang="ts">
	import { pb } from '$lib/pb';
	import { serverClock } from '$lib/serverclock.svelte';
	import { challengesStore, recordLine, type Challenge, type ChallengeTip } from '$lib/challenges.svelte';
	import Avatar from './Avatar.svelte';
	import { Check, Swords } from '@lucide/svelte';

	let {
		c,
		showMatch = true,
		onChange
	}: {
		c: Challenge;
		/** Show the match line (off on the match page itself). */
		showMatch?: boolean;
		onChange?: (c: Challenge) => void;
	} = $props();

	let busy = $state(false);
	let error = $state('');
	const avatar = (id: string, file?: string) =>
		file ? pb.files.getURL({ id, collectionName: 'users' }, file) : null;
	const name = $derived(c.them.name);
	const record = $derived(recordLine(challengesStore.records[c.them.userId], name));
	const kickedOff = $derived(
		c.settled || c.match.status !== 'scheduled' || new Date(c.match.kickoff).getTime() <= serverClock.now()
	);
	const matchLine = $derived(`${c.match.home.name} – ${c.match.away.name}`);
	const when = $derived.by(() => {
		const d = new Date(c.match.kickoff);
		return `${d.toLocaleDateString(undefined, { weekday: 'short' })} ${d.getDate()} · ${d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })}`;
	});

	let title = $derived.by(() => {
		if (c.settled) {
			if (c.outcome === 'won') return `You beat ${name}`;
			if (c.outcome === 'lost') return `${name} beat you`;
			return `A draw with ${name}`;
		}
		if (c.status === 'pending') return c.mine ? `You challenged ${name}` : `${name} challenges you`;
		return `You vs ${name}`;
	});
	let pill = $derived.by((): { text: string; cls: string } | null => {
		if (c.settled) return { text: `${c.myPts}–${c.theirPts}`, cls: c.outcome === 'won' ? 'ok' : '' };
		if (c.status === 'expired') return { text: 'expired', cls: '' };
		if (c.status === 'pending') return c.mine ? { text: 'waiting', cls: '' } : null;
		if (c.status === 'accepted' && !kickedOff)
			return c.myTip && c.theirTipped ? { text: 'both tipped', cls: 'ok' } : { text: 'accepted', cls: 'ok' };
		if (c.status === 'accepted') return { text: c.match.status === 'live' ? 'live' : 'on', cls: c.match.status === 'live' ? 'live' : '' };
		return null;
	});
	const tipText = (t: ChallengeTip | null | undefined) => (t ? `${t.ftHome}:${t.ftAway}` : '–');

	async function act(action: 'accept' | 'decline' | 'cancel') {
		busy = true;
		error = '';
		try {
			onChange?.(await challengesStore.answer(c.id, action));
		} catch (e) {
			const r = (e as { response?: { error?: string } })?.response;
			error = r?.error || (e instanceof Error ? e.message : 'Something went wrong.');
		} finally {
			busy = false;
		}
	}
</script>

<div class="card ch" class:incoming={c.status === 'pending' && !c.mine}>
	<div class="head">
		<Avatar name={c.them.name} id={c.them.userId} preset={c.them.avatarPreset} src={avatar(c.them.userId, c.them.avatar)} size={32} />
		<span class="txt">
			<b>{title}</b>
			<span class="muted">
				{#if showMatch}<a href={`/m/${c.match.id}`}>{matchLine}</a> · {when}{:else}{record || (c.mine ? 'sent by you' : `sent by ${name}`)}{/if}
			</span>
		</span>
		{#if pill}<span class="pill {pill.cls}">{pill.text}</span>{/if}
	</div>

	{#if c.stake}
		<div class="stake"><span class="lbl">Stake</span><span>“{c.stake}”</span></div>
	{/if}

	{#if error}<p class="error small">{error}</p>{/if}

	{#if c.status === 'pending' && !c.mine}
		<div class="foot">
			<button class="btn sm" disabled={busy} onclick={() => act('accept')}><Swords size={15} /> Accept</button>
			<button class="btn secondary sm" disabled={busy} onclick={() => act('decline')}>Decline</button>
			<span class="muted note">accept before kick-off</span>
		</div>
	{:else if c.status === 'pending' && c.mine}
		<div class="foot">
			<span class="muted note">Waiting for {name} to accept before kick-off.</span>
			<button class="btn ghost sm" disabled={busy} onclick={() => act('cancel')}>Withdraw</button>
		</div>
	{:else if c.status === 'accepted' && !kickedOff}
		<div class="foot tips">
			<span class:done={!!c.myTip}>{#if c.myTip}<Check size={14} /> You tipped{:else}You haven't tipped{/if}</span>
			<span class:done={c.theirTipped}>{#if c.theirTipped}<Check size={14} /> {name} tipped{:else}{name} hasn't tipped{/if}</span>
			<span class="muted">· hidden till kick-off</span>
		</div>
	{:else if c.status === 'accepted'}
		<div class="foot tips">
			<span>You <b class="digits">{tipText(c.myTip)}</b></span>
			<span>{name} <b class="digits">{tipText(c.theirTip)}</b></span>
			{#if c.settled && record}<span class="muted">· {record}</span>{/if}
		</div>
	{/if}
</div>

<style>
	.ch {
		padding: 0.85rem 0.9rem;
		display: flex;
		flex-direction: column;
		gap: 0.65rem;
	}
	.ch.incoming {
		border-color: color-mix(in srgb, var(--accent) 35%, var(--border));
	}
	.head {
		display: flex;
		align-items: center;
		gap: 0.7rem;
	}
	.txt {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		min-width: 0;
		flex: 1;
		font-size: 0.92rem;
	}
	.txt .muted {
		font-size: 0.8rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.txt .muted a {
		color: inherit;
	}
	.stake {
		display: flex;
		align-items: baseline;
		gap: 0.6rem;
		padding: 0.5rem 0.7rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: color-mix(in srgb, var(--bg) 50%, transparent);
		font-size: 0.88rem;
	}
	.stake .lbl {
		font-size: 0.66rem;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--muted);
		flex: none;
	}
	.foot {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-wrap: wrap;
	}
	.foot .note {
		font-size: 0.78rem;
	}
	.foot .btn.ghost {
		margin-left: auto;
	}
	.tips {
		font-size: 0.84rem;
		gap: 0.4rem 0.9rem;
	}
	.tips span {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
	}
	.tips .done {
		color: var(--success);
		font-weight: 700;
	}
	.btn.sm {
		width: auto;
		flex: none;
		padding: 0.45rem 0.9rem;
		font-size: 0.85rem;
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
	}
	.small {
		font-size: 0.8rem;
		margin: 0;
	}
</style>
