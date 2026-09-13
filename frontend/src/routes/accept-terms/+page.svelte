<script lang="ts">
	// Terms interstitial. The layout sends every signed-in account whose
	// stored termsVersion is not the current one here: Google sign-ups (no
	// checkbox before the popup), imported accounts, and everyone after a
	// terms bump. Nothing else in the app is reachable until accepted.
	import { auth } from '$lib/auth.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import AuthShell from '$lib/components/AuthShell.svelte';

	let agreed = $state(false);
	let busy = $state(false);
	let error = $state('');

	// Where to go afterwards: an invite carried through, else Home.
	let invite = $derived($page.url.searchParams.get('invite'));

	async function accept() {
		error = '';
		busy = true;
		try {
			await auth.acceptTerms();
			goto(invite ? `/join/${invite}` : '/', { replaceState: true });
		} catch (err: unknown) {
			error = (err as { message?: string })?.message ?? 'Could not save your acceptance.';
		} finally {
			busy = false;
		}
	}
	function decline() {
		auth.logout();
		goto('/login', { replaceState: true });
	}
</script>

<AuthShell title="One thing before you play" lead="Please accept the terms to use Matchowl.">
	<div class="card">
		<p>
			Matchowl is a free prediction game among friends. To run it we store your
			account, your tips and forecasts, and the pools and friendships you set
			up. We send email only to verified addresses, and only what you switch on
			in Settings — plus the account mails you request.
		</p>
		<p class="muted small">
			The <a href="/legal/terms" target="_blank" rel="noopener">terms of use</a>
			and the <a href="/legal/privacy" target="_blank" rel="noopener">privacy notice</a>
			have the details.
		</p>
		<label class="terms">
			<input type="checkbox" bind:checked={agreed} />
			<span>I accept the terms of use and have read the privacy notice.</span>
		</label>
		{#if error}<p class="error">{error}</p>{/if}
		<button class="btn" onclick={accept} disabled={!agreed || busy}>
			{busy ? 'Saving…' : 'Accept and continue'}
		</button>
		<button class="btn ghost" onclick={decline} disabled={busy}>Not now — sign out</button>
	</div>
</AuthShell>

<style>
	.card p {
		margin: 0 0 0.75rem;
		font-size: 0.95rem;
		line-height: 1.45;
	}
	.small {
		font-size: 0.85rem;
	}
	.terms {
		display: flex;
		align-items: flex-start;
		gap: 0.6rem;
		margin: 0.5rem 0 1rem;
		font-size: 0.9rem;
		line-height: 1.35;
		cursor: pointer;
	}
	.terms input {
		margin-top: 0.2rem;
		accent-color: var(--accent);
		flex: none;
	}
	.btn + .btn {
		margin-top: 0.6rem;
	}
</style>
