<script lang="ts">
    import * as M from "maplibre-gl";

    import GPX from "$lib/models/gpx/gpx";
    import { fromFile } from "$lib/util/gpx_util";
    import { onMount } from "svelte";
    interface Props {
        trailFile: File | undefined | null;
        trailData: string | undefined;
        hasTrail?: boolean;
        label?: string;
        onchange: (data: string | null) => void;
    }

    let {
        trailFile = $bindable(),
        trailData = $bindable(),
        hasTrail = false,
        label = "",
        onchange
    }: Props = $props();

    let map: M.Map;
    let layer: M.LineLayerSpecification | undefined;
    let source: M.GeoJSONSource | undefined;

    onMount(async () => {
        if (!map) {
            await initMap();
        }
    });

    async function initMap() {
        map = new M.Map({
            container: "trail-picker-map",
            attributionControl: false,
            dragPan: false,
            scrollZoom: false,
            preserveDrawingBuffer: true,
        });
    }

    const hasTrailState = $derived(
        Boolean(trailFile || trailData || hasTrail),
    );

    function openTrailBrowser() {
        if (hasTrailState) {
            trailFile = null;
            trailData = undefined;

            onchange?.(null);
        } else {
            document.getElementById("trail-input")!.click();
        }
    }

    async function handleTrailSelection(files?: FileList | null) {
        if (!files) {
            files = (document.getElementById("trail-input") as HTMLInputElement)
                .files;
        }

        if (!files) {
            return;
        }

        trailFile = files.item(0);
        if (!trailFile) {
            return;
        }
        const parseResult = await fromFile(trailFile);

        trailData = parseResult.gpxData;

        onchange?.(trailData);
    }

    async function showTrailOnMap() {
        if (!trailData || !map) {
            return;
        }
        const sourceId = "trail-picker-geojson-source";
        const layerId = "trail-picker-geojson-layer";

        const gpx = GPX.parse(trailData)
        if(gpx instanceof Error) {
            throw gpx;
        }

        const geojson = gpx.toGeoJSON()

        if (layer) {
            map.removeLayer(layerId);
            map.removeSource(sourceId);
        }

        map.addSource(sourceId, {
            type: "geojson",
            data: geojson,
        });
        map.addLayer({
            id: layerId,
            type: "line",
            source: sourceId,
            paint: {
                "line-color": "#3388ff",
                "line-width": 3,
            },
        });
        layer = map.getLayer(layerId) as M.LineLayerSpecification;
        source = map.getSource(sourceId) as M.GeoJSONSource;
        map.resize();
        map.fitBounds(geojson.bbox as M.LngLatBoundsLike, {
            animate: false,
            padding: 8,
        });
    }

    function removeTrailFromMap() {
        if (layer) {
            map?.removeLayer(layer.id);
            layer = undefined;
        }
        if (source) {
            map?.removeSource(source.id);
            source = undefined;
        }
    }
    $effect(() => {
        if (trailData !== undefined) {
            showTrailOnMap();
        } else {
            removeTrailFromMap();
        }
    });
</script>

<div>
    {#if label.length}
        <p class="text-sm font-medium pb-1">
            {label}
        </p>
    {/if}
    <div class="flex gap-x-4" role="dialog">
        <input
            type="file"
            id="trail-input"
            accept=".gpx,.GPX,.tcx,.TCX,.kml,.KML,.fit,.FIT"
            multiple={false}
            style="display: none;"
            onchange={() => handleTrailSelection()}
        />
        <button
            type="button"
            onclick={openTrailBrowser}
            class="h-28 aspect-square rounded-xl !bg-background border border-input-border-focus hover:!bg-secondary-hover group"
            id="trail-picker-map"
        >
            {#if !hasTrailState}
                <i class="fa fa-plus text-lg"></i>
            {:else}
                <i
                    class="fa fa-trash text-red-500 text-lg hidden group-hover:block relative"
                    style="z-index: 1000"
                ></i>
            {/if}
        </button>
    </div>
</div>
