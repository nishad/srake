import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		proxy: {
			'/api/v1': {
				target: 'http://localhost:8080',
				changeOrigin: true
			},
			'/mcp': {
				target: 'http://localhost:8080',
				changeOrigin: true
			}
		}
	}
});
