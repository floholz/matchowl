<!-- The challenge icon: a boxing glove, tilted as a punch. Lucide has no
     glove, so it is drawn on Lucide's 24px grid with its 2px round strokes
     and takes the same props as a Lucide icon (size, class, colour from
     currentColor). Picked from the FigCraft options (design/, Proposals,
     "Challenge icon — options", H) over the crossed swords, which read as
     an ✕ at row size. -->
<script lang="ts">
	let { size = 24, class: cls = '', strokeWidth = 2 }: { size?: number; class?: string; strokeWidth?: number } = $props();
</script>

<svg
	xmlns="http://www.w3.org/2000/svg"
	width={size}
	height={size}
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width={strokeWidth}
	stroke-linecap="round"
	stroke-linejoin="round"
	class="lucide-icon lucide boxing-glove {cls}"
	aria-hidden="true"
>
	<g transform="rotate(30 12 12)">
		<path d="M8 13.5V8a5 5 0 0 1 5-5h1.5A4.5 4.5 0 0 1 19 7.5v5a3 3 0 0 1-2 2.83V16" />
		<path d="M8 9.5H7a2.5 2.5 0 0 0 0 5h4.5" />
		<path d="M7 16h10v4a1 1 0 0 1-1 1H8a1 1 0 0 1-1-1z" />
	</g>
</svg>
