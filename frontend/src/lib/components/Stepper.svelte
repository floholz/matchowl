<script lang="ts">
	import { Minus, Plus } from '@lucide/svelte';

	let {
		value = $bindable(0),
		min = 0,
		max = 99,
		disabled = false,
		onchange = undefined,
		size = 'md'
	}: {
		value: number;
		min?: number;
		max?: number;
		disabled?: boolean;
		onchange?: (value: number) => void;
		/** `lg` = the match page's big steppers. */
		size?: 'md' | 'lg';
	} = $props();

	function bump(d: number) {
		const n = value + d;
		if (n >= min && n <= max) {
			value = n;
			onchange?.(n);
		}
	}
</script>

<div class="stepper {size}" class:disabled>
	<button type="button" aria-label="decrease" onclick={() => bump(-1)} {disabled}>
		<Minus size={size === 'lg' ? 20 : 16} />
	</button>
	<span class="val digits">{value}</span>
	<button type="button" aria-label="increase" onclick={() => bump(1)} {disabled}>
		<Plus size={size === 'lg' ? 20 : 16} />
	</button>
</div>

<style>
	.stepper {
		display: inline-flex;
		align-items: center;
		background: var(--bg);
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
	}
	.stepper.disabled {
		opacity: 0.6;
	}
	.stepper button {
		display: grid;
		place-items: center;
		width: 34px;
		height: 34px;
		background: none;
		border: none;
		color: var(--accent);
	}
	.stepper button:disabled {
		color: var(--muted);
	}
	.val {
		min-width: 1.6rem;
		text-align: center;
		font-weight: 800;
		font-size: 1.05rem;
	}
	.lg button {
		width: 44px;
		height: 44px;
	}
	.lg .val {
		min-width: 2.4rem;
		font-size: 1.9rem;
	}
</style>
