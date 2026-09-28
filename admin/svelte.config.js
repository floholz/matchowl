import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		// SPA mode like the player app: no SSR, index.html fallback. The Go
		// binary embeds this directory and serves it on the admin host.
		adapter: adapter({
			pages: '../internal/web/admin',
			assets: '../internal/web/admin',
			fallback: 'index.html',
			precompress: false,
			strict: true
		})
	}
};

export default config;
