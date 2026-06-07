import { env } from "$env/dynamic/public";

export const MAP_MAX_POLYLINES = Number(env.PUBLIC_MAP_MAX_POLYLINES || 100);
