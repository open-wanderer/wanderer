import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { extname, resolve } from 'node:path';
import { compile } from 'svelte/compiler';
import { build } from 'vite';
import { expect, type Page } from '@playwright/test';

const root = fileURLToPath(new URL('../../../', import.meta.url));

// Keep the real dialogs, controls and API stores; replace unrelated app state.
export async function buildTrailModalFixture(dev: boolean, pairSource?: string) {
    const result = await build({
        root,
        configFile: false,
        logLevel: 'error',
        resolve: { conditions: ['browser', dev ? 'development' : 'production'] },
        plugins: [{
            name: 'trail-modal-fixture',
            enforce: 'pre',
            resolveId(id) {
                if (id.startsWith('fixture:')) return `\0${id}`;
                if (id === '$app/environment') return '\0fixture:environment';
                if (id === '$app/state') return '\0fixture:state';
                if (id === '$app/navigation') return '\0fixture:navigation';
                if (id === 'svelte-i18n') return '\0fixture:i18n';
                if (id === '$lib/stores/toast_store.svelte') return '\0fixture:toast';
                if (id.startsWith('$lib/')) {
                    const path = resolve(root, 'src/lib', id.slice(5));
                    return extname(id) ? path : [path + '.ts', path + '.js', resolve(path, 'index.ts'), resolve(path, 'index.js')].find(existsSync);
                }
            },
            async load(id) {
                if (id === '\0fixture:entry') return `
                    import { mount, unmount } from 'svelte';
                    import Pair from 'fixture:pair.svelte';
                    window.modalCalls = [];
                    const fixture = mount(Pair, { target: document.body });
                    window.unmountModalFixture = () => unmount(fixture);
                    document.body.dataset.modalFixtureReady = 'true';
                `;
                if (id === '\0fixture:environment') return 'export const browser = true;';
                if (id === '\0fixture:state') return 'export const page = { url: new URL(location.href), params: {}, data: {} };';
                if (id === '\0fixture:navigation') return 'export function pushState() {}';
                if (id === '\0fixture:i18n') return `
                    import { writable } from 'svelte/store';
                    export const _ = writable(key => key);
                    export const locale = writable('en');
                `;
                if (id === resolve(root, 'src/lib/stores/user_store.ts')) return `
                    import { writable } from 'svelte/store';
                    export const currentUser = writable({ id: 'tester', actor: 'tester' });
                `;
                if (id === resolve(root, 'src/lib/stores/trail_store.ts')) return `
                    import { writable } from 'svelte/store';
                    export const trail = writable({});
                    export async function trails_show(id) { return { id, name: id }; }
                `;
                if (id === resolve(root, 'src/lib/stores/search_store.ts')) return 'export async function searchActors() { return []; }';
                if (id === '\0fixture:toast') return 'export function show_toast() {}';

                let source: string;
                if (id === '\0fixture:pair.svelte') source = pairSource ?? `
                    <script>
                        import Duplicate from '$lib/components/trail/trail_duplicate_modal.svelte';
                        import Export from '$lib/components/trail/trail_export_modal.svelte';
                        import Share from '$lib/components/trail/trail_share_modal.svelte';
                        import Send from '$lib/components/trail/trail_send_modal.svelte';
                        import Merge from '$lib/components/trail/trail_merge_modal.svelte';
                        import Confirm from '$lib/components/confirm_modal.svelte';
                        import Lists from '$lib/components/list/list_search_modal.svelte';
                        import BulkEdit from '$lib/components/trail/trail_bulk_edit_modal.svelte';
                        import Editor from '$lib/components/base/editor.svelte';
                        import Modal from '$lib/components/base/modal.svelte';
                        const duplicates = {}, exports = {}, shares = {}, sends = {}, merges = {}, confirmations = {}, lists = {}, bulkEdits = {};
                        let namedConfirmation;
                        let editorParent;
                        let alphaText = $state('<p>Alpha content</p>');
                        let betaText = $state('<p>Beta content</p>');
                        const trails = {
                            alpha: { id: 'alpha', name: 'Trail Alpha', public: true },
                            beta: { id: 'beta', name: 'Trail Beta', public: true }
                        };
                        function record(type, trailId, settings, selection) {
                            window.modalCalls.push(JSON.parse(JSON.stringify({ type, trailId, settings, selection })));
                        }
                    </script>
                    {#each ['alpha', 'beta'] as id}
                        <section data-testid={id}>
                            <button data-testid={id + '-duplicate'} onclick={() => duplicates[id].openModal(trails[id], id === 'beta')}>Duplicate</button>
                            <button data-testid={id + '-export'} onclick={() => exports[id].openModal()}>Export</button>
                            <button data-testid={id + '-share'} onclick={() => shares[id].openModal()}>Share</button>
                            <button data-testid={id + '-send'} onclick={() => sends[id].openModal()}>Send</button>
                            <button data-testid={id + '-merge'} onclick={() => merges[id].openModal([trails[id], { id: id + '-source', name: id + ' source' }])}>Merge</button>
                            <button data-testid={id + '-delete'} onclick={() => confirmations[id].openModal()}>Delete</button>
                            <button data-testid={id + '-list'} onclick={() => lists[id].openModal()}>Lists</button>
                            <button data-testid={id + '-bulk-edit'} onclick={() => bulkEdits[id].openModal()}>Bulk edit</button>
                            <Duplicate bind:this={duplicates[id]} onduplicate={settings => record('duplicate', id, settings)} />
                            <Export bind:this={exports[id]} onexport={settings => record('export', id, settings)} />
                            <Share trail={trails[id]} bind:this={shares[id]} onsave={() => record('share', id)} />
                            <Send trail={trails[id]} bind:this={sends[id]} />
                            <Merge bind:this={merges[id]} onmerge={(settings, selection) => record('merge', id, settings, selection)} />
                            <Confirm text={'Delete ' + trails[id].name + '?'} bind:this={confirmations[id]} onconfirm={() => record('delete', id)} oncancel={() => record('cancel-delete', id)} />
                            <Lists lists={[{ id: id + '-list', name: id + ' list', trails: [id] }]} trails={new Set([trails[id]])} bind:this={lists[id]} onchange={list => record('list', id, { listId: list.id })} />
                            <BulkEdit selectedCount={1} bind:this={bulkEdits[id]} onapply={changes => record('bulk-edit', id, changes)} />
                        </section>
                    {/each}
                    <button data-testid="named-confirm" onclick={() => namedConfirmation.openModal()}>Named confirmation</button>
                    <Confirm id="custom-confirm-dialog" text="Named confirmation" bind:this={namedConfirmation} onconfirm={() => record('custom-confirm', 'custom')} />
                    <div data-testid="editor-alpha"><Editor bind:value={alphaText} /></div>
                    <button data-testid="editor-open-parent" onclick={() => editorParent.openModal()}>Open editor parent</button>
                    <Modal id="fixture-editor-parent-modal" title="Summit log" bind:this={editorParent}>
                        {#snippet content()}
                            <div data-testid="editor-beta"><Editor bind:value={betaText} /></div>
                        {/snippet}
                        {#snippet footer({ closeModal })}
                            <button data-testid="editor-close-parent" onclick={closeModal}>Close parent</button>
                        {/snippet}
                    </Modal>
                `;
                else if (id.endsWith('.svelte')) source = await readFile(id, 'utf8');
                else return;
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

export async function mountTrailModalFixture(page: Page, bundle: string) {
    const writes: { path: string; body: Record<string, unknown> }[] = [];
    await page.route('http://trail-modals.test/**', async route => {
        const request = route.request();
        const path = new URL(request.url()).pathname;
        if (path === '/') {
            await route.fulfill({ contentType: 'text/html', body: '<!doctype html><html><body></body></html>' });
            return;
        }
        if (request.method() !== 'GET') writes.push({ path, body: request.postDataJSON() });
        let json: unknown = { items: [] };
        if (path === '/api/v1/plugin-system/plugins') json = { items: [{
            id: 'device', name: 'Test Device', status: 'available',
            capabilities: ['prepare_trail_send.v1'], manifest: { auth: {}, metadata: {} },
        }] };
        else if (path === '/api/v1/plugin-instance') json = { items: [{ plugin_id: 'device', enabled: true }] };
        else if (path === '/api/v1/trail-merge/suggest') {
            const { trailIds } = request.postDataJSON();
            json = { targetTrailId: trailIds[0], candidates: trailIds.map((trailId: string) => ({
                trailId, selectable: true, warnings: [], reason: 'selected_trail',
            })) };
        }
        await route.fulfill({ json });
    });
    await page.goto('http://trail-modals.test/');
    await page.addScriptTag({ content: bundle });
    await expect(page.locator('body[data-modal-fixture-ready="true"]')).toBeAttached();
    return writes;
}
