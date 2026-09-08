import { afterAll, afterEach, describe, expect, it, vi } from 'vitest';
import { Meilisearch, type MultiSearchParams, type SearchParams, type SearchResponse } from 'meilisearch';
import { cases, dataset, label, readJSON } from '../../../tests/srch0/fixtures';
import { POST as proxy } from '../../routes/api/v1/search/[index]/+server';
import { POST as multi } from '../../routes/api/v1/search/multi/+server';
import { assertSearchPlausibility, normalizeResult, eligibleDocuments } from '../../../../scripts/srch0/engine-results.mjs';
import { API_SEARCH_CASES, API_STARTUP_CASES, API_COMPLETENESS_CASES } from '../../../../scripts/srch0/selection.mjs';
import { generateScaleDataset } from '../../../../scripts/srch0/dataset-generator.mjs';
import { observeAPI } from '../../../tests/srch0/api-adapter';
import { apiEvent, routeError } from '../../../tests/srch0/api-context';
import type { Dataset, JsonRecord } from '../../../tests/srch0/types';

type Order = Record<string, unknown>;
type EngineInput =
    | { adapter: 'engine-search'; index: string; request: SearchParams & { q?: string }; proxy_request?: { q?: string; options: SearchParams }; order?: Order }
    | { adapter: 'engine-multi'; request: MultiSearchParams; proxy_request?: MultiSearchParams; orders?: Order[] };
type EngineResponse = SearchResponse<JsonRecord> & { results: SearchResponse<JsonRecord>[] };
interface StartupInput { adapter: 'go-startup'; indexes: string; empty_source: boolean }
interface StartupObservation { mutation_state: { document_counts: Record<string, number> }; startup_api: unknown }

const profile = process.env.SRCH0_MEILI_PROFILE;

describe.skipIf(!profile)('SRCH0 real engine through SvelteKit API and tenant principal', () => {
    afterEach(() => vi.unstubAllEnvs());
    for (const fixture of cases<EngineInput>(['search']).filter(fixture => fixture.case_id in API_SEARCH_CASES)) {
        it(`${label(fixture)} active_profile=${profile}`, async () => {
            vi.stubEnv('TZ', fixture.context.timezone);
            const token = process.env[`SRCH0_MEILI_${fixture.context.principal.toUpperCase()}_TOKEN`];
            expect(token, 'Disposable engine tenant credential is required').toBeTruthy();
            expect(process.env.SRCH0_MEILI_URL, 'Disposable engine URL is required').toBeTruthy();
            const ms = new Meilisearch({ host: process.env.SRCH0_MEILI_URL!, apiKey: token });
            const input = fixture.input;
            // Bereits kompilierte Filter enthalten ihre Präferenzen. Nur rohe
            // Proxyfälle lassen die produktive Route diese Klauseln einsetzen.
            const preferences = input.proxy_request ? fixture.context.preferences : {};
            const { q, ...options } = input.request as SearchParams & { q?: string };
            const body = input.proxy_request ?? (input.adapter === 'engine-multi' ? input.request : { q, options });
            const event = apiEvent({ principal: fixture.context.principal, preferences, ms,
                params: input.adapter === 'engine-search' ? { index: input.index } : {}, body });
            let actual;
            // Negative engine cases are intentional; avoid printing request or
            // SDK errors, which can include the ephemeral credential context.
            const stderr = vi.spyOn(console, 'error').mockImplementation(() => {});
            try {
                const response = await (input.adapter === 'engine-multi' ? multi : proxy)(event);
                const body = await response.json() as EngineResponse;
                if (input.adapter === 'engine-multi') body.results.forEach((result, i) =>
                    assertSearchPlausibility(result, input.request.queries[i], dataset(), fixture.context.principal, input.request.queries[i].indexUid));
                else assertSearchPlausibility(body, input.request, dataset(), fixture.context.principal, input.index);
                actual = input.adapter === 'engine-multi'
                    ? { results: body.results.map((result, i) => normalizeResult(result, input.orders?.[i] ?? {})) }
                    : { result: normalizeResult(body, input.order ?? {}) };
            } catch (error) {
                if (!error || typeof error !== 'object' || !('status' in error)) throw error;
                actual = { diagnostics: routeError(error) };
            } finally { stderr.mockRestore(); }
            const observed = fixture.observed;
            expect(actual).toEqual(observed.api_diagnostics ? { diagnostics: observed.api_diagnostics } : observed);
        });
    }
});

describe.skipIf(!profile)('SRCH0 vollständige Produktabfragen gegen die echte Engine', () => {
    let replacedTrails = false;
    const admin = () => new Meilisearch({ host: process.env.SRCH0_MEILI_URL!, apiKey: process.env.SRCH0_MEILI_KEY });
    async function replaceTrails(trails: object[]) {
        const target = admin().index('trails');
        expect((await target.deleteAllDocuments().waitTask({ timeout: 30000, interval: 10 })).status).toBe('succeeded');
        for (let offset = 0; offset < trails.length; offset += 1000) {
            expect((await target.addDocuments(trails.slice(offset, offset + 1000)).waitTask({ timeout: 30000, interval: 10 })).status).toBe('succeeded');
        }
    }
    afterEach(async () => {
        if (replacedTrails) await replaceTrails(dataset().trails);
        replacedTrails = false;
        vi.restoreAllMocks();
    });
    for (const fixture of cases<EngineInput>(['search']).filter(fixture => fixture.case_id in API_COMPLETENESS_CASES)) {
        it(`${label(fixture)} active_profile=${profile} complete_consumer`, async () => {
            expect(process.env.SRCH0_MEILI_DISPOSABLE).toBe('true');
            const token = process.env[`SRCH0_MEILI_${fixture.context.principal.toUpperCase()}_TOKEN`];
            expect(token).toBeTruthy();
            const ms = new Meilisearch({ host: process.env.SRCH0_MEILI_URL!, apiKey: token });
            let actual;
            if (fixture.case_id === 'SRCH0-SEARCH-117') {
                const source = dataset();
                const match = source.trails.find(trail => trail.id === 'duplicate-21')!;
                const firstPage = await ms.index('trails').search('', { attributesToRetrieve: ['id'] });
                expect(firstPage.hits.length).toBeLessThan(eligibleDocuments(source, 'trails', fixture.context.principal).length);
                const response = await observeAPI(fixture.context, {
                    adapter: 'api-upload', trail: { name: 'Synthetic upload', distance: match.distance, elevation_gain: match.elevation_gain,
                        elevation_loss: match.elevation_loss, lat: match._geo.lat, lon: match._geo.lng, tags: [], photos: [], expand: {} },
                }, source, ms);
                expect(response).toMatchObject({ status: 400, result: { duplicate_id: match.id } });
                actual = { status: response.status, result: response.result };

                // Welcher Treffer ausserhalb der ersten Seite liegt, bestimmt
                // die Engine. Weit getrennte Distanzen verhindern, dass bereits
                // ein anderer Kandidat als fachlich gleiches Duplikat gilt.
                const candidates = source.trails.filter(trail => trail.id.startsWith('duplicate-'))
                    .map((trail, i) => ({ ...trail, distance: 1000000 + i * 1000 }));
                replacedTrails = true;
                await replaceTrails(candidates);
                const page = await ms.index('trails').search('', { attributesToRetrieve: ['id'] });
                const ids = new Set(page.hits.map(hit => hit.id));
                const omitted = candidates.find(trail => !ids.has(trail.id));
                expect(omitted, 'Ein eindeutiger Duplikatkandidat muss ausserhalb der ersten Seite liegen').toBeDefined();
                const later = await observeAPI(fixture.context, {
                    adapter: 'api-upload', trail: { name: 'Synthetic later duplicate', distance: omitted!.distance,
                        elevation_gain: omitted!.elevation_gain, elevation_loss: omitted!.elevation_loss,
                        lat: omitted!._geo.lat, lon: omitted!._geo.lng, tags: [], photos: [], expand: {} },
                }, { ...source, trails: candidates }, ms);
                expect(later).toMatchObject({ status: 400, result: { duplicate_id: omitted!.id } });
            } else {
                const source = generateScaleDataset(dataset(), readJSON(fixture.dataset_ref)) as Dataset;
                replacedTrails = true;
                await replaceTrails(source.trails);
                const cap = (await admin().index('trails').getSettings()).pagination!.maxTotalHits!;
                expect(source.trails.length).toBeGreaterThan(cap);
                const response = await observeAPI(fixture.context, {
                    adapter: 'api-cluster', body: { southWest: { lat: -85, lng: -180 }, northEast: { lat: 85, lng: 180 }, zoom: 0 },
                }, source, ms);
                expect(response.status).toBe(200);
                const body = response.response as { totalHits: number; features: { properties: { point_count: number } }[] };
                const represented = body.features.reduce((sum, feature) => sum + feature.properties.point_count, 0);
                const visible = eligibleDocuments(source, 'trails', fixture.context.principal).length;
                expect(body.totalHits).toBe(visible);
                expect(represented).toBe(visible);
                actual = { status: response.status, total: body.totalHits, represented };
            }
            expect(actual).toEqual(fixture.observed.api_result);
        }, 120000);
    }
});

describe.skipIf(!profile)('SRCH0 API nach vollständig abgeschlossenem Startup', () => {
    const indexes = ['trails', 'lists', 'actors'] as const;
    let didMutate = false;
    const admin = () => new Meilisearch({ host: process.env.SRCH0_MEILI_URL!, apiKey: process.env.SRCH0_MEILI_KEY });

    async function materialize(index: string, documents: object[]) {
        const target = admin().index(index);
        const deleted = await target.deleteAllDocuments().waitTask({ timeout: 10000, interval: 10 });
        expect(deleted.status, `${index}: prepare recorded startup state`).toBe('succeeded');
        if (documents.length) {
            const added = await target.addDocuments(documents).waitTask({ timeout: 10000, interval: 10 });
            expect(added.status, `${index}: materialize recorded startup state`).toBe('succeeded');
        }
    }

    afterEach(() => vi.unstubAllEnvs());
    afterAll(async () => {
        if (!didMutate) return;
        const source = dataset();
        for (const index of indexes) await materialize(index, source[index]);
    });

    for (const fixture of cases<StartupInput, StartupObservation>(['mutation']).filter(fixture => fixture.case_id in API_STARTUP_CASES)) {
        it(`${label(fixture)} active_profile=${profile} startup_api`, async () => {
            vi.stubEnv('TZ', fixture.context.timezone);
            // The engine runner owns this disposable service. This suite changes
            // its contents; ambient developer service URLs must not enable it.
            expect(process.env.SRCH0_MEILI_DISPOSABLE).toBe('true');
            expect(process.env.SRCH0_MEILI_KEY).toBeTruthy();
            expect(process.env.SRCH0_MEILI_ANONYMOUS_TOKEN).toBeTruthy();
            const source = dataset();
            const rebuild: Record<string, object[]> = {
                trails: source.trails.filter(trail => ['public-alpine', 'public-lake'].includes(trail.id))
                    .map(trail => ({ ...trail, shares: [], likes: [], like_count: 0 })),
                lists: source.lists.filter(list => list.id === 'list-local'),
                actors: source.actors,
            };
            if (fixture.input.empty_source) for (const index of indexes) rebuild[index] = [];
            const staged: Record<string, object[]> = Object.fromEntries(indexes.map(index => [index, fixture.input.indexes === 'existing' ? [{ id: 'prior' }] : []]));
            const ms = new Meilisearch({ host: process.env.SRCH0_MEILI_URL!, apiKey: process.env.SRCH0_MEILI_ANONYMOUS_TOKEN });
            didMutate = true;

            // Go verifies that the real serve hook cannot open search before
            // completion. This API check then qualifies only its ready state.
            const counts: Record<string, number> = {};
            for (const index of indexes) {
                await materialize(index, fixture.input.indexes === 'existing' ? staged[index] : rebuild[index]);
                counts[index] = (await admin().index(index).getStats()).numberOfDocuments;
            }
            expect(counts).toEqual(fixture.observed.mutation_state.document_counts);
            const event = apiEvent({ principal: 'anonymous', ms,
                body: { q: '', options: { attributesToRetrieve: ['id'], sort: ['created:asc'], hitsPerPage: 25, page: 1 } } });
            const response = await proxy(event);
            const result = await response.json() as SearchResponse<{ id: string }>;
            const actual = [{ phase: 'ready', status: response.status, hit_ids: result.hits.map(hit => hit.id).sort(), total: result.totalHits }];
            expect(actual).toEqual(fixture.observed.startup_api);
        }, 30000);
    }
});
