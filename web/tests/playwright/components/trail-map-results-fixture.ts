import { fileURLToPath } from 'node:url';

const componentPath = (name: string) => fileURLToPath(new URL(`../../../src/lib/components/${name}.svelte`, import.meta.url));

// Keep the real map-result component. Only the rendering children and the map
// engine are replaced; camera changes still drive its real viewport callbacks.
const selectableItem = (testId: string) => `
    <script>let { trail, hovered } = $props();</script>
    <div data-testid="${testId}" data-hovered={hovered}><h4>{trail.name}</h4><span>Trail details</span></div>
`;
export const mapResultStubs = {
    [componentPath('trail/trail_card')]: selectableItem('map-card'),
    [componentPath('trail/trail_list_item')]: selectableItem('map-list-item'),
    [componentPath('trail/trail_table')]: `
        <script>let { trails, selection, onTrailSelect } = $props();</script>
        <div data-testid="map-table">
                {#each trails ?? [] as trail}
                    <input type="checkbox" aria-label={'Select ' + trail.name} checked={selection?.has(trail) ?? false} onclick={() => onTrailSelect(trail)} />
                {/each}
        </div>
    `,
    [componentPath('trail/trail_dropdown')]: `
        <script>
            import Dropdown from ${JSON.stringify(componentPath('base/dropdown'))};
            let { trails, toggle, onUpdate, onDelete, onMerge, showMergeDialog } = $props();
            let dialog;
            $effect(() => {
                window.mapResultsFixture.selected = () => [...trails].map(trail => trail.id);
                window.mapResultsFixture.update = onUpdate;
                window.mapResultsFixture.delete = onDelete;
                window.mapResultsFixture.merge = onMerge;
                window.mapResultsFixture.showMergeDialog = showMergeDialog;
                window.mapResultsFixture.openActionDialog = () => dialog.showModal();
            });
        </script>
        <Dropdown items={[{ text: 'Export', value: 'export' }]}>
            {#snippet children({ toggleMenu })}{@render toggle({ toggleMenu, loading: false })}{/snippet}
        </Dropdown>
        <dialog bind:this={dialog}><input aria-label="Action value" value="Keep this value" /></dialog>
    `,
    [componentPath('base/skeleton_list_item')]: '<div>Loading list item</div>',
    [componentPath('base/skeleton_card')]: '<div>Loading</div>',
    [componentPath('empty_states/empty_state_search')]: '<div>No results</div>',
    [componentPath('trail/map_with_elevation_maplibre')]: `
        <script>
            import { onMount, tick } from 'svelte';
            let { map = $bindable(), oninit, onmoveend } = $props();
            let container;
            class FakeMap {
                style = {};
                camera = { center: [0, 0], zoom: 3, bearing: 0, pitch: 0 };
                jumpTo(change) {
                    this.camera = { ...this.camera, ...change };
                    onmoveend();
                }
                stop() { window.mapResultsFixture.moving = false; onmoveend(); }
                resize() { if (window.mapResultsFixture.emitResizeMoves) onmoveend(); }
                canvas = document.createElement("canvas");
                getCanvas() { return this.canvas; }
                _resizeInternal() { this.canvas.width = container.clientWidth; this.canvas.height = container.clientHeight; }
                _update() {}
                fire(type, data) { if (type === "moveend") onmoveend(data); return this; }
                _render() { this.redraw(); }
                redraw() { window.mapResultsFixture.redraws = (window.mapResultsFixture.redraws ?? 0) + 1; }
                addControl(control) { container.querySelector('.maplibregl-ctrl-top-right').append(control.onAdd(this)); }
                getCenter() { return { toArray: () => [...this.camera.center] }; }
                getZoom() { return this.camera.zoom; }
                getBearing() { return this.camera.bearing; }
                getPitch() { return this.camera.pitch; }
                isMoving() { return Boolean(window.mapResultsFixture.moving); }
                getBounds() {
                    const [lng, lat] = this.camera.center;
                    if (window.mapInitial?.cameraGeometry) {
                        const angle = this.camera.bearing * Math.PI / 180;
                        const scale = Math.pow(2, 3 - this.camera.zoom) / 5000;
                        const width = container.clientWidth, height = container.clientHeight / Math.cos(this.camera.pitch * Math.PI / 180);
                        const dx = (Math.abs(Math.cos(angle)) * width + Math.abs(Math.sin(angle)) * height) * scale / 2;
                        const dy = (Math.abs(Math.sin(angle)) * width + Math.abs(Math.cos(angle)) * height) * scale / 2;
                        return { getNorth: () => lat + dy, getSouth: () => lat - dy, getEast: () => lng + dx, getWest: () => lng - dx };
                    }
                    return { getNorth: () => lat + .1, getSouth: () => lat - .1,
                        getEast: () => lng + (window.mapResultsFixture.sizedBounds ? container.clientWidth / 5000 : .1), getWest: () => lng - (window.mapResultsFixture.sizedBounds ? container.clientWidth / 5000 : .1) };
                }
                fitBounds(bounds, options = {}) {
                    const [[west, south], [east, north]] = Array.isArray(bounds) ? bounds : bounds.toArray();
                    window.mapResultsFixture.fitCount = (window.mapResultsFixture.fitCount ?? 0) + 1;
                    let zoom = 12;
                    if (window.mapInitial?.cameraGeometry) {
                        const angle = (options.bearing ?? 0) * Math.PI / 180;
                        const width = Math.abs(Math.cos(angle)) * (east - west) + Math.abs(Math.sin(angle)) * (north - south);
                        const height = (Math.abs(Math.sin(angle)) * (east - west) + Math.abs(Math.cos(angle)) * (north - south)) / Math.cos((options.pitch ?? this.camera.pitch) * Math.PI / 180);
                        zoom = 3 + Math.log2(Math.min(container.clientWidth / (5000 * width), container.clientHeight / (5000 * height)));
                    }
                    window.mapResultsFixture.fittedSize = { width: container.clientWidth, height: container.clientHeight };
                    this.jumpTo({ center: [(west + east) / 2, (south + north) / 2], zoom, bearing: options.bearing ?? 0, pitch: options.pitch ?? this.camera.pitch });
                }
            }
            onMount(() => {
                map = new FakeMap();
                window.mapResultsFixture.camera = () => JSON.parse(JSON.stringify(map.camera));
                window.mapResultsFixture.renderedBounds = () => {
                    const bounds = map.getBounds();
                    return { northEast: { lat: bounds.getNorth(), lng: bounds.getEast() },
                        southWest: { lat: bounds.getSouth(), lng: bounds.getWest() } };
                };
                window.mapResultsFixture.setCamera = change => map.jumpTo(change);
                window.mapResultsFixture.pan = () => map.jumpTo({ center: [8, 47], zoom: 10, bearing: 12, pitch: 30 });
                // The real map also reports initialization asynchronously.
                window.mapResultsFixture.initialize = oninit;
                if (!window.mapInitial?.delayInitialization) void tick().then(() => oninit());
            });
            export function highlightCluster() {}
            export function unHighlightCluster() {}
        </script>
        <div data-testid="fake-map" bind:this={container} style="height: 100%; min-height: 1px; position: relative">
            <div class="maplibregl-ctrl-top-right" style="position: absolute; top: 10px; right: 10px"></div>
            <div class="maplibregl-ctrl-bottom-right" style="position: absolute; bottom: 0; right: 0"></div>
        </div>
    `,
};
