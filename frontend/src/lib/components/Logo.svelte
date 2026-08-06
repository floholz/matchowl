<script lang="ts">
	let { compact = false }: { compact?: boolean } = $props();
</script>

<a class="logo" href="/" aria-label="Matchowl home">
	<!-- The Matchowl mark: an owl face built from football pitch markings
	     (see docs/matchowl-icon.svg). Inlined so it ships with the bundle. -->
	<svg
		class="mark"
		viewBox="0 0 256 256"
		role="img"
		aria-hidden="true"
		focusable="false"
	>
		<rect width="256" height="256" rx="56" fill="#FFD7B4" />
		<path d="M20 201L236 201" stroke="#FF7700" stroke-width="14" />
		<rect
			x="127.971"
			y="131"
			width="24"
			height="24"
			transform="rotate(45 127.971 131)"
			fill="#FF7700"
		/>
		<circle cx="94" cy="120" r="36" fill="#141414" />
		<circle cx="94" cy="120" r="23" fill="#fff" />
		<circle cx="94" cy="119.5" r="12" fill="#FF7700" />
		<circle cx="162" cy="120" r="36" fill="#141414" />
		<circle cx="162" cy="120" r="23" fill="#fff" />
		<circle cx="162" cy="119.5" r="12" fill="#FF7700" />
		<path d="M198 20V88H58V20" stroke="#141414" stroke-width="14" fill="none" />
		<path d="M161 20V56H95V20" stroke="#141414" stroke-width="14" fill="none" />
	</svg>
	{#if !compact}<span class="word">Matchowl</span>{/if}
</a>

<style>
	.logo {
		display: inline-flex;
		align-items: center;
		gap: 0.7rem;
		color: var(--text);
		font-weight: 800;
		letter-spacing: -0.02em;
	}
	.mark {
		height: 30px;
		width: 30px;
		display: block;
	}
	.word {
		font-family: var(--font-display);
		font-weight: 700;
		font-size: 1.35rem;
		letter-spacing: -0.015em;
	}
</style>
