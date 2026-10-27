import { defineEnvVars } from '@sveltejs/kit/env';

// Variables without a schema are required, even at build time. Keep them
// optional so the image builds without the runtime configuration.
const optional = (value: string | undefined) => value;

export const variables = defineEnvVars({
	ORIGIN: { schema: optional },
	MEILI_URL: { schema: optional },
	BODY_SIZE_LIMIT: { schema: optional },
	POCKETBASE_PROXY_SECRET: { schema: optional },
	VALHALLA_URL: { schema: optional },
	NOMINATIM_URL: { schema: optional },
	OVERPASS_API_URL: { schema: optional },
	PUBLIC_VALHALLA_URL: { public: true, schema: optional },
	PUBLIC_NOMINATIM_URL: { public: true, schema: optional },
	PUBLIC_OVERPASS_API_URL: { public: true, schema: optional },
	PUBLIC_POCKETBASE_URL: { public: true, schema: optional },
	PUBLIC_DISABLE_SIGNUP: { public: true, schema: optional },
	PUBLIC_IS_DEMO: { public: true, schema: optional },
	PUBLIC_PRIVATE_INSTANCE: { public: true, schema: optional },
	PUBLIC_MAP_MAX_POLYLINES: { public: true, schema: optional }
});
