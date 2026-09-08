import { afterAll, afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { mkdirSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { MeilisearchApiError } from 'meilisearch';
import { POST as proxy } from '../../routes/api/v1/search/[index]/+server';
import { POST as multi } from '../../routes/api/v1/search/multi/+server';
import { handleError } from '$lib/util/api_util';
import { sanitizeTrailFilter } from '$lib/util/trail_filter_util';
import { defaultTrailSearchAttributes, Trail } from '$lib/models/trail';
import { apiEvent, routeError } from '../../../tests/srch0/api-context';
import { observeCompiler, observeDTO } from '../../../tests/srch0/compiler-adapter';
import { observeAPI } from '../../../tests/srch0/api-adapter';
import { loadFilter } from '../../../tests/srch0/state-adapter';
import { cases, dataset, digest } from '../../../tests/srch0/fixtures';
import type { CompilerInput, JsonRecord, WebInput } from '../../../tests/srch0/types';

const context = cases<WebInput>(['compiler'])[0].context;
const findings: JsonRecord[] = [];

function record(id: string, property: string, actual: unknown, satisfied: boolean) {
    findings.push({ id, eigenschaft: property, ergebnis: satisfied ? 'erfüllt' : 'verletzt', beobachtung: actual });
    expect(satisfied, `${id}: ${property}: ${JSON.stringify(actual)}`).toBe(true);
}

async function request(input: CompilerInput) {
    const result = await observeCompiler(context, input, dataset());
    return result.engine_requests[0].body as { attributesToRetrieve?: string[]; options: { filter: string; sort: string[]; attributesToRetrieve?: string[] } };
}
const radii = (filter: string) => filter.match(/_geoRadius\([^)]*\)/g) ?? [];

describe('SRCH0: fachliche Eigenschaften müssen erfüllt sein', () => {
    beforeEach(() => vi.stubEnv('TZ', context.timezone));
    afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); vi.restoreAllMocks(); });
    afterAll(() => {
        const directory = resolve(process.env.SRCH0_REPORT_DIR ?? 'test-results');
        mkdirSync(directory, { recursive: true });
        writeFileSync(resolve(directory, 'srch0-plausibility.json'), JSON.stringify({
            manifest_sha256: digest('manifest.json'), dataset_sha256: digest('datasets/reference.json'), changes_sha256: digest('changes.json'),
            hinweis: 'Jede verletzte Eigenschaft lässt die Prüfung scheitern; bekannte Fehler sind keine Ausnahme.',
            pruefungen: findings,
        }, null, 2) + '\n');
    });

    it('erhält einen vom SDK gemeldeten Clientfehler als 4xx', async () => {
        const sdkError = new MeilisearchApiError(new Response(null, { status: 400 }), {
            message: 'synthetic invalid filter', code: 'invalid_search_filter', type: 'invalid_request', link: 'https://example.invalid/error',
        });
        const event = apiEvent({ principal: 'anonymous', ms: { index: () => ({ search: async () => { throw sdkError; } }) }, body: { q: '', options: {} } });
        vi.spyOn(console, 'error').mockImplementation(() => {});
        let status = 0;
        try { status = (await proxy(event)).status; } catch (error) { status = routeError(error).status; }
        record('SRCH0-P-HTTP', 'Der SDK-Status wird unverändert weitergegeben.',
            { sdk_status: sdkError.response.status, api_status: status }, status === sdkError.response.status);
    });

    it('behandelt gültige Nullkoordinaten wie andere Startpunkte', async () => {
        const counts = [];
        for (const [lat, lon] of [[46, 7], [0, 7], [46, 0]]) {
            counts.push(radii((await request({ adapter: 'list-search', filter: { near: { lat, lon, radius: 2000 } } })).options.filter).length);
        }
        expect(counts[0], 'Kontrollfall mit gültigen Koordinaten').toBeGreaterThan(0);
        record('SRCH0-P-GEO-ZERO', 'Ein gültiger Startpunkt auf Äquator oder Nullmeridian aktiviert den Radius.',
            { normal: counts[0], latitude_null: counts[1], longitude_null: counts[2] }, counts.every(count => count > 0));
    });

    it('benötigt denselben Radiusfilter nur einmal', async () => {
        const clauses = radii((await request({ adapter: 'list-search', filter: { near: { lat: 46, lon: 7, radius: 2000 } } })).options.filter);
        record('SRCH0-P-GEO-DUPLICATE', 'Ein Radius wird genau einmal angewendet.', clauses, clauses.length === 1);
    });

    it.each(['2026-09-07', '2026-03-29', '2026-10-25'])('umfasst den lokalen Kalendertag %s einschliesslich Zeitumstellung', async day => {
        const filter = (await request({ adapter: 'list-search', filter: { startDate: day, endDate: day } })).options.filter;
        const start = new Date(`${day}T00:00:00`);
        const next = new Date(start);
        next.setDate(next.getDate() + 1);
        const lower = filter.match(/date\s*>=\s*(\d+)/);
        const upper = filter.match(/date\s*(<=|<)\s*(\d+)/);
        record(`SRCH0-P-DATE-${day}`, 'Datumsfilter beginnen lokal um Mitternacht und enden exklusiv am nächsten Kalendertag.',
            { filter, beginn: start.getTime() / 1000, ende: next.getTime() / 1000 },
            Number(lower?.[1]) === start.getTime() / 1000 && upper?.[1] === '<' && Number(upper?.[2]) === next.getTime() / 1000);
    });

    it('verwendet für die Verlustachse deren eigenen Grenzwert', async () => {
        const limits = { max_distance: 900, max_elevation_gain: 800, max_elevation_loss: 700 };
        const filter = await loadFilter({ ...context, surface: 'map' }, { limits }, dataset());
        record('SRCH0-P-LOSS-LIMIT', 'elevationLossLimit entspricht max_elevation_loss und ist unabhängig von max_elevation_gain.',
            { loss_limit: filter.elevationLossLimit, loss_max: filter.elevationLossMax, gain_limit: filter.elevationGainLimit },
            filter.elevationLossLimit === limits.max_elevation_loss);
    });

    it('gibt die beabsichtigte Feldauswahl an die Suchoptionen weiter', async () => {
        const body = await request({ adapter: 'generic-search', q: 'ridge', options: { limit: 10 } });
        const nested = body.options.attributesToRetrieve;
        record('SRCH0-P-RETRIEVAL', 'Die vom Helper ergänzte Feldauswahl liegt in options, das der Proxy an die Engine übergibt.',
            { in_options: nested ?? null, ausserhalb_options: body.attributesToRetrieve ?? null },
            Array.isArray(nested) && nested.length > 0 && body.attributesToRetrieve === undefined);
        expect(nested).toEqual(defaultTrailSearchAttributes);
        expect((await request({ adapter: 'generic-search', q: '', options: { attributesToRetrieve: ['id'] } })).options.attributesToRetrieve).toEqual(['id']);
    });

    it.each([400, 403, 503])('erhält SDK-Status %s auch in Multi-Suche und gemeinsamem Fehlerhandler', async status => {
        const sdkError = new MeilisearchApiError(new Response(null, { status }), { message: 'search failed', code: 'invalid_search_filter', type: 'invalid_request', link: '' });
        vi.spyOn(console, 'error').mockImplementation(() => {});
        const event = apiEvent({ principal: 'anonymous', ms: { multiSearch: async () => { throw sdkError; } }, body: { queries: [] } });
        let actual;
        try { actual = (await multi(event)).status; } catch (error) { actual = routeError(error).status; }
        expect(actual).toBe(status);
        expect(handleError(sdkError).status).toBe(status);
        expect(handleError(new Error('network failed')).status).toBe(500);
    });

    it('verwendet für ungültige Sortwerte gültige Vorgaben', async () => {
        const body = await request({ adapter: 'list-search', filter: { sort: 'unknown', sortOrder: 'raw' } });
        expect(body.options.sort).toEqual(['created:asc']);
        const defaults = await loadFilter({ ...context, surface: 'map' }, {}, dataset());
        expect(sanitizeTrailFilter({ sort: 'unknown', sortOrder: 'raw' }, defaults)).toMatchObject({ sort: 'created', sortOrder: '-' });
    });

    it('meldet fehlenden Actor-Suchtext als Clientfehler', async () => {
        const result = await observeAPI({ ...context, principal: 'alice' }, { adapter: 'api-actor', url: '/api/v1/search/actor' }, dataset());
        expect(result.diagnostics?.status ?? result.status).toBe(400);
    });

    it('übergibt Actor-Limits als Zahlen und weist ungültige Werte ab', async () => {
        const actorContext = { ...context, principal: 'alice' as const };
        const valid = await observeAPI(actorContext, { adapter: 'api-actor', url: '/api/v1/search/actor?q=ali&limit=7' }, dataset());
        expect(valid.engine_requests[0].options).toMatchObject({ limit: 7 });
        for (const limit of ['bad', '-1', '1.5']) {
            const result = await observeAPI(actorContext, { adapter: 'api-actor', url: `/api/v1/search/actor?q=ali&limit=${limit}` }, dataset());
            expect(result.diagnostics?.status ?? result.status).toBe(400);
        }
    });

    it('stellt unbekannte Schwierigkeiten ohne erfundene Schwierigkeitsstufe dar', async () => {
        expect(new Trail('Unbekannte Schwierigkeit').difficulty).toBeUndefined();
        for (const difficulty of [null, undefined, '', 'unknown']) {
            const result = await observeDTO({ adapter: 'trail-dto', ids: ['public-alpine'], hit_overrides: { 'public-alpine': { difficulty } }, fields: ['difficulty'] }, dataset());
            expect(result.dto[0].difficulty, `Difficulty ${String(difficulty)}`).toBeUndefined();
        }
    });

    it('schliesst bei allen Schwierigkeitsstufen auch Unknown ein, bei einer Teilmenge nicht', async () => {
        const all = (await request({ adapter: 'list-search' })).options.filter;
        expect(all).not.toMatch(/difficulty\s+IN/);
        const subset = (await request({ adapter: 'list-search', filter: { difficulty: [0, 2] } })).options.filter;
        expect(subset).toContain('difficulty IN [0,2]');
        expect(subset).not.toContain('difficulty IS NULL');
    });
});
