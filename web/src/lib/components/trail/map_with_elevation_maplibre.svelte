<script lang="ts">
    import { page } from "$app/state";
    import directionCaret from "$lib/assets/svgs/caret-right-solid.svg";
    import GPX from "$lib/models/gpx/gpx";
    import type { Trail } from "$lib/models/trail";
    import type { Waypoint } from "$lib/models/waypoint";
    import { theme } from "$lib/stores/theme_store";
    import { findStartAndEndPoints } from "$lib/util/geojson_util";
    import {
        createMarkerFromWaypoint,
        createPopupFromTrail,
        FontawesomeMarker,
    } from "$lib/util/maplibre_util";
    import { decodePolyline } from "$lib/util/polyline_util";
    import { TRAIL_COLORS, trailColor } from "$lib/util/trail_color_util";
    import type { ElevationProfileControl } from "$lib/vendor/maplibre-elevation-profile/elevationprofile-control";
    import { FullscreenControl } from "$lib/vendor/maplibre-fullscreen/fullscreen-control";
    import MaplibreGraticule from "$lib/vendor/maplibre-graticule/maplibre-graticule";
    import { CaretLayer } from "$lib/vendor/maplibre-layer-manager/caret-layer";
    import { ClusterLayer } from "$lib/vendor/maplibre-layer-manager/cluster-layer";
    import { baseMapStyles } from "$lib/vendor/maplibre-layer-manager/layers";
    import { LayerManager } from "$lib/vendor/maplibre-layer-manager/maplibre-layer-manager";
    import type { OverpassPopupActionFactory } from "$lib/vendor/maplibre-layer-manager/overpass-layer";
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
        previewTrails?: Trail[];
        focusedTrailId?: string;
        elevationPreviewGpx?: GPX;
        serverClusters?: GeoJSON.FeatureCollection;
        gpx?: GPX;
        waypoints?: Waypoint[];
        markers?: M.Marker[];
        map?: M.Map | null;
        drawing?: boolean;
        displayWaypoints?: boolean;
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
        oncontextmenu?: (event: M.MapMouseEvent & Object) => void;
        onUnclusteredClick?: (
            event: M.MapMouseEvent & Object,
            trail: Trail,
        ) => void;
        oninit?: (map: M.Map) => void;
        autoGeolocateOnDrawing?: boolean;
        buildPoiAnchorAction?: OverpassPopupActionFactory;
    }

    let {
        trails = [],
        previewTrails = [],
        focusedTrailId = undefined,
        elevationPreviewGpx = undefined,
        serverClusters = undefined,
        waypoints = [],
        markers = $bindable([]),
        map = $bindable(),
        drawing = false,
        displayWaypoints = true,
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
        oncontextmenu,
        onUnclusteredClick,
        oninit,
        autoGeolocateOnDrawing = false,
        buildPoiAnchorAction = undefined,
    }: Props = $props();

    let mapContainer: HTMLDivElement;
    let epc: ElevationProfileControl;
    let graticule: MaplibreGraticule;

    let layerManager: LayerManager;

    let elevationMarker: FontawesomeMarker;

    let draggingSegment: number | null = null;

    let hoveringTrail: boolean = false;

    let mapLoaded: boolean = $state(false);
    let terrainEnabled: boolean | null = null;
    let elevationProfileVisibilityPreference: boolean | null = null;

    let clusterPopup: M.Popup | null = null;

    let mapData = $derived(getData(trails, serverClusters));
    let previewGpxDataMap = $derived(getGpxData(previewTrails));
    let gpxDataMap = $derived({ ...mapData[0], ...previewGpxDataMap });
    let clusterData = $derived(mapData[1]);
    let previewData = $derived(mapData[2]);

    $effect(() => {
        // Track dependencies for Svelte 5
        mapData;
        previewGpxDataMap;
        previewTrails;

        if (map && mapLoaded) {
            untrack(() => initMap(map?.loaded() ?? false));
        }
    });
    $effect(() => {
        focusedTrailId;
        trails;
        previewTrails;
        if (map && mapLoaded) {
            untrack(() => adjustTrailLineFocus());
        }
    });
    $effect(() => {
        adjustTrailFocus(activeTrail);
    });
    $effect(() => {
        elevationPreviewGpx;
        if (map && mapLoaded) {
            untrack(() => refreshElevationProfile());
        }
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
        displayWaypoints;
        untrack(() => {
            syncWaypointMarkers();
            refreshElevationProfile();
        });
    });

    function getData(
        trails: Trail[],
        serverClusters?: GeoJSON.FeatureCollection
    ): [
        Record<string, FeatureCollection>,
        FeatureCollection,
        FeatureCollection,
    ] {
        let clusterData: FeatureCollection = serverClusters ?? {
            type: "FeatureCollection",
            features: [],
        };
        let previewData: FeatureCollection = {
            type: "FeatureCollection",
            features: [],
        };
        const gpxDataMap = getGpxData(trails);

        trails.forEach((t) => {
            if (clusterTrails) {
                if (!serverClusters && t.lat !== undefined && t.lon !== undefined) {
                    clusterData.features.push({
                        id: t.id,
                        type: "Feature",
                        properties: {
                            trail: t.id,
                            bounding_box_diagonal: t.bounding_box_diagonal,
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
                            bounding_box_diagonal: t.bounding_box_diagonal,
                            color: TRAIL_COLORS[
                                hashStringToIndex(
                                    t.id ?? "",
                                    TRAIL_COLORS.length,
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

        return [gpxDataMap, clusterData, previewData];
    }

    function getGpxData(trails: Trail[]) {
        const data: Record<string, FeatureCollection> = {};
        for (const trail of trails) {
            if (!trail.id) continue;
            let featureCollection: FeatureCollection | null = null;
            if (trail.expand?.gpx) {
                featureCollection = trail.expand.gpx.toGeoJSON();
            } else if (trail.expand?.gpx_data) {
                featureCollection = GPX.parse(trail.expand.gpx_data).toGeoJSON();
            }
            if (!featureCollection) continue;
            featureCollection.features.forEach((feature) => {
                if (feature.properties) {
                    feature.properties.bounding_box_diagonal =
                        trail.bounding_box_diagonal;
                }
            });
            data[trail.id] = featureCollection;
        }
        return data;
    }

    function initMap(mapLoaded: boolean) {
        if (!map || !layerManager) {
            return;
        }

        refreshElevationProfile();
        syncElevationProfileVisibility();

        previewTrails.forEach((t, index) => {
            const layerId = t.id!;
            addTrailLayer(t, layerId, index + 1, gpxDataMap[layerId], false);
        });

        trails.forEach((t) => {
            const layerId = t.id!;
            addTrailLayer(t, layerId, 0, gpxDataMap[layerId]);
        });

        if (previewTrails.length) {
            trails.forEach((trail) => {
                if (trail.id && map?.getLayer(trail.id)) {
                    map.moveLayer(trail.id);
                }
            });
        }

        Object.entries(layerManager.layers).forEach(([id, layer]) => {
            if (!(layer instanceof TrailLayer)) {
                return;
            }
            const isStillVisible = [...trails, ...previewTrails].some(
                (t) => t.id === id,
            );
            if (!isStillVisible) {
                removeCaretLayer();
                removeTrailLayer(id);
            }
        });

        if (clusterTrails) {
            addPreviewLayer(previewData);
            addClusterLayer(clusterData);
        }

        if (!drawing && fitBounds !== "off") {
            const currentBboxes = Object.values(gpxDataMap)
                .map((d) => d.bbox)
                .filter((b) => b !== undefined);

            if (
                activeTrail !== null &&
                trails[activeTrail] &&
                mapLoaded &&
                gpxDataMap[trails[activeTrail].id!]
            ) {
                focusTrail(trails[activeTrail]);
            } else if (currentBboxes.length > 0) {
                flyToBounds();
            }
        } else if (drawing && activeTrail !== null && mapLoaded) {
            const activeId = trails[activeTrail]?.id;
            if (activeId && gpxDataMap[activeId]) {
                addCaretLayer(gpxDataMap[activeId]);
            }
        }
    }

    function syncHillshadingVisibility() {
        if (!map?.getLayer("hillshading")) {
            terrainEnabled = null;
            return;
        }

        const isTerrainEnabled = Boolean(map.getTerrain());
        if (isTerrainEnabled === terrainEnabled) {
            return;
        }

        map.setLayoutProperty(
            "hillshading",
            "visibility",
            isTerrainEnabled ? "visible" : "none",
        );
        terrainEnabled = isTerrainEnabled;
    }

    export function refreshElevationProfile() {
        if (elevationPreviewGpx) {
            epc?.setData(elevationPreviewGpx.toGeoJSON(), waypoints);
            return;
        }
        const activeId = activeTrail !== null ? trails[activeTrail]?.id : null;
        if (activeId && gpxDataMap[activeId]) {
            epc?.setData(gpxDataMap[activeId]!, waypoints);
        }
    }

    function syncElevationProfileVisibility() {
        if (
            showElevation &&
            (elevationPreviewGpx || Object.keys(gpxDataMap).length) &&
            (elevationPreviewGpx || activeTrail !== null) &&
            elevationProfileVisibilityPreference !== false
        ) {
            epc?.showProfile();
        } else {
            epc?.hideProfile();
        }
    }

    function getBounds() {
        let minX = Infinity,
            minY = Infinity,
            maxX = -Infinity,
            maxY = -Infinity;

        for (const [xMin, yMin, xMax, yMax] of Object.values(gpxDataMap)
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

    export function fitToBounds(bounds?: M.LngLatBoundsLike) {
        const activeId = activeTrail !== null ? trails[activeTrail]?.id : null;
        const boundsToFit =
            bounds ??
            (activeId && gpxDataMap[activeId]
                ? (gpxDataMap[activeId].bbox as M.LngLatBoundsLike)
                : getBounds());

        if (!boundsToFit || !map) {
            return;
        }

        map!.fitBounds(boundsToFit, {
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

    export function fitToAllBounds() {
        fitToBounds(getBounds());
    }

    function flyToBounds() {
        fitToBounds();
    }

    function removeTrailLayer(id: string) {
        layerManager.removeLayer(id);
    }

    function addTrailLayer(
        trail: Trail,
        id: string,
        index: number,
        geojson: GeoJSON.FeatureCollection | null | undefined,
        interactive = true,
    ) {
        if (!geojson || !map) {
            return;
        }
        const listeners = interactive
            ? {
                  onEnter: (e: M.MapMouseEvent) =>
                      highlightTrail(
                          id,
                          trails[activeTrail ?? -1]?.id == id,
                      ),
                  onLeave: (e: M.MapMouseEvent) => unHighlightTrail(id),
                  onMouseUp: (e: M.MapMouseEvent) => {
                      activeTrail = trails.findIndex(
                          (t) => t.id == trail.id,
                      );
                  },
                  onMouseMove: moveCrosshairToCursorPosition,
                  onMouseDown: (e: M.MapMouseEvent) => handleDragStart(e, id),
              }
            : undefined;
        const trailLayer = new TrailLayer(
            id,
            geojson,
            trailColor(
                clusterTrails
                    ? hashStringToIndex(id ?? "", TRAIL_COLORS.length)
                    : index,
            ),
            {
                listeners,
                lineWidth: interactive ? 5 : 4,
                lineOpacity: interactive ? 1 : 0.8,
            },
        );

        layerManager.addLayer(id, trailLayer);

        if (interactive && !drawing && !clusterTrails) {
            addStartEndMarkers(trail, id, geojson);
        }
    }

    function adjustTrailLineFocus() {
        if (!map) return;
        const allTrails = [...trails, ...previewTrails];
        const activeFocus = focusedTrailId &&
            allTrails.some((trail) => trail.id === focusedTrailId)
                ? focusedTrailId
                : undefined;

        for (const trail of allTrails) {
            if (!trail.id || !map.getLayer(trail.id)) continue;
            const isPreview = previewTrails.some(
                (preview) => preview.id === trail.id,
            );
            const isFocused = trail.id === activeFocus;
            map.setPaintProperty(
                trail.id,
                "line-width",
                activeFocus ? (isFocused ? 7 : 3) : isPreview ? 4 : 5,
            );
            map.setPaintProperty(
                trail.id,
                "line-opacity",
                activeFocus ? (isFocused ? 1 : 0.5) : isPreview ? 0.8 : 1,
            );
        }

        if (activeFocus && map.getLayer(activeFocus)) {
            map.moveLayer(activeFocus);
        } else if (previewTrails.length) {
            for (const trail of trails) {
                if (trail.id && map.getLayer(trail.id)) {
                    map.moveLayer(trail.id);
                }
            }
        }
    }

    function addClusterLayer(geojson: FeatureCollection) {
        if (!geojson || !map || !map.style) {
            return;
        }
        layerManager.addLayer(
            "clusters",
            new ClusterLayer(map, geojson, {
                "unclustered-point": {
                    onEnter: (e) => {
                        if (map) map.getCanvas().style.cursor = "pointer";
                        const id = (e as any).features[0].properties.id;
                        const trail = trails.find((t) => t.id === id);
                        if (!hasTrailDetails(trail)) return;
                        highlightCluster(trail, e.lngLat);
                    },
                },
            }),
        );
    }

    function addPreviewLayer(geojson: FeatureCollection) {
        if (!geojson || !map || !map.style) {
            return;
        }
        layerManager.addLayer(
            "preview",
            new PreviewLayer(map, geojson, {
                showStartMarker: page.data.settings?.behavior?.showTrailStartMarker ?? false,
                listeners: {
                    preview: {
                        onEnter: (e) => {
                            if (map) map.getCanvas().style.cursor = "pointer";
                            const trail = trails.find(
                                (t) =>
                                    t.id ===
                                    (e as any).features[0].properties.trail,
                            );
                            if (!hasTrailDetails(trail)) return;
                            highlightCluster(trail, e.lngLat);
                        },
                    },
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

    function hasTrailDetails(trail: Trail | undefined): trail is Trail {
        return Boolean(trail?.name?.trim());
    }

    export async function highlightCluster(
        trail: Trail,
        lnglat?: M.LngLatLike,
    ) {
        if (!map || !map.style || !hasTrailDetails(trail)) {
            return;
        }
        clusterPopup?.remove();
        clusterPopup = createPopupFromTrail(trail);
        clusterPopup.setLngLat(lnglat ?? [trail.lon!, trail.lat!]).addTo(map);
        clusterPopup.on("close", () => {
            unHighlightCluster(false);
        });
        map.on("mousemove", unHighlightClusterDistanceNotifier);
    }

    function unHighlightClusterDistanceNotifier(e: M.MapMouseEvent) {
        if (!clusterPopup || !map) {
            return;
        }
        if (
            map.project(clusterPopup.getLngLat()).dist(map.project(e.lngLat)) >
            60
        ) {
            clusterPopup.remove();
            map.off("mousemove", unHighlightClusterDistanceNotifier);
        }
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
                Object.values(gpxDataMap).some((d) => d.bbox !== undefined)
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
            syncElevationProfileVisibility();
            syncWaypointMarkers();
            if (trail.id && gpxDataMap[trail.id]) {
                addCaretLayer(gpxDataMap[trail.id]);
            }
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
        activeTrail ??= 0;
        map.getCanvas().style.cursor = "crosshair";
        if (trails[activeTrail]) {
            removeStartEndMarkers(trails[activeTrail].id);
        }

        if (autoGeolocateOnDrawing) {
            geolocate();
        }
    }

    function stopDrawing() {
        if (!map) {
            return;
        }
        syncWaypointMarkers();
        map.getCanvas().style.cursor = "inherit";

        if (activeTrail !== null && trails[activeTrail] && !clusterTrails) {
            const activeId = trails[activeTrail].id;
            addStartEndMarkers(
                trails[activeTrail],
                activeId,
                activeId ? gpxDataMap[activeId] : null,
            );
        }
    }

    function addStartEndMarkers(
        trail: Trail,
        id: string | undefined,
        geojson: GeoJSON | null | undefined,
    ) {
        const layer = trailLayer(id);
        if (!map || !trail || !layer) {
            return;
        }

        removeStartEndMarkers(id);

        const startMarker = new FontawesomeMarker(
            { icon: "fa fa-bullseye" },
            {},
        );
        layer.markers.start = startMarker;

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
        layer.markers.end = endMarker;

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
        const layer = trailLayer(id);
        if (!layer) {
            return;
        }
        layer.markers.start?.remove();
        layer.markers.end?.remove();
        delete layer.markers.start;
        delete layer.markers.end;
    }

    function trailLayer(id: string | undefined): TrailLayer | undefined {
        if (!id || !layerManager) {
            return undefined;
        }
        const layer = layerManager.layers[id];
        return layer instanceof TrailLayer ? layer : undefined;
    }

    function showWaypoints() {
        if (!map) {
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

    function syncWaypointMarkers() {
        if (displayWaypoints) {
            showWaypoints();
        } else {
            hideWaypoints();
        }
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

    let geolocateControl: M.GeolocateControl;

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

        layerManager = new LayerManager(map, { overpassActionFactory: buildPoiAnchorAction });

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

        geolocateControl = new M.GeolocateControl({
            positionOptions: {
                enableHighAccuracy: true,
            },
            fitBoundsOptions: {
                animate: fitBounds == "animate",
            },
            trackUserLocation: true,
        });
        map.addControl(geolocateControl);

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
                onToggle: (visible) => {
                    elevationProfileVisibilityPreference = visible;
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
                syncHillshadingVisibility();
            }
        });

        map.on("render", () => {
            syncHillshadingVisibility();
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

        map.on("contextmenu", (e) => {
            oncontextmenu?.(e);
        });

        map.on("load", () => {
            layerManager.init();
            initMap(true);
            oninit?.(map!);
            mapLoaded = true;
        });

        syncWaypointMarkers();
    });

    function geolocate() {
        if (!page.data.settings?.behavior) return;

        if (page.data.settings.behavior.allowAutoGeolocate === true) {
            if (geolocateControl._watchState === "OFF") {
                geolocateControl.options.trackUserLocation = true;
                geolocateControl.trigger();
            }
        }
    }

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
                const trailId = trails[0].id!;
                addTrailLayer(trails[0], trailId, 0, gpxDataMap[trailId]);
                addCaretLayer(gpxDataMap[trailId]);
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

    :global(
            .maplibregl-user-location-accuracy-circle,
            .maplibregl-user-location-dot
        ) {
        pointer-events: none;
    }
</style>
