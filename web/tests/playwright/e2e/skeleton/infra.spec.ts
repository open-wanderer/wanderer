import { test, expect } from '@playwright/test';
import { deleteTrail } from '../../helpers/api';
import * as fs from 'fs';
import * as path from 'path';

test('infrastructure: upload and delete a trail end-to-end', async ({ request }) => {
    // Read the GPX fixture from disk — path relative to the web/ directory (Playwright cwd)
    const gpxPath = path.join('tests', 'playwright', 'fixtures', 'trail.gpx');
    const gpxBuffer = fs.readFileSync(gpxPath);

    // Upload the GPX file via the SvelteKit proxy endpoint.
    // ignoreDuplicates='true' bypasses duplicate detection (required to re-run the spec cleanly).
    // The request fixture inherits storageState from the chromium project config,
    // so the pb_auth session cookie is forwarded automatically.
    const uploadResponse = await request.put('/api/v1/trail/upload', {
        multipart: {
            file: {
                name: 'trail.gpx',
                mimeType: 'application/gpx+xml',
                buffer: gpxBuffer,
            },
            ignoreDuplicates: 'true',
        },
    });

    expect(uploadResponse.status()).toBe(200);

    const trail = await uploadResponse.json();

    // Assert the trail was created with a valid id
    expect(trail.id).toBeTruthy();

    // Assert the name was auto-populated from <trk><name>Test Trail</name> in the GPX
    expect(trail.name).toBe('Test Trail');

    // Assert geometry survived the smoothing thresholds (XY > 5m, Z > 5m)
    expect(trail.distance).toBeGreaterThan(0);
    expect(trail.elevation_gain).toBeGreaterThan(0);

    // Teardown: delete the trail via the typed helper (throws on non-ok response)
    await deleteTrail(request, trail.id);

    // Assert the trail is gone — a follow-up GET should return 404
    const getResponse = await request.get(`/api/v1/trail/${trail.id}`);
    expect(getResponse.status()).toBe(404);
});
