<script lang="ts">
	// "Continue with Google", light theme, per Google's Identity branding
	// guidelines: colours, logo, Roboto Medium and capitalisation must not be
	// altered. https://developers.google.com/identity/branding-guidelines
	// Roboto is self-hosted (Fontsource) so an installed PWA never fetches a
	// font from Google on the sign-in page.
	import '@fontsource/roboto/500.css';

	let {
		disabled = false,
		onclick
	}: { disabled?: boolean; onclick: () => void } = $props();
</script>

<button
	type="button"
	class="gsi"
	{disabled}
	{onclick}
	aria-label="Continue with Google"
>
	<svg class="gsi-logo" viewBox="0 0 48 48" aria-hidden="true">
		<path
			fill="#EA4335"
			d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z"
		/>
		<path
			fill="#4285F4"
			d="M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 7.18l7.73 6c4.51-4.18 7.09-10.36 7.09-17.65z"
		/>
		<path
			fill="#FBBC05"
			d="M10.53 28.59c-.48-1.45-.76-2.99-.76-4.59s.27-3.14.76-4.59l-7.98-6.19C.92 16.46 0 20.12 0 24c0 3.88.92 7.54 2.56 10.78l7.97-6.19z"
		/>
		<path
			fill="#34A853"
			d="M24 48c6.48 0 11.93-2.13 15.89-5.81l-7.73-6c-2.15 1.45-4.92 2.3-8.16 2.3-6.26 0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z"
		/>
	</svg>
	<span class="gsi-text">Continue with Google</span>
</button>

<style>
	.gsi {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 12px;
		width: 100%;
		height: 40px;
		padding: 0 12px;
		background: #ffffff;
		border: 1px solid #747775;
		border-radius: 4px;
		color: #1f1f1f;
		font-family: 'Roboto', arial, sans-serif;
		font-size: 14px;
		font-weight: 500;
		letter-spacing: 0.25px;
		text-transform: none;
		cursor: pointer;
		transition:
			background-color 0.15s ease,
			border-color 0.15s ease;
	}
	.gsi:hover:not(:disabled) {
		/* Google light-theme hover state layer: #303030 @ ~8% over white */
		background: #f7f8f8;
		border-color: #747775;
		box-shadow: 0 1px 2px rgba(60, 64, 67, 0.3);
	}
	.gsi:focus-visible {
		outline: 2px solid #8ab4f8;
		outline-offset: 2px;
	}
	.gsi:disabled {
		opacity: 0.38;
		cursor: default;
	}
	.gsi-logo {
		width: 20px;
		height: 20px;
		flex: none;
	}
</style>
