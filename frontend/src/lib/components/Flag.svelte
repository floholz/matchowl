<script lang="ts">
	// Renders the bundled /flags/<iso2>.svg, else the team's crest (clubs,
	// or nations without a bundled flag), else the code chip.
	let {
		iso2 = '',
		code = '',
		logo = '',
		size = 22
	}: { iso2?: string; code?: string; logo?: string; size?: number } = $props();
	let flagFailed = $state(false);
	let logoFailed = $state(false);
</script>

{#if iso2 && !flagFailed}
	<img
		class="flag"
		src={`/flags/${iso2}.svg`}
		alt={code}
		style="width:{size}px;height:{size * 0.72}px"
		onerror={() => (flagFailed = true)}
		loading="lazy"
	/>
{:else if logo && !logoFailed}
	<img
		class="crest"
		src={logo}
		alt={code}
		style="width:{size}px;height:{size}px"
		onerror={() => (logoFailed = true)}
		loading="lazy"
	/>
{:else}
	<span class="flag chip" style="font-size:{size * 0.42}px">{code || '—'}</span>
{/if}

<style>
	.flag {
		display: inline-block;
		object-fit: cover;
		border-radius: 3px;
		border: 1px solid var(--border);
		flex: none;
		vertical-align: middle;
	}
	.crest {
		display: inline-block;
		object-fit: contain;
		flex: none;
		vertical-align: middle;
	}
	.chip {
		display: inline-grid;
		place-items: center;
		min-width: 2.2em;
		padding: 0.15em 0.35em;
		background: var(--surface-2);
		color: var(--muted);
		font-weight: 700;
		letter-spacing: 0.03em;
	}
</style>
