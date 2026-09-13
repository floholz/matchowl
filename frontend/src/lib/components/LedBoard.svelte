<!-- The stadium board: a match's FINAL score as seven-segment LEDs on a
     black inset tile — amber once played, red while live, unlit before
     kick-off. A small LED dot marks the advancer of a knockout. Two-legged
     ties dock a "1st / leg" lug to the front of the tile. -->
<script lang="ts">
	let {
		home = null,
		away = null,
		live = false,
		advancer = '',
		leg = '',
		size = 'row'
	}: {
		/** Goals; null = not lit (nothing to show yet). */
		home?: number | null;
		away?: number | null;
		live?: boolean;
		/** Side that goes through (knockouts) — an LED dot next to its digits. */
		advancer?: '' | 'home' | 'away';
		/** Leg ordinal of a two-legged tie, shown on the lug. */
		leg?: '' | '1st' | '2nd';
		/** `hero` = horizontal, big (the match page). */
		size?: 'row' | 'hero';
	} = $props();
	let lit = $derived(home !== null && away !== null);
	const ghost = (n: number | null) => '8'.repeat(Math.max(1, String(n ?? 0).length));
	let label = $derived(lit ? `Score ${home}–${away}${live ? ', live' : ''}` : 'Not started');
</script>

<span class="board {size}" class:live class:off={!lit} role="img" aria-label={label}>
	{#if leg}
		<span class="lug"><b>{leg}</b><span>leg</span></span>
	{/if}
	<span class="tile">
		<span class="d"
			><i>{ghost(home)}</i><em>{lit ? home : ''}</em>{#if advancer === 'home'}<span class="mk"></span>{/if}</span
		>
		{#if size === 'hero'}<span class="colon">:</span>{/if}
		<span class="d"
			><i>{ghost(away)}</i><em>{lit ? away : ''}</em>{#if advancer === 'away'}<span class="mk"></span>{/if}</span
		>
	</span>
</span>

<style>
	.board {
		--led: var(--board-led, var(--warning));
		--led-glow: var(--board-glow, rgba(255, 180, 61, 0.55));
		display: inline-flex;
		align-items: center;
		flex: none;
		font-family: var(--font-led);
		font-weight: 700;
		color: var(--led);
	}
	.board.live {
		--led: var(--live);
		--led-glow: var(--board-live-glow, rgba(255, 61, 46, 0.6));
	}
	.tile {
		display: flex;
		flex-direction: column;
		align-items: stretch;
		gap: 6px;
		min-width: 44px;
		padding: 5px 4px;
		background: var(--board-bg);
		border-radius: 9px;
		box-shadow:
			inset 0 0 0 1px var(--board-line),
			inset 0 2px 6px var(--board-shade, rgba(0, 0, 0, 0.8));
		position: relative;
		z-index: 1;
	}
	.d {
		position: relative;
		height: 17px;
		line-height: 17px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 15px;
	}
	/* Unlit segments behind every digit: the board is always "on". */
	.d i {
		position: absolute;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		font-style: normal;
		opacity: 0.2;
	}
	.d em {
		position: relative;
		font-style: normal;
		text-shadow: 0 0 7px var(--led-glow);
	}
	.off .d em {
		color: transparent;
		text-shadow: none;
	}
	.mk {
		position: absolute;
		left: 4px;
		top: 50%;
		width: 5px;
		height: 5px;
		margin-top: -2.5px;
		border-radius: 50%;
		background: currentColor;
		box-shadow: 0 0 6px currentColor;
	}
	/* Leg lug: one black block with the tile, rounded on the outside only. */
	.lug {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 1px;
		height: 28px;
		width: 24px;
		margin-right: -8px;
		padding-right: 2px;
		background: var(--board-bg);
		border-radius: 7px 0 0 7px;
		box-shadow:
			inset 1px 0 0 var(--board-line),
			inset 0 1px 0 var(--board-line),
			inset 0 -1px 0 var(--board-line);
		line-height: 1;
		font-family: var(--font);
		font-size: 6.5px;
		font-weight: 800;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.lug b {
		color: var(--led);
		font-size: 9px;
		font-family: var(--font-mono);
	}
	/* Hero: the same tile, horizontal and big. */
	.hero .tile {
		flex-direction: row;
		align-items: center;
		gap: 10px;
		padding: 8px 12px;
		border-radius: 12px;
	}
	.hero .d {
		height: 34px;
		line-height: 34px;
		font-size: 30px;
		min-width: 26px;
	}
	.hero .colon {
		font-size: 22px;
		opacity: 0.55;
		line-height: 34px;
	}
	.hero .mk {
		left: -8px;
		width: 6px;
		height: 6px;
		margin-top: -3px;
	}
	.hero .lug {
		height: 40px;
		width: 30px;
		font-size: 8px;
	}
	.hero .lug b {
		font-size: 12px;
	}
</style>
