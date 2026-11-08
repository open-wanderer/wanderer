import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { extname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { compile, compileModule } from 'svelte/compiler';
import { build, transformWithOxc } from 'vite';

const root = fileURLToPath(new URL('../../../', import.meta.url));

// Mount the editor and its real form, selection handler and EXIF reader.
// Unrelated controls and the map are replaced by inert components.
export async function buildDevicePhotoFixture(initialPhotos?: { saved: number; pending: number }) {
    const result = await build({
        root,
        configFile: false,
        logLevel: 'error',
        resolve: { conditions: ['browser', 'production'] },
        plugins: [{
            name: 'device-photo-fixture',
            enforce: 'pre',
            resolveId(id) {
                if (id.startsWith('fixture:')) return `\0${id}`;
                if (id.startsWith('$app/')) return `\0fixture:${id}`;
                if (id === 'svelte-i18n') return '\0fixture:i18n';
                if (id.startsWith('#lib/components/')) return '\0fixture:control.svelte';
                if (id === '#lib/stores/toast_store.svelte.ts') return '\0fixture:toast';
                if (id === '#lib/stores/user_store.ts') return '\0fixture:user';
                if (id === '#lib/stores/theme_store.ts') return '\0fixture:theme';
                if (id.startsWith('#lib/')) {
                    const path = resolve(root, 'src/lib', id.slice(5));
                    if (extname(id)) return path.endsWith('.js') && !existsSync(path) ? path.slice(0, -3) + '.ts' : path;
                    return [path + '.ts', path + '.js', resolve(path, 'index.ts')].find(existsSync);
                }
            },
            async load(id) {
                if (id === '\0fixture:entry') return `
                    import { mount } from 'svelte';
                    import Editor from ${JSON.stringify(resolve(root, 'src/routes/trail/edit/[id]/+page.svelte'))};
                    import { Trail } from '#lib/models/trail.ts';
                    import { Waypoint } from '#lib/models/waypoint.ts';
                    import { extractDevicePhotoExif } from '#lib/util/exif_util.ts';
                    window.photoToasts = [];
                    window.readDevicePhoto = bytes => extractDevicePhotoExif(new Blob([new Uint8Array(bytes)]));
                    const initialPhotos = ${JSON.stringify(initialPhotos ?? null)};
                    const waypoints = [];
                    if (initialPhotos) {
                        const waypoint = new Waypoint(46.5, -7.25, {
                            id: 'waypoint0000001',
                            photos: Array.from({ length: initialPhotos.saved }, (_, index) => 'saved-' + index + '.jpg'),
                        });
                        waypoint._photos = Array.from({ length: initialPhotos.pending }, (_, index) => new File(['photo'], 'pending-' + index + '.jpg'));
                        waypoints.push(waypoint);
                    }
                    const trail = new Trail('Photo trail', { waypoints });
                    trail.completed = true;
                    mount(Editor, { target: document.body, props: { data: {
                        trail, lists: { items: [] }, categories: [], categoryPreferences: [],
                    } } });
                    document.body.dataset.photoFixtureReady = 'true';
                `;
                if (id === '\0fixture:$app/env') return 'export const browser = true;';
                if (id === '\0fixture:$app/env/public') return 'export const PUBLIC_POCKETBASE_URL = "https://a.example"; export const PUBLIC_MAP_MAX_POLYLINES = undefined; export const PUBLIC_PRIVATE_INSTANCE = undefined;';
                if (id === '\0fixture:$app/state') return 'export const page = { url: new URL(location.href), params: { id: "new" }, data: {} };';
                if (id === '\0fixture:$app/navigation') return 'export function goto() {}';
                if (id === '\0fixture:i18n') return `
                    import { writable } from 'svelte/store';
                    export const _ = writable((key, options) => options?.values ? key + ' ' + JSON.stringify(options.values) : key);
                    export const locale = writable('en');
                `;
                if (id === '\0fixture:user') return 'import { writable } from "svelte/store"; export const currentUser = writable({ id: "photo-user", actor: "photo-actor" });';
                if (id === '\0fixture:theme') return 'import { writable } from "svelte/store"; export const theme = writable("light");';
                if (id === '\0fixture:toast') return 'export function show_toast(toast, duration) { window.photoToasts.push({ ...toast, duration }); }';
                if (id.endsWith('.svelte.ts')) return compileModule((await transformWithOxc(await readFile(id, 'utf8'), id, { lang: 'ts', target: 'esnext' })).code, { filename: id, generate: 'client', dev: false }).js.code;
                if (id.endsWith('.svelte')) {
                    const source = id === '\0fixture:control.svelte' ? `
                        <script>
                            let { waypoint, value = $bindable(), photos = $bindable(), thumbnail = $bindable(),
                                photoFiles = $bindable(), map = $bindable(), options = $bindable(),
                                showWaypoints = $bindable() } = $props();
                        </script>
                        {#if waypoint}
                            <output data-testid="photo-waypoint" data-lat={waypoint.lat} data-lon={waypoint.lon}
                                data-saved-count={waypoint.photos.length} data-pending-count={waypoint._photos?.length ?? 0}>
                                {#each waypoint._photos ?? [] as photo}<span>{photo.name}</span>{/each}
                            </output>
                        {/if}
                    ` : await readFile(id, 'utf8');
                    return compile(source, { filename: id, generate: 'client', dev: false, css: 'injected' }).js.code;
                }
            },
        }],
        build: { write: false, minify: false, rolldownOptions: { input: 'fixture:entry', output: { format: 'iife' } } },
    });
    if (Array.isArray(result) || !('output' in result)) throw new Error('Expected one photo fixture bundle');
    const chunk = result.output.find(output => output.type === 'chunk');
    if (!chunk) throw new Error('Missing photo fixture bundle');
    return chunk.code;
}

function u16(value: number) { const bytes = Buffer.alloc(2); bytes.writeUInt16LE(value); return bytes; }
function u32(value: number) { const bytes = Buffer.alloc(4); bytes.writeUInt32LE(value); return bytes; }
function be16(value: number) { const bytes = Buffer.alloc(2); bytes.writeUInt16BE(value); return bytes; }
function be32(value: number) { const bytes = Buffer.alloc(4); bytes.writeUInt32BE(value); return bytes; }

// Minimal metadata containers, with the same position in every format.
function positionTIFF() {
    const gps = [
        { tag: 1, type: 2, count: 2, bytes: Buffer.from('N\0') },
        { tag: 2, type: 5, count: 3, bytes: Buffer.concat([46, 1, 30, 1, 0, 1].map(u32)) },
        { tag: 3, type: 2, count: 2, bytes: Buffer.from('W\0') },
        { tag: 4, type: 5, count: 3, bytes: Buffer.concat([7, 1, 15, 1, 0, 1].map(u32)) },
    ];
    const gpsOffset = 26;
    let offset = gpsOffset + 2 + 12 * gps.length + 4;
    const data: Buffer[] = [];
    const entries = gps.map(entry => {
        const value = Buffer.alloc(4);
        if (entry.bytes.length <= 4) entry.bytes.copy(value);
        else { value.writeUInt32LE(offset); offset += entry.bytes.length; data.push(entry.bytes); }
        return Buffer.concat([u16(entry.tag), u16(entry.type), u32(entry.count), value]);
    });
    return Buffer.concat([
        Buffer.from('II'), u16(42), u32(8), u16(1),
        u16(0x8825), u16(4), u32(1), u32(gpsOffset), u32(0),
        u16(gps.length), ...entries, u32(0), ...data,
    ]);
}

function box(type: string, content: Buffer) { return Buffer.concat([be32(8 + content.length), Buffer.from(type), content]); }

export function devicePhoto(format: 'jpeg' | 'png' | 'webp' | 'heic' | 'heif'): Buffer {
    const tiff = positionTIFF();
    if (format === 'jpeg') {
        const exif = Buffer.concat([Buffer.from('Exif\0\0'), tiff]);
        return Buffer.concat([Buffer.from([0xff, 0xd8, 0xff, 0xe1]), be16(exif.length + 2), exif, Buffer.from([0xff, 0xd9])]);
    }
    if (format === 'png') {
        const chunk = (type: string, data: Buffer) => Buffer.concat([be32(data.length), Buffer.from(type), data, be32(0)]);
        return Buffer.concat([Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]), chunk('IHDR', Buffer.concat([be32(1), be32(1), Buffer.from([8, 2, 0, 0, 0])])), chunk('eXIf', tiff), chunk('IEND', Buffer.alloc(0))]);
    }
    if (format === 'webp') {
        const exif = Buffer.concat([Buffer.from('EXIF'), u32(tiff.length), tiff, ...(tiff.length % 2 ? [Buffer.from([0])] : [])]);
        return Buffer.concat([Buffer.from('RIFF'), u32(4 + exif.length), Buffer.from('WEBP'), exif]);
    }
    const ftyp = box('ftyp', Buffer.concat([Buffer.from(format === 'heic' ? 'heic' : 'mif1'), be32(0), Buffer.from('heicmif1')]));
    const infe = box('infe', Buffer.concat([Buffer.from([2, 0, 0, 0]), be16(1), be16(0), Buffer.from('Exif\0')]));
    const iinf = box('iinf', Buffer.concat([be32(0), be16(1), infe]));
    const iloc = (offset: number) => box('iloc', Buffer.concat([be32(0), Buffer.from([0x44, 0]), be16(1), be16(1), be16(0), be16(1), be32(offset), be32(tiff.length + 4)]));
    const meta = (offset: number) => box('meta', Buffer.concat([be32(0), iinf, iloc(offset)]));
    const dataOffset = ftyp.length + meta(0).length + 8;
    return Buffer.concat([ftyp, meta(dataOffset), box('mdat', Buffer.concat([be32(0), tiff]))]);
}
