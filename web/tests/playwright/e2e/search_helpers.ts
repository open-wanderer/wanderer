import type { BrowserContext } from '@playwright/test';

export async function mockEmptyGeocoding(context: BrowserContext) {
    await context.route('**/api/v1/geocoding/search**', route => route.fulfill({ json: {
        type: 'FeatureCollection', features: [],
    } }));
}
