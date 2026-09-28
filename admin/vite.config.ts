import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		// Own port (the player app's dev server is 5173); API and PocketBase
		// UI proxied to the local backend so everything is same-origin.
		port: 5174,
		strictPort: true,
		proxy: {
			'/api': 'http://127.0.0.1:8090',
			'/_': 'http://127.0.0.1:8090'
		}
	}
});
