import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';
import openapiPlugin from 'sveltekit-openapi-generator';
import fs from 'fs';

const packageJson = JSON.parse(fs.readFileSync('./package.json', 'utf-8'));

export default defineConfig({
	plugins: [openapiPlugin({
		info: {
			title: 'Wanderer API',
			version: `${packageJson.version}`,
			description: 'API documentation for wanderer backend',
		},
		outputPath: 'static/docs/api/wanderer.openapi.json',
		include: ['src/routes/api/v1/**/*.{js,ts}'],
		baseSchemasPath: 'src/lib/models/api/openapi_schemas.ts',
	}),tailwindcss(), sveltekit()],
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
