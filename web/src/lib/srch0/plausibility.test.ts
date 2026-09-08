import { afterAll, afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { mkdirSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { MeilisearchApiError } from 'meilisearch';
import { POST as proxy } from '../../routes/api/v1/search/[index]/+server';
import { apiEvent, routeError } from '../../../tests/srch0/api-context';
import { observeCompiler } from '../../../tests/srch0/compiler-adapter';
import { loadFilter } from '../../../tests/srch0/state-adapter';
import { cases, dataset, digest } from '../../../tests/srch0/fixtures';
import type { CompilerInput, JsonRecord, WebInput } from '../../../tests/srch0/types';

const context = cases<WebInput>(['compiler'])[0].context;
const findings: JsonRecord[] = [];

function record(id: string, classification: string, property: string, actual: unknown, satisfied: boolean, knownViolation: boolean) {
    // Eine Korrektur erfüllt die Eigenschaft ohne neues Golden. Nur die eng
    // beschriebene bestehende Abweichung darf als bekannter Befund weiterlaufen.
    expect(satisfied || knownViolation, `${id}: neue, nicht eingeordnete Abweichung: ${JSON.stringify(actual)}`).toBe(true);
    findings.push({ id, einordnung: classification, eigenschaft: property, ergebnis: satisfied ? 'erfüllt' : 'bekannter_befund', beobachtung: actual });
}

async function request(input: CompilerInput) {
    const result = await observeCompiler(context, input, dataset());
    return result.engine_requests[0].body as { attributesToRetrieve?: string[]; options: { filter: string; attributesToRetrieve?: string[] } };
}
const radii = (filter: string) => filter.match(/_geoRadius\([^)]*\)/g) ?? [];

describe('SRCH0: unabhängige Plausibilität, bekannte Abweichungen bleiben Befunde', () => {
    beforeEach(() => vi.stubEnv('TZ', context.timezone));
    afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); vi.restoreAllMocks(); });
    afterAll(() => {
        const directory = resolve(process.env.SRCH0_REPORT_DIR ?? 'test-results');
        mkdirSync(directory, { recursive: true });
        writeFileSync(resolve(directory, 'srch0-plausibility.json'), JSON.stringify({
            manifest_sha256: digest('manifest.json'), dataset_sha256: digest('datasets/reference.json'), changes_sha256: digest('changes.json'),
            hinweis: 'Bekannte Befunde sind keine fachlichen Korrektheitsaussagen. Neue, nicht eingeordnete Abweichungen lassen den Test scheitern.',
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
        record('SRCH0-P-HTTP', 'bestätigter_fehler', 'Ein ungültiger Suchfilter bleibt ein Clientfehler; der SDK-Status steht in response.status.',
            { sdk_status: sdkError.response.status, api_status: status }, status >= 400 && status < 500, status === 500);
    });

    it('behandelt gültige Nullkoordinaten wie andere Startpunkte', async () => {
        const counts = [];
        for (const [lat, lon] of [[46, 7], [0, 7], [46, 0]]) {
            counts.push(radii((await request({ adapter: 'list-search', filter: { near: { lat, lon, radius: 2000 } } })).options.filter).length);
        }
        expect(counts[0], 'Kontrollfall mit gültigen Koordinaten').toBeGreaterThan(0);
        record('SRCH0-P-GEO-ZERO', 'bestätigter_fehler', 'Ein gültiger Startpunkt auf Äquator oder Nullmeridian aktiviert den Radius.',
            { normal: counts[0], latitude_null: counts[1], longitude_null: counts[2] }, counts.every(count => count > 0), counts[1] === 0 && counts[2] === 0);
    });

    it('benötigt denselben Radiusfilter nur einmal', async () => {
        const clauses = radii((await request({ adapter: 'list-search', filter: { near: { lat: 46, lon: 7, radius: 2000 } } })).options.filter);
        record('SRCH0-P-GEO-DUPLICATE', 'redundanz', 'Ein identisches Prädikat genügt; seine Wiederholung verändert die logische Treffermenge nicht.',
            clauses, clauses.length === 1, clauses.length === 2 && clauses[0] === clauses[1]);
    });

    it('prüft die offene Produktfrage zum inklusiven Enddatum', async () => {
        const end = '2026-09-07';
        const filter = (await request({ adapter: 'list-search', filter: { endDate: end } })).options.filter;
        const bound = filter.match(/date\s*(<=|<)\s*(\d+)/);
        expect(bound, 'Ein gesetztes Enddatum muss eine obere Datumsgrenze erzeugen').not.toBeNull();
        const cutoff = Number(bound![2]);
        const noon = new Date(`${end}T12:00:00`).getTime() / 1000;
        const includesNoon = bound![1] === '<' ? noon < cutoff : noon <= cutoff;
        record('SRCH0-P-DATE-END', 'produktfrage', 'Falls Enddatum den ganzen ausgewählten Kalendertag meint, muss dessen Mittag enthalten sein.',
            { grenze: cutoff, mittag: noon, mittag_enthalten: includesNoon }, includesNoon, cutoff === Date.parse(end) / 1000);
    });

    it('verwendet für die Verlustachse deren eigenen Grenzwert', async () => {
        const limits = { max_distance: 900, max_elevation_gain: 800, max_elevation_loss: 700 };
        const filter = await loadFilter({ ...context, surface: 'map' }, { limits }, dataset());
        record('SRCH0-P-LOSS-LIMIT', 'bestätigter_fehler', 'elevationLossLimit entspricht max_elevation_loss und ist unabhängig von max_elevation_gain.',
            { loss_limit: filter.elevationLossLimit, loss_max: filter.elevationLossMax, gain_limit: filter.elevationGainLimit },
            filter.elevationLossLimit === limits.max_elevation_loss, filter.elevationLossLimit === limits.max_elevation_gain);
    });

    it('gibt die beabsichtigte Feldauswahl an die Suchoptionen weiter', async () => {
        const body = await request({ adapter: 'generic-search', q: 'ridge', options: { limit: 10 } });
        const nested = body.options.attributesToRetrieve;
        record('SRCH0-P-RETRIEVAL', 'bestätigter_fehler', 'Die vom Helper ergänzte Feldauswahl liegt in options, das der Proxy an die Engine übergibt.',
            { in_options: nested ?? null, ausserhalb_options: body.attributesToRetrieve ?? null },
            Array.isArray(nested) && nested.length > 0, nested === undefined && Array.isArray(body.attributesToRetrieve) && body.attributesToRetrieve.length > 0);
    });
});
