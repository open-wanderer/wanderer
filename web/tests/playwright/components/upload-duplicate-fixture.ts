import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { basename, extname, resolve } from 'node:path';
import { compile, compileModule } from 'svelte/compiler';
import { ModuleKind, ScriptTarget, transpileModule } from 'typescript';
import { build } from 'vite';
import { expect, type Page } from '@playwright/test';

const root = fileURLToPath(new URL('../../../', import.meta.url));

type Scope = { includePublic?: boolean; includeShared?: boolean };
export type FixtureSettings = { id: string; uploadDuplicateCheck?: Scope | null };
export type Duplicate = { id: string; name: string; author: string; domain: string };

declare global {
    interface Window {
        uploadDuplicateInitialSettings: FixtureSettings;
        uploadDuplicateFixture: {
            setSettings(settings: FixtureSettings): Promise<void>;
            remount(): Promise<void>;
            toasts: { type: string; text: string }[];
            navigations: string[];
            invalidations: number;
            uploads(): { status: string; duplicate?: Duplicate }[];
        };
    }
}

// Keep the settings page, controls, settings_update, upload store and dialog real.
// Only unrelated export code, app context and the upload HTTP boundary are replaced.
export async function buildUploadDuplicateFixture(dev: boolean) {
    const translations = await readFile(resolve(root, 'src/lib/i18n/locales/en.json'), 'utf8');
    const stubs: Record<string, string> = {
        '$app/state': 'state',
        '$app/navigation': 'navigation',
        'svelte-i18n': 'i18n',
        '$lib/stores/theme_store': 'theme',
        '$lib/stores/user_store': 'user',
        '$lib/stores/toast_store.svelte': 'toast',
        '$lib/stores/trail_store': 'trails',
        '$lib/util/api_util': 'api',
        '$lib/util/file_util': 'files',
        '$lib/util/gpx_util': 'gpx',
        '$lib/vendor/toGeoJSON/toGeoJSON': 'geojson',
        'jszip': 'zip',
    };
    const result = await build({
        root,
        configFile: false,
        logLevel: 'error',
        resolve: { conditions: ['browser', dev ? 'development' : 'production'] },
        plugins: [{
            name: 'upload-duplicate-fixture',
            enforce: 'pre',
            resolveId(id) {
                if (id.startsWith('fixture:')) return `\0${id}`;
                if (stubs[id]) return `\0fixture:${stubs[id]}`;
                if (id === '$lib/stores/upload_store.svelte') return resolve(root, 'src/lib/stores/upload_store.svelte.ts');
                if (id.startsWith('$lib/')) return resolve(root, 'src/lib', id.slice(5) + (extname(id) ? '' : '.ts'));
            },
            async load(id) {
                if (id === '\0fixture:entry') return `
                    import { mount, unmount, tick } from 'svelte';
                    import Settings from ${JSON.stringify(resolve(root, 'src/routes/settings/export/+page.svelte'))};
                    import UploadDialog from ${JSON.stringify(resolve(root, 'src/lib/components/settings/upload_dialog.svelte'))};
                    import { uploadStore } from '$lib/stores/upload_store.svelte';
                    import { page } from '$app/state';
                    const settingsTarget = document.body.appendChild(document.createElement('main'));
                    const dialogTarget = document.body.appendChild(document.createElement('aside'));
                    let settings;
                    window.uploadDuplicateFixture = {
                        toasts: [], navigations: [], invalidations: 0,
                        async setSettings(value) { page.data.settings = value; await tick(); },
                        async remount() {
                            await unmount(settings);
                            settings = mount(Settings, { target: settingsTarget });
                            await tick();
                        },
                        uploads() { return uploadStore.completedUploads.map(u => ({ status: u.status, duplicate: u.duplicate })); }
                    };
                    settings = mount(Settings, { target: settingsTarget });
                    mount(UploadDialog, { target: dialogTarget });
                `;
                if (id === '\0fixture:state') return compileModule(
                    'export const page = $state({ data: { settings: window.uploadDuplicateInitialSettings } });',
                    { filename: 'fixture-state.svelte.js', generate: 'client', dev },
                ).js.code;
                if (id === '\0fixture:navigation') return `
                    import { page } from '$app/state';
                    export function goto(url) { window.uploadDuplicateFixture.navigations.push(url); }
                    export async function invalidateAll() {
                        window.uploadDuplicateFixture.invalidations++;
                        page.data.settings = await (await fetch('/fixture/settings')).json();
                    }
                `;
                if (id === '\0fixture:i18n') return String.raw`
                    const messages = ${translations};
                    export const _ = { subscribe(fn) {
                        fn((key, options) => (messages[key] ?? key).replace(/\{(\w+)\}/g, (_, name) => options?.values?.[name] ?? name));
                        return () => {};
                    } };
                `;
                if (id === '\0fixture:theme') return "import { writable } from 'svelte/store'; export const theme = writable('light');";
                if (id === '\0fixture:user') return "import { writable } from 'svelte/store'; export const currentUser = writable({ id: 'user', actor: 'own-actor' });";
                if (id === '\0fixture:toast') return 'export function show_toast(toast) { window.uploadDuplicateFixture.toasts.push(toast); }';
                if (id === '\0fixture:api') return `
                    export class APIError extends Error {
                        constructor(status, message, detail) { super(message); this.status = status; this.detail = detail; }
                    }
                `;
                if (id === '\0fixture:trails') return `
                    import { APIError } from '$lib/util/api_util';
                    export async function trails_upload(file, ignoreDuplicates, onProgress) {
                        const response = await fetch('/fixture/upload', {
                            method: 'POST', body: JSON.stringify({ filename: file.name, ignoreDuplicates })
                        });
                        const body = await response.json();
                        if (!response.ok) throw new APIError(response.status, body.message, body);
                        onProgress?.(100);
                        return body;
                    }
                    export function fetchGPX() {} export function trails_index() {}
                `;
                if (id === '\0fixture:files') return 'export function getFileURL() {} export function saveAs() {}';
                if (id === '\0fixture:gpx') return 'export function trail2gpx() {}';
                if (id === '\0fixture:geojson') return 'export function gpx() {}';
                if (id === '\0fixture:zip') return 'export default class JSZip {}';
                if (id.endsWith('.svelte.ts')) {
                    const source = transpileModule(await readFile(id, 'utf8'), {
                        compilerOptions: { target: ScriptTarget.ESNext, module: ModuleKind.ESNext },
                    }).outputText;
                    return compileModule(source, { filename: id, generate: 'client', dev }).js.code;
                }
                if (!id.endsWith('.svelte')) return;
                const source = basename(id) === 'trail_export_modal.svelte'
                    ? '<script>export function openModal() {}</script>'
                    : await readFile(id, 'utf8');
                return compile(source, { filename: id, generate: 'client', dev, css: 'injected' }).js.code;
            },
        }],
        build: {
            write: false,
            minify: false,
            rolldownOptions: { input: 'fixture:entry', output: { format: 'iife' } },
        },
    });
    if (Array.isArray(result) || !('output' in result)) throw new Error('Expected a single bundle');
    const chunk = result.output.find(output => output.type === 'chunk');
    if (!chunk) throw new Error('Missing fixture bundle');
    return chunk.code;
}

export async function mountUploadDuplicateFixture(page: Page, bundle: string, settings: FixtureSettings = { id: 'settings-a' }) {
    const server = {
        settings,
        status: 200,
        gate: Promise.resolve(),
        duplicate: { id: 'duplicate', name: 'Private morning walk', author: 'own-actor', domain: 'alice' } as Duplicate,
    };
    const saves: FixtureSettings[] = [];
    const uploads: { filename: string; ignoreDuplicates: boolean }[] = [];
    await page.route('http://upload-duplicate.test/**', route => route.fulfill({
        contentType: 'text/html', body: '<!doctype html><html><body></body></html>',
    }));
    await page.route('**/fixture/settings', route => route.fulfill({ json: server.settings }));
    await page.route('**/api/v1/settings/*', async route => {
        const patch = route.request().postDataJSON() as FixtureSettings;
        saves.push(patch);
        await server.gate;
        if (server.status !== 200) {
            await route.fulfill({ status: server.status, json: { message: 'Save failed' } });
            return;
        }
        if (server.settings.id === patch.id) server.settings = { ...server.settings, ...patch };
        await route.fulfill({ json: patch });
    });
    await page.route('**/fixture/upload', async route => {
        const request = route.request().postDataJSON();
        uploads.push(request);
        await route.fulfill(request.ignoreDuplicates
            ? { json: { id: 'created-trail' } }
            : { status: 400, json: { message: 'Duplicate trail', ...server.duplicate } });
    });
    await page.goto('http://upload-duplicate.test/');
    await page.evaluate(settings => { window.uploadDuplicateInitialSettings = settings; }, settings);
    await page.addScriptTag({ content: bundle });
    await expect(page.getByRole('heading', { name: 'Check for duplicates during import' })).toBeVisible();
    return {
        server, saves, uploads,
        async refresh(settings: FixtureSettings) {
            server.settings = settings;
            await page.evaluate(settings => window.uploadDuplicateFixture.setSettings(settings), settings);
        },
        async remount() { await page.evaluate(() => window.uploadDuplicateFixture.remount()); },
    };
}
