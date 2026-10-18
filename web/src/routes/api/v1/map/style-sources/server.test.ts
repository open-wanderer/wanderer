import type { RequestEvent } from '@sveltejs/kit';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const env = vi.hoisted(() => ({} as Record<string, string | undefined>));
vi.mock('$env/dynamic/private', () => ({ env }));

import { GET } from './+server';

function event(): RequestEvent {
  return { locals: { pb: { authStore: { record: { id: 'user' } } } } } as unknown as RequestEvent;
}

describe('GET /api/v1/map/style-sources', () => {
  beforeEach(() => {
    for (const key of Object.keys(env)) delete env[key];
  });

  it('treats empty values as unset', async () => {
    Object.assign(env, { TILE_SERVER_URL: '', MAP_ASSETS_URL: ' ', PROTOMAPS_API_KEY: 'key' });

    const body = await (await GET(event())).json();

    expect(body.tileUrl).toBe('https://api.protomaps.com/tiles/v4/{z}/{x}/{y}.mvt?key=key');
    expect(body.glyphUrl).toBe('https://protomaps.github.io/basemaps-assets/fonts/{fontstack}/{range}.pbf');
  });

  it('uses configured values', async () => {
    Object.assign(env, { TILE_SERVER_URL: 'https://tiles.example/{z}/{x}/{y}.mvt', MAP_ASSETS_URL: 'https://assets.example/' });

    const body = await (await GET(event())).json();

    expect(body.tileUrl).toBe('https://tiles.example/{z}/{x}/{y}.mvt');
    expect(body.spriteUrl).toBe('https://assets.example/sprites/v4');
  });
});
