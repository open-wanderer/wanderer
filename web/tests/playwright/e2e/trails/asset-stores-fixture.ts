import { existsSync } from 'node:fs';
import { extname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { build } from 'vite';

// The editor has no direct action for moving a saved photo between targets.
// Exercise its production stores in the browser against the same live API.
export async function buildAssetStoresFixture(): Promise<string> {
    const root = fileURLToPath(new URL('../../../../', import.meta.url));
    const result = await build({
        root, configFile: false, logLevel: 'error',
        resolve: { conditions: ['browser', 'production'] },
        plugins: [{
            name: 'asset-stores-fixture', enforce: 'pre',
            resolveId(id) {
                if (id === 'fixture:entry') return '\0fixture:entry';
                if (id.startsWith('$app/')) return `\0fixture:${id}`;
                if (id.startsWith('#lib/')) {
                    const path = resolve(root, 'src/lib', id.slice(5));
                    return extname(id) ? path : [path + '.ts', path + '.js'].find(existsSync);
                }
            },
            load(id) {
                if (id === '\0fixture:$app/env') return 'export const browser = true;';
                if (id === '\0fixture:$app/env/public') return 'export const PUBLIC_POCKETBASE_URL = location.origin; export const PUBLIC_MAP_MAX_POLYLINES = undefined; export const PUBLIC_PRIVATE_INSTANCE = undefined;';
                if (id === '\0fixture:entry') return `
                    import { trails_show, trails_update } from '#lib/stores/trail_store.ts';
                    import { currentUser } from '#lib/stores/user_store.ts';
                    import { getPb } from '#lib/pocketbase.ts';
                    import { assetIdFromPhotoURL } from '#lib/util/asset_link_util.ts';
                    const pb = getPb();
                    pb.authStore.loadFromCookie(document.cookie);
                    window.moveSavedWaypointPhoto = async (id, removeWaypoint) => {
                        const user = pb.authStore.record;
                        if (!user?.username) throw new Error('Missing signed-in user');
                        const response = await fetch('/api/v1/profile/@' + encodeURIComponent(user.username));
                        if (!response.ok) throw new Error('Could not load current profile');
                        const profile = await response.json();
                        currentUser.set({ ...user, actor: profile.actor.id });
                        const before = await trails_show(id, undefined, undefined, true);
                        const pending = structuredClone(before);
                        const source = pending.expand.waypoints_via_trail[0];
                        const photo = source.photos[0];
                        pending.photos = [...(pending.photos ?? []), photo];
                        pending._assetLinks = [assetIdFromPhotoURL(photo)];
                        if (removeWaypoint) pending.expand.waypoints_via_trail = [];
                        else source.photos = [];
                        const saved = await trails_update(before, pending);
                        return { photos: saved.photos, waypoints: saved.expand.waypoints_via_trail, photo };
                    };
                `;
            },
        }],
        build: { write: false, minify: false, rolldownOptions: { input: 'fixture:entry', output: { format: 'iife' } } },
    });
    const built = Array.isArray(result) ? result[0] : result;
    if (!('output' in built)) throw new Error('Expected an in-memory browser build');
    const chunk = built.output.find(item => item.type === 'chunk');
    if (!chunk || chunk.type !== 'chunk') throw new Error('Missing production store browser bundle');
    return chunk.code;
}
