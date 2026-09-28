<!-- The mail as a recipient will see it: rendered by the server with the
     real template and layout, shown in a sandboxed iframe. Debounced while
     the admin types; the recipient's first name is the admin's own. -->
<script lang="ts">
	import { api } from '$lib/api';
	import { Eye } from '@lucide/svelte';

	let {
		subject,
		body,
		ctaText = '',
		ctaUrl = '',
		event = 'mailing'
	}: { subject: string; body: string; ctaText?: string; ctaUrl?: string; event?: 'mailing' | 'announcement' } = $props();

	let html = $state('');
	let renderedSubject = $state('');
	let error = $state('');
	let open = $state(true);
	let timer: ReturnType<typeof setTimeout> | undefined;

	$effect(() => {
		const p = { subject, body, ctaText, ctaUrl, event };
		clearTimeout(timer);
		if (!p.subject.trim() && !p.body.trim()) {
			html = '';
			return;
		}
		timer = setTimeout(async () => {
			try {
				const r = await api.mailRender(p);
				html = r.html;
				renderedSubject = r.subject;
				error = '';
			} catch (e) {
				error = e instanceof Error ? e.message : 'Preview failed.';
			}
		}, 350);
		return () => clearTimeout(timer);
	});
</script>

<div class="pv">
	<button class="pvhead" onclick={() => (open = !open)} aria-expanded={open}>
		<Eye size={14} /> Mail preview{#if renderedSubject} · <span class="subj">{renderedSubject}</span>{/if}
	</button>
	{#if open}
		{#if error}
			<p class="error">{error}</p>
		{:else if html}
			<iframe title="Mail preview" sandbox="" srcdoc={html}></iframe>
		{:else}
			<p class="muted small">Type a subject and a message to see the mail.</p>
		{/if}
	{/if}
</div>

<style>
	.pv {
		margin: 0.8rem 0;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		overflow: hidden;
	}
	.pvhead {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		width: 100%;
		padding: 0.5rem 0.7rem;
		border: none;
		background: var(--surface-2);
		color: var(--muted);
		font: inherit;
		font-size: 0.8rem;
		font-weight: 700;
		text-align: left;
		cursor: pointer;
	}
	.subj {
		color: var(--text);
		font-weight: 600;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	iframe {
		display: block;
		width: 100%;
		height: 560px;
		border: none;
		background: #0b0f1a;
	}
	.small {
		font-size: 0.8rem;
		padding: 0.6rem 0.7rem;
		margin: 0;
	}
	.error {
		padding: 0.6rem 0.7rem;
		margin: 0;
	}
</style>
