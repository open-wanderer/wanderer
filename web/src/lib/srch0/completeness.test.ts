import { afterEach, describe, expect, it } from 'vitest';
import { observeAPI } from '../../../tests/srch0/api-adapter';
import { resetAdapters } from '../../../tests/srch0/adapters';
import { dataset } from '../../../tests/srch0/fixtures';
import type { Context } from '../../../tests/srch0/types';

const context: Context = { principal: 'alice', timezone: 'Europe/Zurich', locale: 'de', surface: 'list', consumer: 'upload-duplicate', preferences: {} };

// Die externe Engine liefert begrenzte Antworten. Die fachliche Erwartung
// stammt aus dem vollständigen Bestand und nicht aus dem bisherigen Resultat.
function page(hits: unknown[]) {
    const result = { hits, estimatedTotalHits: hits.length, offset: 0, limit: 1000 };
    return { ...result, results: [result] };
}

describe('SRCH0: vollständige Suchgrundlage für Duplikate und Cluster', () => {
    afterEach(resetAdapters);

    it('findet ein Duplikat hinter der ersten Engineantwort', async () => {
        const source = dataset();
        const preceding = source.trails.filter(trail => /^duplicate-(0[1-9]|1[0-9]|20)$/.test(trail.id));
        const match = source.trails.find(trail => trail.id === 'duplicate-21')!;
        expect(preceding).toHaveLength(20);
        const actual = await observeAPI(context, {
            adapter: 'api-upload',
            trail: { name: 'Synthetic upload', distance: 4242, elevation_gain: 30, elevation_loss: 30, lat: 47, lon: 8, tags: [], photos: [], expand: {} },
            responses: [page(preceding), page([match]), page([])],
        }, source);
        expect(actual).toMatchObject({ status: 400, result: { duplicate_id: match.id } });
    });

    it('stellt auch Trails jenseits des Engine-Caps vollständig auf der Karte dar', async () => {
        const hits = Array.from({ length: 1001 }, (_, i) => ({ id: `complete-${i}`, _geo: { lat: 47, lng: 8 }, bounding_box_diagonal: 0 }));
        const actual = await observeAPI({ ...context, consumer: 'map-cluster' }, {
            adapter: 'api-cluster',
            body: { southWest: { lat: 46, lng: 7 }, northEast: { lat: 48, lng: 9 }, zoom: 0 },
            responses: [page(hits.slice(0, 1000)), page(hits.slice(1000)), page([])],
        }, dataset());
        expect(actual.status).toBe(200);
        const response = actual.response as { totalHits: number; features: { properties: { point_count: number } }[] };
        expect(response.totalHits).toBe(hits.length);
        expect(response.features.reduce((sum, feature) => sum + feature.properties.point_count, 0)).toBe(hits.length);
    });
});
