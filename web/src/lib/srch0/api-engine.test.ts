import { afterAll, afterEach, describe, expect, it, vi } from 'vitest';
import { Meilisearch, type MultiSearchParams, type SearchParams, type SearchResponse } from 'meilisearch';
import { cases, dataset, label } from '../../../tests/srch0/fixtures';
import { POST as proxy } from '../../routes/api/v1/search/[index]/+server';
import { POST as multi } from '../../routes/api/v1/search/multi/+server';
import { assertSearchPlausibility, normalizeResult } from '../../../../scripts/srch0/engine-results.mjs';
import { isAPIIntegrationCase } from '../../../../scripts/srch0/selection.mjs';
import { apiEvent, routeError } from '../../../tests/srch0/api-context';
import type { JsonRecord } from '../../../tests/srch0/types';

type Order = Record<string, unknown>;
type EngineInput =
    | { adapter: 'engine-search'; index: string; request: SearchParams & { q?: string }; proxy_request?: { q?: string; options: SearchParams }; order?: Order }
    | { adapter: 'engine-multi'; request: MultiSearchParams; proxy_request?: MultiSearchParams; orders?: Order[] };
type EngineResponse = SearchResponse<JsonRecord> & { results: SearchResponse<JsonRecord>[] };
interface StartupInput { adapter: 'go-startup'; indexes: string; empty_source: boolean }
interface StartupStage { phase: string; document_counts: Record<string, number> }
interface StartupObservation { mutation_state: { availability: StartupStage[] }; startup_api: unknown }

const profile = process.env.SRCH0_MEILI_PROFILE;

describe.skipIf(!profile)('SRCH0 real engine through SvelteKit API and tenant principal', () => {
    afterEach(() => vi.unstubAllEnvs());
    for (const fixture of cases<EngineInput>(['search']).filter(isAPIIntegrationCase)) {
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

describe.skipIf(!profile)('SRCH0 API reachability in recorded startup index states', () => {
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

    for (const fixture of cases<StartupInput, StartupObservation>(['mutation']).filter(isAPIIntegrationCase)) {
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
            const actual = [];
            didMutate = true;

            // Go separately runs and pauses the real serve hook to observe the
            // phases. Here each recorded state is materialized to qualify search
            // through the actual API and tenant-scoped engine. Test setup waits
            // are not runtime task barriers or a concurrent process-start claim.
            for (const stage of fixture.observed.mutation_state.availability) {
                const counts: Record<string, number> = {};
                for (const index of indexes) {
                    await materialize(index, staged[index]);
                    counts[index] = (await admin().index(index).getStats()).numberOfDocuments;
                }
                expect(counts, `${fixture.case_id}: ${stage.phase}`).toEqual(stage.document_counts);
                const event = apiEvent({ principal: 'anonymous', ms,
                    body: { q: '', options: { attributesToRetrieve: ['id'], sort: ['created:asc'], hitsPerPage: 25, page: 1 } } });
                const response = await proxy(event);
                const result = await response.json() as SearchResponse<{ id: string }>;
                actual.push({ phase: stage.phase, status: response.status, hit_ids: result.hits.map(hit => hit.id).sort(), total: result.totalHits });

                // Phase observations are made before submission, so only the
                // next phase incorporates the current operation's materialization.
                const [method, path] = stage.phase.split(' ');
                const index = path.split('/')[2];
                staged[index] = method === 'DELETE' ? [] : rebuild[index];
            }
            expect(actual).toEqual(fixture.observed.startup_api);
        }, 30000);
    }
});
