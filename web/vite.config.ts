import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';
import { enhancedImages } from '@sveltejs/enhanced-img';

export default defineConfig({
	plugins: [enhancedImages(), tailwindcss(), sveltekit()],
	test: { include: ['src/**/*.{test,spec}.{js,ts}'] },
	ssr: { noExternal: ['three'] },
	...(process.env.WANDERER_ENV == "dev" ? {
		server: {
			// https: {
			// 	key: fs.readFileSync('.svelte-kit/key.pem'),
			// 	cert: fs.readFileSync('.svelte-kit/cert.pem')
			// },
			// host: true, // true
			// port: 443 // 443
		}
	} : {})
});
