export function openapiOptions(version) {
	return {
		info: {
			title: 'Wanderer API',
			version,
			description: 'API documentation for wanderer backend',
		},
		outputPath: 'static/docs/api/wanderer.openapi.json',
		include: ['src/routes/api/v1/**/*.{js,ts}'],
		baseSchemasPath: 'src/lib/models/api/openapi_schemas.ts',
	};
}
