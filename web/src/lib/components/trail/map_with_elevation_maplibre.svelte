<script lang="ts">
    import { page } from "$app/state";
    import directionCaret from "$lib/assets/svgs/caret-right-solid.svg";
    import GPX from "$lib/models/gpx/gpx";
    import type { Trail } from "$lib/models/trail";
    import type { Waypoint } from "$lib/models/waypoint";
    import { theme } from "$lib/stores/theme_store";
    import { fetchGPX } from "$lib/stores/trail_store";
    import { findStartAndEndPoints } from "$lib/util/geojson_util";
    import {
        createMarkerFromWaypoint,
        createPopupFromTrail,
        FontawesomeMarker,
    } from "$lib/util/maplibre_util";
    import { decodePolyline } from "$lib/util/polyline_util";
    import type { ElevationProfileControl } from "$lib/vendor/maplibre-elevation-profile/elevationprofile-control";
    import { FullscreenControl } from "$lib/vendor/maplibre-fullscreen/fullscreen-control";
    import MaplibreGraticule from "$lib/vendor/maplibre-graticule/maplibre-graticule";
    import { CaretLayer } from "$lib/vendor/maplibre-layer-manager/caret-layer";
    import { ClusterLayer } from "$lib/vendor/maplibre-layer-manager/cluster-layer";
    import { baseMapStyles } from "$lib/vendor/maplibre-layer-manager/layers";
    import { LayerManager } from "$lib/vendor/maplibre-layer-manager/maplibre-layer-manager";
    import { PreviewLayer } from "$lib/vendor/maplibre-layer-manager/preview-layer";
    import { TerrainLayer } from "$lib/vendor/maplibre-layer-manager/terrain-layer";
    import { TrailLayer } from "$lib/vendor/maplibre-layer-manager/trail-layer";
    import { StyleSwitcherControl } from "$lib/vendor/maplibre-style-switcher/style-switcher-control";
    import type { Feature, FeatureCollection, GeoJSON } from "geojson";
    import * as M from "maplibre-gl";
    import "maplibre-gl/dist/maplibre-gl.css";
    import { onDestroy, onMount, untrack } from "svelte";

    interface Props {
        trails?: Trail[];
        gpx?: GPX;
        waypoints?: Waypoint[];
        markers?: M.Marker[];
        map?: M.Map | null;
        drawing?: boolean;
        showElevation?: boolean;
        showInfoPopup?: boolean;
        showGrid?: boolean;
        showStyleSwitcher?: boolean;
        showFullscreen?: boolean;
        showTerrain?: boolean;
        fitBounds?: "animate" | "instant" | "off";
        onmarkerdragend?:
            | ((marker: M.Marker, wpId?: string) => void)
            | undefined;
        elevationProfileContainer?: string | HTMLDivElement | undefined;
        mapOptions?: Partial<M.MapOptions> | undefined;
        activeTrail?: number | null;
        clusterTrails?: boolean;
        onsegmentdragend?: (data: {
            segment: number;
            event: M.MapMouseEvent;
        }) => void;
        onsegmentclick?: (data: {
            segment: number;
            event: M.MapMouseEvent;
        }) => void;
        onselect?: (trail: Trail) => void;
        onunselect?: (trail: Trail) => void;
        onfullscreen?: () => void;
        onmoveend?: (map: M.Map) => void;
        onzoom?: (map: M.Map) => void;
        onclick?: (event: M.MapMouseEvent & Object) => void;
        onUnclusteredClick?: (
            event: M.MapMouseEvent & Object,
            trail: Trail,
        ) => void;
        oninit?: (map: M.Map) => void;
    }

    let {
        trails = [],
        waypoints = [],
        markers = $bindable([]),
        map = $bindable(),
        drawing = false,
        showElevation = true,
        showInfoPopup = false,
        showGrid = false,
        showStyleSwitcher = true,
        showFullscreen = false,
        showTerrain = false,
        fitBounds = "instant",
        elevationProfileContainer = undefined,
        mapOptions = undefined,
        activeTrail = $bindable(0),
        clusterTrails = false,
        onmarkerdragend,
        onsegmentdragend,
        onsegmentclick,
        onselect,
        onunselect,
        onfullscreen,
        onmoveend,
        onzoom,
        onclick,
        onUnclusteredClick,
        oninit,
    }: Props = $props();

    let mapContainer: HTMLDivElement;
    let epc: ElevationProfileControl;
    let graticule: MaplibreGraticule;

    let layerManager: LayerManager;

    let elevationMarker: FontawesomeMarker;

    let draggingSegment: number | null = null;

    let hoveringTrail: boolean = false;

    let mapLoaded: boolean = false;

    const trailColors = [
        "#3549bb", // blue
        "#ff7f0e", // orange
        "#2ca02c", // green
        "#d62728", // red
        "#9467bd", // purple
        "#8c564b", // brown
        "#e377c2", // pink
        "#373642", // gray
        "#fae455", // yellow
        "#17becf", // teal
    ];

    let clusterPopup: M.Popup | null = null;

    let [data, clusterData, previewData] = $derived(getData(trails));
    $effect(() => {
        if (data && map && mapLoaded) {
            untrack(() => initMap(map?.loaded() ?? false));
        }
    });
    $effect(() => {
        adjustTrailFocus(activeTrail);
    });
    $effect(() => {
        toggleEpcTheme();
    });
    $effect(() => {
        if (drawing && map && layerManager) {
            untrack(() => startDrawing());
        } else if (map && layerManager) {
            untrack(() => stopDrawing());
        }
    });
    $effect(() => {
        if (showGrid) {
            if (!graticule) {
                graticule = new MaplibreGraticule({
                    minZoom: 0,
                    maxZoom: 20,
                    showLabels: true,
                    labelType: "hdms",
                    labelSize: 10,
                    labelColor: "#858585",
                    longitudePosition: "top",
                    latitudePosition: "right",
                    paint: {
                        "line-opacity": 0.8,
                        "line-color": "rgba(0,0,0,0.2)",
                    },
                });
            }
            map?.addControl(graticule);
        } else {
            if (graticule) {
                map?.removeControl(graticule);
            }
        }
    });
    $effect(() => {
        waypoints;
        untrack(() => {
            showWaypoints();
            refreshElevationProfile();
        });
    });

    function getData(
        trails: Trail[],
    ): [FeatureCollection[], FeatureCollection, FeatureCollection] {
        let clusterData: FeatureCollection = {
            type: "FeatureCollection",
            features: [],
        };
        let previewData: FeatureCollection = {
            type: "FeatureCollection",
            features: [],
        };
        let r: FeatureCollection[] = [];

        trails.forEach((t, i) => {
            if (t.expand?.gpx) {
                r.push(t.expand.gpx.toGeoJSON());
            } else if (t.expand?.gpx_data) {
                r.push(GPX.parse(t.expand.gpx_data).toGeoJSON());
            }
            if (clusterTrails) {
                if (t.lat !== null && t.lon !== null) {
                    clusterData.features.push({
                        id: t.id,
                        type: "Feature",
                        properties: {
                            trail: t.id,
                        },
                        geometry: {
                            type: "Point",
                            coordinates: [t.lon ?? 0, t.lat ?? 0],
                        },
                    } as Feature);
                }

                if (t.polyline) {
                    previewData.features.push({
                        id: t.id,
                        type: "Feature",
                        properties: {
                            trail: t.id,
                            color: trailColors[
                                hashStringToIndex(
                                    t.id ?? "",
                                    trailColors.length,
                                )
                            ],
                        },
                        geometry: {
                            type: "LineString",
                            coordinates: decodePolyline(t.polyline, 5),
                        },
                    });
                }
            }
        });

        return [r, clusterData, previewData];
    }

    function initMap(mapLoaded: boolean) {
        if (!map || !layerManager) {
            return;
        }

        refreshElevationProfile();
        if (showElevation && data.length && activeTrail !== null) {
            epc?.showProfile();
        } else {
            epc?.hideProfile();
        }

        trails.forEach((t, i) => {
            const layerId = t.id!;
            addTrailLayer(t, layerId, i, data[i]);
        });
        if (clusterTrails) {
            addClusterLayer(clusterData);
            addPreviewLayer(previewData);
        }

        Object.entries(layerManager.layers).forEach(([id, layer]) => {
            if (!(layer instanceof TrailLayer)) {
                return;
            }
            const isStillVisible = trails.some((t) => t.id === id);
            if (!isStillVisible) {
                removeCaretLayer();
                removeTrailLayer(id);
            }
        });

        if (
            !drawing &&
            fitBounds !== "off" &&
            data.some((d) => d.bbox !== undefined)
        ) {
            if (activeTrail !== null && trails[activeTrail] && mapLoaded) {
                focusTrail(trails[activeTrail]);
            } else {
                flyToBounds();
            }
        } else if (drawing && activeTrail !== null && mapLoaded) {
            addCaretLayer(data[activeTrail]);
        }
    }

    export function refreshElevationProfile() {
        if (activeTrail !== null && data[activeTrail]) {
            epc?.setData(data[activeTrail]!, waypoints);
        }
    }

    function getBounds() {
        let minX = Infinity,
            minY = Infinity,
            maxX = -Infinity,
            maxY = -Infinity;

        for (const [xMin, yMin, xMax, yMax] of data
            .filter((d) => d.bbox !== undefined)
            .map((d) => d.bbox!)) {
            minX = Math.min(minX, xMin);
            minY = Math.min(minY, yMin);
            maxX = Math.max(maxX, xMax);
            maxY = Math.max(maxY, yMax);
        }

        if (
            minX < Infinity &&
            minY < Infinity &&
            maxX > -Infinity &&
            maxY > -Infinity
        ) {
            return new M.LngLatBounds([minX, minY, maxX, maxY]);
        } else {
            return new M.LngLatBounds([0, 0, 0, 0]);
        }
    }

    function flyToBounds() {
        const bounds =
            activeTrail !== null && data[activeTrail]
                ? (data[activeTrail].bbox as M.LngLatBoundsLike)
                : getBounds();

        if (!bounds || !map) {
            return;
        }

        map!.fitBounds(bounds, {
            animate: fitBounds == "animate",
            padding: {
                top: 16,
                left: 16,
                right: 16,
                bottom:
                    16 +
                    (epc?.isProfileShown && !elevationProfileContainer
                        ? map!.getContainer().clientHeight * 0.3
                        : 0),
            },
        });
    }

    function removeTrailLayer(id: string) {
        layerManager.removeLayer(id);
    }

    function addTrailLayer(
        trail: Trail,
        id: string,
        index: number,
        geojson: GeoJSON.FeatureCollection | null | undefined,
    ) {
        if (!geojson || !map) {
            return;
        }
        const trailLayer = new TrailLayer(
            id,
            geojson,
            trailColors[
                clusterTrails
                    ? hashStringToIndex(id ?? "", trailColors.length)
                    : 0
            ],
            {
                onEnter: (e) =>
                    highlightTrail(id, trails[activeTrail ?? -1]?.id == id),

                onLeave: (e) => unHighlightTrail(id),
                onMouseUp: (e) => {
                    activeTrail = trails.findIndex((t) => t.id == trail.id);
                },
                onMouseMove: moveCrosshairToCursorPosition,
                onMouseDown: (e) => handleDragStart(e, id),
            },
        );

        layerManager.addLayer(id, trailLayer);

        if (!drawing && !clusterTrails) {
            addStartEndMarkers(trail, id, geojson);
        }
    }

    function addClusterLayer(geojson: FeatureCollection) {
        if (!geojson || !map || !map.style) {
            return;
        }
        layerManager.addLayer("clusters", new ClusterLayer(map, geojson));
    }

    function addPreviewLayer(geojson: FeatureCollection) {
        if (!geojson || !map || !map.style) {
            return;
        }
        layerManager.addLayer(
            "preview",
            new PreviewLayer(map, geojson, {
                preview: {
                    onEnter: (e) => {
                        const trail = trails.find(t => t.id === (e as any).features[0].properties.trail)
                        if(!trail) return;
                        highlightCluster(trail)
                    },
                    onLeave: (e) => {
                        unHighlightCluster()
                    }
                },
            }),
        );
    }

    function moveCrosshairToCursorPosition(e: M.MapMouseEvent) {
        epc?.moveCrosshair(e.lngLat.lat, e.lngLat.lng);
        moveElevationMarkerToCursorPosition(e);
    }

    function moveElevationMarkerToCursorPosition(e: M.MapMouseEvent) {
        elevationMarker.setLngLat(e.lngLat);
    }

    function handleDragStart(e: M.MapMouseEvent, id: string) {
        if (
            !drawing ||
            (e.originalEvent.target as HTMLElement | null)?.classList.contains(
                "route-anchor",
            )
        ) {
            return;
        }
        e.preventDefault();

        const features = map?.queryRenderedFeatures(e.point, {
            layers: [id],
        });
        const segmentId = features?.at(0)?.properties.segmentId;
        if (segmentId !== null) {
            draggingSegment = segmentId;
        }

        map?.on("mousemove", moveElevationMarkerToCursorPosition);
        map?.once("mouseup", (e2) => handleDragEnd(e2, e));
    }

    function handleDragEnd(end: M.MapMouseEvent, start: M.MapMouseEvent) {
        map?.off("mousemove", moveElevationMarkerToCursorPosition);
        epc?.hideCrosshair();
        const distanceDragged = Math.sqrt(
            Math.pow(end.originalEvent.x - start.originalEvent.x, 2) +
                Math.pow(end.originalEvent.y - start.originalEvent.y, 2),
        );
        if (distanceDragged < 0.5) {
            onsegmentclick?.({ segment: draggingSegment!, event: end });
        } else {
            onsegmentdragend?.({ segment: draggingSegment!, event: end });
        }
        draggingSegment = null;
    }

    function addCaretLayer(geojson: GeoJSON) {
        if (!map) {
            return;
        }
        if (map.getLayer("direction-carets")) {
            removeCaretLayer();
        }
        layerManager.addLayer(
            "direction-carets",
            new CaretLayer({ type: "geojson", data: geojson }),
        );
    }

    function removeCaretLayer() {
        layerManager?.removeLayer("direction-carets");
    }

    export function highlightTrail(
        id: string,
        showElevationMarker: boolean = false,
    ) {
        if (!id) {
            return;
        }
        if (showElevationMarker) {
            elevationMarker.setOpacity("1");
        }
        map?.setPaintProperty(id, "line-width", 7);
        if (map?.getLayer(id)) {
            hoveringTrail = true;
        }
        // map?.setPaintProperty(id, "line-color", "#2766e3");
    }

    export function unHighlightTrail(id: string | undefined) {
        if (!id || draggingSegment !== null) {
            return;
        }
        elevationMarker.setOpacity("0");
        epc?.hideCrosshair();
        hoveringTrail = false;
        if (map?.getLayer(id)) {
            map?.setPaintProperty(id, "line-width", 5);
        }
        // map?.setPaintProperty(id, "line-color", "#648ad5");
    }

    export async function highlightCluster(trail: Trail) {
        if (!map || !map.style) {
            return;
        }
        clusterPopup = createPopupFromTrail(trail);
        clusterPopup.setLngLat([trail.lon!, trail.lat!]).addTo(map);

        clusterPopup.on("close", () => {
            unHighlightCluster(false);
        });
    }

    export async function unHighlightCluster(closePopup: boolean = true) {
        if (!map || !map.style) {
            return;
        }
        layerManager.removeLayer("cluster-highlight");
        if (closePopup) {
            clusterPopup?.remove();
        }
    }

    function adjustTrailFocus(activeTrail: number | null) {
        if (activeTrail !== null && trails[activeTrail] !== undefined) {
            if (
                !drawing &&
                fitBounds !== "off" &&
                data.some((d) => d.bbox !== undefined)
            ) {
                untrack(() => focusTrail(trails[activeTrail]));
            }
        } else if (activeTrail === null && trails.length) {
            untrack(() => unFocusTrail());
        }
    }

    function focusTrail(trail: Trail) {
        activeTrail = trails.findIndex((t) => t.id == trail.id);
        if (activeTrail < 0) {
            activeTrail = null;
            return;
        }
        onselect?.(trail);

        try {
            refreshElevationProfile();
            if (showElevation) {
                epc?.showProfile();
            }
            showWaypoints();
            addCaretLayer(data[activeTrail]);
            flyToBounds();
        } catch (e) {
            console.warn(e);
        }
    }

    function unFocusTrail(trail?: Trail) {
        if (trail) {
            onunselect?.(trail);
            unHighlightTrail(trail.id!);
        }

        activeTrail = null;
        flyToBounds();

        if (showElevation) {
            epc?.hideProfile();
        }
        hideWaypoints();
        removeCaretLayer();
    }

    function startDrawing() {
        if (!map) {
            return;
        }
        hideWaypoints();
        activeTrail ??= 0;
        map.getCanvas().style.cursor = "crosshair";
        if (trails[activeTrail]) {
            removeStartEndMarkers(trails[activeTrail].id);
        }
    }

    function stopDrawing() {
        if (!map) {
            return;
        }
        showWaypoints();
        map.getCanvas().style.cursor = "inherit";

        if (activeTrail !== null && trails[activeTrail] && !clusterTrails) {
            addStartEndMarkers(
                trails[activeTrail],
                trails[activeTrail].id,
                data[activeTrail],
            );
        }
    }

    function addStartEndMarkers(
        trail: Trail,
        id: string | undefined,
        geojson: GeoJSON | null | undefined,
    ) {
        if (
            !map ||
            !trail ||
            !id ||
            !(layerManager.layers[id] instanceof TrailLayer)
        ) {
            return;
        }

        const startMarker = new FontawesomeMarker(
            { icon: "fa fa-bullseye" },
            {},
        );
        layerManager.layers[id].markers.start = startMarker;

        if (!geojson) {
            if (trail.lon && trail.lat) {
                startMarker.setLngLat([trail.lon, trail.lat]).addTo(map);
            }
            return;
        }

        const startEndPoint = findStartAndEndPoints(geojson);

        if (!startEndPoint.length) {
            return;
        }

        const endMarker = new FontawesomeMarker(
            { icon: "fa fa-flag-checkered" },
            {},
        );
        layerManager.layers[id].markers.end = endMarker;

        startMarker.setLngLat(startEndPoint[0] as M.LngLatLike);
        endMarker.setLngLat(
            startEndPoint[startEndPoint.length - 1] as M.LngLatLike,
        );

        if (showInfoPopup) {
            const popup = createPopupFromTrail(trail);
            startMarker.setPopup(popup);
            endMarker.setPopup(popup);
        }

        startMarker.addTo(map);
        if (!clusterTrails) {
            endMarker.addTo(map);
        }
    }

    function removeStartEndMarkers(id: string | undefined) {
        if (!id) {
            return;
        }
        layerManager.layers[id].markers?.start?.remove();
        layerManager.layers[id].markers?.end?.remove();
    }

    function showWaypoints() {
        if (!map || drawing) {
            return;
        }

        hideWaypoints();
        for (const waypoint of waypoints) {
            if (!markers.find((m) => m._element.id == waypoint.id)) {
                const marker = createMarkerFromWaypoint(
                    waypoint,
                    onmarkerdragend,
                );
                marker.addTo(map);
                markers.push(marker);
            }
        }
        markers = markers.filter((marker) => {
            if (!waypoints.find((w) => w.id == marker._element.id)) {
                marker.remove();
                return false;
            }
            return true;
        });
    }

    function hideWaypoints() {
        if (!map) {
            return;
        }
        for (const m of markers) {
            m.remove();
        }
        markers = [];
    }

    function toggleEpcTheme() {
        if ($theme == "dark") {
            epc?.toggleTheme({
                profileBackgroundColor: "#191b24",
                elevationGridColor: "#ddd2",
                labelColor: "#ddd8",
                crosshairColor: "#fff5",
                tooltipBackgroundColor: "#242734",
                tooltipTextColor: "#fff",
            });
        } else {
            epc?.toggleTheme({
                profileBackgroundColor: "#242734",
                elevationGridColor: "#0002",
                labelColor: "#0009",
                crosshairColor: "#0005",
                tooltipBackgroundColor: "#fff",
                tooltipTextColor: "#000",
            });
        }
    }

    onMount(async () => {
        const initialState = {
            lng: 0,
            lat: 0,
            zoom: 1,
        };
        const ElevationProfileControl = (
            await import(
                "$lib/vendor/maplibre-elevation-profile/elevationprofile-control"
            )
        ).ElevationProfileControl;

        if (!mapContainer) {
            return;
        }

        for (const tileset of page.data.settings?.tilesets ?? []) {
            baseMapStyles[tileset.name] = tileset.url;
        }

        const finalMapOptions: M.MapOptions = {
            ...{
                container: mapContainer,
                center: [initialState.lng, initialState.lat],
                zoom: initialState.zoom,
            },
            ...mapOptions,
        };
        map = new M.Map(finalMapOptions);

        layerManager = new LayerManager(map);

        elevationMarker = new FontawesomeMarker(
            {
                id: "elevation-marker",
                icon: "fa-regular fa-circle",
                fontSize: "xs",
                width: 4,
                backgroundColor: "bg-primary",
                fontColor: "white",
            },
            {},
        );
        elevationMarker.setLngLat([0, 0]).addTo(map);
        elevationMarker.setOpacity("0");

        let img = new Image(20, 20);
        img.onload = () => map!.addImage("direction-caret", img);
        img.src = directionCaret;

        const switcherControl = new StyleSwitcherControl({
            styles: baseMapStyles,
            onchange: (state) => {
                layerManager.update(JSON.parse(JSON.stringify(state)));
            },
            state: JSON.parse(JSON.stringify(layerManager.state)),
        });

        map.addControl(
            new M.NavigationControl({ visualizePitch: showTerrain }),
        );
        map.addControl(
            new M.ScaleControl({
                maxWidth: 120,
                unit: page.data.settings?.unit ?? "metric",
            }),
            "top-left",
        );

        map.addControl(
            new M.GeolocateControl({
                positionOptions: {
                    enableHighAccuracy: true,
                },
                fitBoundsOptions: {
                    animate: fitBounds == "animate",
                },
                trackUserLocation: true,
            }));

        if (showStyleSwitcher) {
            map.addControl(switcherControl);
        }

        if (showElevation) {
            epc = new ElevationProfileControl({
                visible: false,
                profileBackgroundColor:
                    $theme == "light" ? "#242734" : "#191b24",
                backgroundColor: "bg-menu-background/90",
                unit: page.data.settings?.unit ?? "metric",
                profileLineWidth: 3,
                displayDistanceGrid: true,
                tooltipDisplayDPlus: false,
                tooltipBackgroundColor: $theme == "light" ? "#fff" : "#242734",
                tooltipTextColor: $theme == "light" ? "#000" : "#fff",
                zoom: false,
                container: elevationProfileContainer,
                onEnter: () => {
                    elevationMarker.setOpacity("1");
                },
                onLeave: () => {
                    elevationMarker.setOpacity("0");
                },
                onMove: (data) => {
                    if (!hoveringTrail) {
                        elevationMarker.setLngLat(
                            data.position as M.LngLatLike,
                        );
                    }
                },
            });
            toggleEpcTheme();
            map.addControl(epc);
        }

        if (showFullscreen) {
            map.addControl(
                new FullscreenControl(() => {
                    onfullscreen?.();
                }),
                "bottom-right",
            );
        }

        if (showTerrain && page.data.settings?.terrain?.terrain) {
            map!.addControl(
                new M.TerrainControl({
                    source: "terrain",
                }),
            );
        }

        map.on("styledata", (e) => {
            if (showTerrain && page.data.settings?.terrain?.terrain) {
                layerManager.addLayer(
                    "terrain",
                    new TerrainLayer(
                        page.data.settings.terrain.terrain,
                        page.data.settings?.terrain?.hillshading,
                    ),
                );
            }
        });

        map.on("moveend", (e) => {
            onmoveend?.(e.target);
        });

        map.on("zoom", (e) => {
            onzoom?.(e.target);
        });

        map.on("click", (e) => {
            if (hoveringTrail && drawing) {
                return;
            }
            onclick?.(e);
        });

        map.on("load", () => {
            layerManager.init();
            initMap(true);
            oninit?.(map!);
            mapLoaded = true;
        });

        showWaypoints();
    });

    onDestroy(() => {
        map?.remove();
    });

    function handleKeydown(e: KeyboardEvent) {
        const target = e.target as HTMLElement;

        const isInputField =
            target.tagName === "INPUT" ||
            target.tagName === "TEXTAREA" ||
            target.isContentEditable;

        if (isInputField) {
            return;
        }

        if (e.key == "m") {
            if (trails.length === 1) {
                removeCaretLayer();
                removeTrailLayer(trails[0].id!);
            }
        }
    }

    function handleKeyup(e: KeyboardEvent) {
        const target = e.target as HTMLElement;

        const isInputField =
            target.tagName === "INPUT" ||
            target.tagName === "TEXTAREA" ||
            target.isContentEditable;

        if (isInputField) {
            return;
        }

        if (e.key == "m") {
            if (trails.length === 1) {
                addTrailLayer(trails[0], trails[0].id!, 0, data[0]);
                addCaretLayer(data[0]);
            }
        } else if (e.key == "p") {
            if (showElevation) {
                epc?.toggleProfile();
            }
        }
    }

    function hashStringToIndex(str: string, max: number) {
        let hash = 0;
        for (let i = 0; i < str.length; i++) {
            hash = (hash << 5) - hash + str.charCodeAt(i);
            hash |= 0;
        }
        return Math.abs(hash) % max;
    }
</script>

<svelte:window on:keydown={handleKeydown} on:keyup={handleKeyup} />
<div id="map" bind:this={mapContainer}></div>

<style lang="postcss">
    @reference "tailwindcss";
    @reference "../../../css/app.css";

    #map {
        width: 100%;
        height: 100%;
    }

    :global(.maplibregl-popup-content) {
        @apply bg-background rounded-md shadow-xl p-0 overflow-hidden pr-5;
    }

    :global(.maplibregl-popup-close-button) {
        top: 4px;
        right: 4px;
        line-height: 0;
        padding-bottom: 2.5px;
        @apply bg-menu-item-background-focus w-3 aspect-square rounded-full;
    }
</style>
