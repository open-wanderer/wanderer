import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { basename, extname, resolve } from 'node:path';
import { compile } from 'svelte/compiler';
import { build } from 'vite';
import { expect, type Page } from '@playwright/test';

const root = fileURLToPath(new URL('../../../', import.meta.url));
const profile = resolve(root, 'src/routes/profile/[handle]/trails/+page.svelte');

// Exercise the original page and pagination controls without requiring a backend.
// The trail presentations are replaced so their unrelated dependencies stay out.
export async function buildPaginationFixture(dev: boolean) {
    const result = await build({
        root,
        configFile: false,
        logLevel: 'error',
        resolve: { conditions: ['browser', dev ? 'development' : 'production'] },
        plugins: [{
            name: 'pagination-fixture',
            enforce: 'pre',
            resolveId(id) {
                if (id.startsWith('fixture:')) return `\0${id}`;
                if (id === '$app/state') return '\0fixture:state';
                if (id === '$app/navigation') return '\0fixture:navigation';
                if (id === 'svelte-i18n') return '\0fixture:i18n';
                if (id === '$lib/stores/profile_store.js') return '\0fixture:store';
                if (id === '$lib/stores/toast_store.svelte.js') return '\0fixture:toast';
                if (id.startsWith('$lib/')) return resolve(root, 'src/lib', id.slice(5) + (extname(id) ? '' : '.ts'));
            },
            async load(id) {
                if (id === '\0fixture:entry') return `
                    import { mount } from 'svelte';
                    import Profile from ${JSON.stringify(profile)};
                    mount(Profile, { target: document.body, props: { data: {
                        trails: { page: 1, totalPages: 10, items: [] },
                        filter: { q: '', sort: 'name', sortOrder: '+' }
                    } } });
                `;
                if (id === '\0fixture:state') return `export const page = { params: { handle: 'tester' } };`;
                if (id === '\0fixture:navigation') return 'export function goto() {}';
                if (id === '\0fixture:toast') return 'export function show_toast() {}';
                if (id === '\0fixture:i18n') return `
                    export const _ = { subscribe(fn) { fn(key => key); return () => {}; } };
                `;
                if (id === '\0fixture:store') return `
                    export async function profile_trails_index(handle, filter, page, items) {
                        const response = await fetch('/fixture/trails?page=' + page + '&items=' + items);
                        if (!response.ok) throw new Error('Failed to load trails');
                        return response.json();
                    }
                `;
                if (!id.endsWith('.svelte')) return;
                let source: string;
                switch (basename(id)) {
                    case 'trail_card.svelte':
                    case 'trail_list_item.svelte':
                        source = '<script>let { trail } = $props();</script><span data-testid="trail-id">{trail.id}</span>';
                        break;
                    case 'trail_table.svelte':
                        source = '<script>let { trails } = $props();</script>{#each trails ?? [] as trail}<span data-testid="trail-id">{trail.id}</span>{/each}';
                        break;
                    case 'skeleton_card.svelte':
                    case 'skeleton_list_item.svelte':
                        source = '<span data-testid="loading"></span>';
                        break;
                    case 'empty_state_search.svelte':
                    case 'trail_dropdown.svelte':
                        source = '';
                        break;
                    default:
                        source = await readFile(id, 'utf8');
                }
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

export async function mountPaginationFixture(
    page: Page,
    bundle: string,
    display = 'cards',
    items = 12,
) {
    const requests: { page: number; items: number }[] = [];
    const response = { total: 120, gate: Promise.resolve() };
    await page.route('http://pagination.test/**', route => route.fulfill({
        contentType: 'text/html', body: '<!doctype html><html><body></body></html>',
    }));
    await page.route('**/fixture/trails?**', async route => {
        const query = new URL(route.request().url()).searchParams;
        const page = Number(query.get('page'));
        const items = Number(query.get('items'));
        requests.push({ page, items });
        await response.gate;
        await route.fulfill({ json: {
            page,
            totalPages: Math.ceil(response.total / items),
            items: Array.from({ length: Math.max(0, Math.min(items, response.total - (page - 1) * items)) }, (_, index) => ({
                id: String((page - 1) * items + index + 1), author: 'tester',
            })),
        } });
    });
    await page.goto('http://pagination.test/');
    await page.evaluate(({ display, items }) => {
        localStorage.setItem('displayOption', display);
        localStorage.setItem('paginationItems', String(items));
    }, { display, items });
    await page.addScriptTag({ content: bundle });
    await expect(page.getByTestId('trail-id')).toHaveCount(items);
    return { requests, response };
}
