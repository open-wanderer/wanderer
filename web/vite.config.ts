import { enhancedImages } from '@sveltejs/enhanced-img';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import fs from 'fs';
import openapiPlugin from 'sveltekit-openapi-generator';
import { defineConfig } from 'vitest/config';
import { openapiOptions } from './openapi.config.js';

const packageJson = JSON.parse(fs.readFileSync('./package.json', 'utf-8'));

export default defineConfig({
	plugins: [enhancedImages(), openapiPlugin(openapiOptions(packageJson.version)), tailwindcss(), sveltekit()],
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
