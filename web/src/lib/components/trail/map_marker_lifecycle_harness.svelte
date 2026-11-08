<script module lang="ts">
    import type { Trail } from "#lib/models/trail.ts";
    export interface MapMarkerLifecycleApi {
        setDrawing: (value: boolean) => void;
        setTrails: (value: Trail[]) => void;
    }
</script>

<script lang="ts">
    import { untrack } from "svelte";
    import MapWithElevationMaplibre from "./map_with_elevation_maplibre.svelte";
    const props: { api: MapMarkerLifecycleApi; initialTrails: Trail[] } = $props();
    const fixture = untrack(() => props);
    let trails = $state(fixture.initialTrails);
    let drawing = $state(false);
    fixture.api.setDrawing = (value) => { drawing = value; };
    fixture.api.setTrails = (value) => { trails = value; };
</script>

<MapWithElevationMaplibre {trails} {drawing} showElevation={false} showStyleSwitcher={false} activeTrail={0} fitBounds="off" />
