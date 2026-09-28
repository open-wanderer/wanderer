import { expect, test } from '@playwright/test';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createServer, type ViteDevServer } from 'vite';

// These parser tests need a real browser, but no application server or account.
test.use({ storageState: { cookies: [], origins: [] } });

let server: ViteDevServer | undefined;
let cacheDir: string | undefined;
let serverUrl: string;

test.beforeAll(async () => {
    const root = fileURLToPath(new URL('../../../../', import.meta.url));
    cacheDir = await mkdtemp(join(tmpdir(), 'wanderer-gpx-playwright-'));
    server = await createServer({
        configFile: false,
        root,
        cacheDir,
        appType: 'custom',
        logLevel: 'error',
        resolve: { alias: { $lib: join(root, 'src/lib') } },
        server: { host: '127.0.0.1', port: 0 },
    });
    server.middlewares.use((request, response, next) => {
        if (request.url === '/') {
            response.setHeader('content-type', 'text/html');
            response.end('<!doctype html><title>GPX parser tests</title>');
        } else {
            next();
        }
    });
    await server.listen();
    const address = server.httpServer?.address();
    if (!address || typeof address === 'string') {
        throw new Error('Missing GPX test server address');
    }
    serverUrl = `http://127.0.0.1:${address.port}`;
});

test.afterAll(async () => {
    await server?.close();
    if (cacheDir) {
        await rm(cacheDir, { recursive: true, force: true });
    }
});

test.beforeEach(async ({ page }) => {
    await page.goto(serverUrl);
});

test('ignores parsed comments and preserves comment-like CDATA', async ({ page }) => {
    const result = await page.evaluate(async () => {
        const modulePath = '/src/lib/models/gpx/gpx.ts';
        const { default: GPX } = await import(modulePath);
        const description = '<scr<!-- keep -->ipt>text</script>';
        const gpx = GPX.parse('<?xml version="1.0"?><!-- before root -->' +
            '<gpx><metadata><name>Before<!-- ignored -->After</name>' +
            `<desc><![CDATA[${description}]]></desc></metadata><!-- end --></gpx>`);
        return {
            name: gpx.metadata?.name,
            description: gpx.metadata?.desc,
            roundTripDescription: GPX.parse(gpx.toString()).metadata?.desc,
        };
    });

    expect(result).toEqual({
        name: 'BeforeAfter',
        description: '<scr<!-- keep -->ipt>text</script>',
        roundTripDescription: '<scr<!-- keep -->ipt>text</script>',
    });
});

test('round-trips GPX prefixes and Garmin sensor extensions', async ({ page }) => {
    const result = await page.evaluate(async () => {
        const modulePath = '/src/lib/models/gpx/gpx.ts';
        const { default: GPX } = await import(modulePath);
        const garminNamespace = 'http://www.garmin.com/xmlschemas/TrackPointExtension/v1';
        const gpx = GPX.parse('<g:gpx xmlns:g="http://www.topografix.com/GPX/1/1" ' +
            `xmlns:ns3="${garminNamespace}"><g:trk><g:trkseg>` +
            '<g:trkpt lat="47" lon="8"><g:extensions><ns3:TrackPointExtension>' +
            '<ns3:hr>82</ns3:hr><ns3:atemp>2.0</ns3:atemp>' +
            '</ns3:TrackPointExtension></g:extensions></g:trkpt>' +
            '<g:trkpt lat="47.01" lon="8.01"/></g:trkseg></g:trk></g:gpx>');
        const output = gpx.toString();
        const document = new DOMParser().parseFromString(output, 'application/xml');
        return {
            points: GPX.parse(output).flatten().length,
            heartRate: document.getElementsByTagNameNS(garminNamespace, 'hr').item(0)?.textContent,
            temperature: document.getElementsByTagNameNS(garminNamespace, 'atemp').item(0)?.textContent,
            hasGarminPrefix: output.includes('<ns3:hr>'),
            hasGPXPrefix: output.includes('<g:trkpt'),
        };
    });

    expect(result).toEqual({
        points: 2,
        heartRate: '82',
        temperature: '2.0',
        hasGarminPrefix: true,
        hasGPXPrefix: false,
    });
});

test('normalizes empty namespaces without modifying text or attributes', async ({ page }) => {
    const result = await page.evaluate(async () => {
        const modulePath = '/src/lib/models/gpx/gpx.ts';
        const { default: GPX } = await import(modulePath);
        const gpx = GPX.parse('<gpx xmlns="" creator=\'text xmlns=""\'>' +
            '<metadata xmlns=""><name>text xmlns=""</name>' +
            '<desc><![CDATA[text xmlns=""]]></desc></metadata></gpx>');
        return {
            namespace: gpx.$.xmlns,
            creator: gpx.$.creator,
            name: gpx.metadata?.name,
            description: gpx.metadata?.desc,
        };
    });

    expect(result).toEqual({
        namespace: 'http://www.topografix.com/GPX/1/1',
        creator: 'text xmlns=""',
        name: 'text xmlns=""',
        description: 'text xmlns=""',
    });
});

test('preserves track point order across interleaved GPX prefixes', async ({ page }) => {
    const latitudes = await page.evaluate(async () => {
        const modulePath = '/src/lib/models/gpx/gpx.ts';
        const { default: GPX } = await import(modulePath);
        const namespace = 'http://www.topografix.com/GPX/1/1';
        const gpx = GPX.parse(`<gpx xmlns="${namespace}" xmlns:a="${namespace}" xmlns:b="${namespace}"><trk><trkseg>` +
            '<a:trkpt lat="47.01" lon="8"/><trkpt lat="47.02" lon="8"/>' +
            '<b:trkpt lat="47.03" lon="8"/><a:trkpt lat="47.04" lon="8"/>' +
            '<trkpt lat="47.05" lon="8"/></trkseg></trk></gpx>');
        const readLatitudes = (parsed: typeof gpx) => parsed.flatten().map((point: { $: { lat: number } }) => point.$.lat);
        return [readLatitudes(gpx), readLatitudes(GPX.parse(gpx.toString()))];
    });

    expect(latitudes).toEqual([
        [47.01, 47.02, 47.03, 47.04, 47.05],
        [47.01, 47.02, 47.03, 47.04, 47.05],
    ]);
});

test('rejects native parser errors even when the document retains its GPX root', async ({ page }) => {
    const result = await page.evaluate(async () => {
        const modulePath = '/src/lib/models/gpx/gpx.ts';
        const { default: GPX } = await import(modulePath);
        const invalidSources = [
            '<gpx><name>unterminated</gpx>',
            '<gpx><extensions><gpxtpx:hr>90</gpxtpx:hr></extensions></gpx>',
            '<gpx><metadata><desc>&nbsp;</desc></metadata></gpx>',
            '<gpx><!-- unterminated</gpx>',
        ];
        const rejected = invalidSources.map((source) => {
            try {
                GPX.parse(source);
                return false;
            } catch {
                return true;
            }
        });
        // An ordinary element of the same name is not a browser error marker.
        GPX.parse('<gpx><extensions><parsererror>data</parsererror></extensions></gpx>');
        return rejected;
    });

    expect(result).toEqual([true, true, true, true]);
});

test('parses each GPX input exactly once with the native XML parser', async ({ page }) => {
    const parseCalls = await page.evaluate(async () => {
        const modulePath = '/src/lib/models/gpx/gpx.ts';
        // Import first: isomorphic-xml2js probes the browser's error format once.
        const { default: GPX } = await import(modulePath);
        const NativeDOMParser = globalThis.DOMParser;
        let calls = 0;
        globalThis.DOMParser = class extends NativeDOMParser {
            parseFromString(...args: Parameters<DOMParser['parseFromString']>) {
                calls++;
                return super.parseFromString(...args);
            }
        };
        try {
            GPX.parse('\uFEFF<?xml version="1.0"?><!-- before root --><gpx/>');
            return calls;
        } finally {
            globalThis.DOMParser = NativeDOMParser;
        }
    });

    expect(parseCalls).toBe(1);
});
