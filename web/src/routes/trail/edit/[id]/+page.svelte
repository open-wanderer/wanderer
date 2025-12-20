<script lang="ts">
    import { env } from "$env/dynamic/public";
    import Button from "$lib/components/base/button.svelte";
    import Datepicker from "$lib/components/base/datepicker.svelte";
    import Select from "$lib/components/base/select.svelte";
    import TextField from "$lib/components/base/text_field.svelte";
    import Toggle from "$lib/components/base/toggle.svelte";
    import ListSelectModal from "$lib/components/list/list_select_modal.svelte";
    import SummitLogCard from "$lib/components/summit_log/summit_log_card.svelte";
    import SummitLogModal from "$lib/components/summit_log/summit_log_modal.svelte";
    import MapWithElevationMaplibre from "$lib/components/trail/map_with_elevation_maplibre.svelte";
    import PhotoPicker from "$lib/components/trail/photo_picker.svelte";
    import WaypointCard from "$lib/components/waypoint/waypoint_card.svelte";
    import WaypointModal from "$lib/components/waypoint/waypoint_modal.svelte";
    import { SummitLogCreateSchema } from "$lib/models/api/summit_log_schema.js";
    import { TrailCreateSchema } from "$lib/models/api/trail_schema.js";
    import { WaypointCreateSchema } from "$lib/models/api/waypoint_schema.js";
    import GPX from "$lib/models/gpx/gpx";
    import GPXWaypoint from "$lib/models/gpx/waypoint";
    import type { List } from "$lib/models/list";
    import { SummitLog } from "$lib/models/summit_log";
    import { Trail } from "$lib/models/trail";
    import type { RoutingOptions, ValhallaAnchor } from "$lib/models/valhalla";
    import { Waypoint } from "$lib/models/waypoint";
    import { categories } from "$lib/stores/category_store";
    import {
        lists_add_trail,
        lists_remove_trail,
    } from "$lib/stores/list_store";
    import { summitLog } from "$lib/stores/summit_log_store";
    import { show_toast } from "$lib/stores/toast_store.svelte.js";
    import {
        trail,
        trails_create,
        trails_update,
    } from "$lib/stores/trail_store.js";
    import {
        valhallaStore,
        calculateRouteBetween,
        clearAnchors,
        clearRoute,
        deleteFromRoute,
        editRoute,
        insertIntoRoute,
        normalizeRouteTime,
        recalculateHeight,
        resetRoute,
        reverseRoute,
        setRoute,
        splitSegment,
        undo,
        redo,
        clearUndoRedoStack,
    } from "$lib/stores/valhalla_store.svelte.js";
    import { waypoint } from "$lib/stores/waypoint_store";
    import { getFileURL } from "$lib/util/file_util";
    import {
        formatDistance,
        formatElevation,
        formatTimeHHMM,
    } from "$lib/util/format_util";
    import { cropGPX, fromFile, gpx2trail } from "$lib/util/gpx_util";

    import { page } from "$app/state";
    import emptyStateTrailDark from "$lib/assets/svgs/empty_states/empty_state_trail_dark.svg";
    import emptyStateTrailLight from "$lib/assets/svgs/empty_states/empty_state_trail_light.svg";
    import Combobox, {
        type ComboboxItem,
    } from "$lib/components/base/combobox.svelte";
    import type { DropdownItem } from "$lib/components/base/dropdown.svelte";
    import Editor from "$lib/components/base/editor.svelte";
    import Search, {
        type SearchItem,
    } from "$lib/components/base/search.svelte";
    import RouteEditor from "$lib/components/trail/route_editor.svelte";
    import { TagCreateSchema } from "$lib/models/api/tag_schema.js";
    import { convertDMSToDD, haversineDistance } from "$lib/models/gpx/utils.js";
    import { Tag } from "$lib/models/tag.js";
    import {
        LocationDetails,
        searchLocationReverse,
        searchLocations,
        type LocationSearchResult,
    } from "$lib/stores/search_store.js";
    import { tags_index } from "$lib/stores/tag_store.js";
    import { theme } from "$lib/stores/theme_store.js";
    import { currentUser } from "$lib/stores/user_store.js";
    import { getIconForLocation } from "$lib/util/icon_util.js";
    import {
        createAnchorMarker,
        createEditTrailMapPopup,
        FontawesomeMarker,
        type OverpassPopupAction,
    } from "$lib/util/maplibre_util";
    import type { OverpassPopupActionFactory } from "$lib/vendor/maplibre-layer-manager/overpass-layer";
    import EXIF from "$lib/vendor/exif-js/exif.js";
    import { validator } from "@felte/validator-zod";
    import cryptoRandomString from "crypto-random-string";
    import { createForm } from "felte";
    import * as M from "maplibre-gl";
    import { onMount, tick, untrack } from "svelte";
    import { _ } from "svelte-i18n";
    import { backInOut } from "svelte/easing";
    import { fly } from "svelte/transition";
    import { z } from "zod";
    import Track from "$lib/models/gpx/track.js";
    import TrackSegment from "$lib/models/gpx/track-segment.js";
    import { Settings } from "$lib/models/settings";
    import { type Category } from "$lib/models/category.js";
    import TrailAnchorList from "$lib/components/trail/trail_anchor_list.svelte";
    import { geoJsonObjectToPositionsAndTimes } from "$lib/vendor/maplibre-elevation-profile/geojson.js";
    import type { Position } from "geojson";
    import GpxMetricsComputation from "$lib/models/gpx/gpx-metrics-computation.js";

    let { data } = $props();

    const cloneLists = (value: typeof data.lists) => {
        try {
            return typeof globalThis.structuredClone === "function"
                ? globalThis.structuredClone(value)
                : JSON.parse(JSON.stringify(value));
        } catch {
            return {
                ...value,
                items: [...(value?.items ?? [])],
            };
        }
    };

    // reactive snapshot used by TrailAnchorList – keep as a fresh array so the child sees updates
    let listData: any[] = $state([]);
    function syncAnchorListData() {
        listData = (valhallaStore.anchors ?? []).map((a) => ({ ...a }));
    }
    onMount(() => {
        syncAnchorListData();
    });

    let map: M.Map | undefined = $state();
    let mapPopup: M.Popup | undefined;
    let mapTrail: Trail[] = $state([]);
    let lists = $state(untrack(() => cloneLists(data.lists)));

    let waypointModal: WaypointModal;
    let summitLogModal: SummitLogModal;
    let listSelectModal: ListSelectModal;

    let loading = $state(false);
    let loadEditing = $state(false);

    let editingBasicInfo: boolean = $state(false);

    let photoFiles: File[] = $state([]);

    let gpxFile: File | Blob | null = null;

    let drawingActive = $state(false);
    let overwriteGPX = false;
    let draggingMarker = false;

    type LocationSearchItem = SearchItem & { value: LocationSearchResult };

    let searchDropdownItems: SearchItem[] = $state([]);
    let selectedSearchLocation: LocationSearchItem | null = $state(null);

    let cropStartMarker: FontawesomeMarker;
    let cropEndMarker: FontawesomeMarker;

    let croppedGPX: GPX | null = null;

    const ClientTrailCreateSchema = TrailCreateSchema.extend({
        expand: z
            .object({
                gpx_data: z.string().optional(),
                summit_logs_via_trail: z
                    .array(SummitLogCreateSchema)
                    .optional(),
                waypoints_via_trail: z
                    .array(
                        WaypointCreateSchema.extend({
                            marker: z.any().optional(),
                        }),
                    )
                    .optional(),
                tags: z.array(TagCreateSchema).optional(),
            })
            .optional(),
    });

    let routingOptions: RoutingOptions = $state({
        autoRouting: true,
        snapToPath: true,
        modeOfTransport: "pedestrian",
    });

    let savedAtLeastOnce = $state(false);

    let tagItems: ComboboxItem[] = $state([]);

    const {
        form,
        errors,
        data: formData,
        setFields,
    } = untrack(() =>
        createForm<z.infer<typeof ClientTrailCreateSchema>>({
            initialValues: {
                ...data.trail,
                public: data.trail.expand?.gpx_data
                    ? data.trail.public
                    : page.data.settings?.privacy?.trails === "public",
                category:
                    data.trail.category ||
                    page.data.settings?.category ||
                    $categories[0].id,
            },
            extend: validator({
                schema: ClientTrailCreateSchema,
            }),
            onSubmit: async (form) => {
                loading = true;
                try {
                    const htmlForm = document.getElementById(
                        "trail-form",
                    ) as HTMLFormElement;
                    const formData = new FormData(htmlForm);
                    if (!formData.get("public")) {
                        form.public = false;
                    }
                    form.photos = form.photos.filter(
                        (p) => !p.startsWith("data:image/svg+xml;base64"),
                    );

                    if (!form.photos?.length && !photoFiles.length) {
                        const canvas = document.querySelector(
                            "#map .maplibregl-canvas",
                        ) as HTMLCanvasElement;

                        const dataURL = canvas.toDataURL("image/webp", 0.3);
                        const response = await fetch(dataURL);
                        const blob = await response.blob();
                        photoFiles = [new File([blob], "route")];
                    }

                    form.expand!.gpx_data = valhallaStore.route.toString();
                    if (form.expand!.gpx_data && overwriteGPX) {
                        gpxFile = new Blob([form.expand!.gpx_data], {
                            type: "text/xml",
                        });
                    }

                    if (
                        (!form.lat || !form.lon) &&
                        valhallaStore.route.trk?.at(0)?.trkseg?.at(0)?.trkpt?.at(
                            0,
                        )
                    ) {
                        form.lat = valhallaStore.route.trk
                            ?.at(0)
                            ?.trkseg?.at(0)
                            ?.trkpt?.at(0)?.$.lat;
                        form.lon = valhallaStore.route.trk
                            ?.at(0)
                            ?.trkseg?.at(0)
                            ?.trkpt?.at(0)?.$.lon;
                    }

                    if (page.params.id === "new" && !savedAtLeastOnce) {
                        const createdTrail = await trails_create(
                            form as Trail,
                            photoFiles,
                            gpxFile,
                        );

                        setFields(createdTrail);
                        trail.set(createdTrail);
                    } else {
                        const updatedTrail = await trails_update(
                            $trail,
                            form as Trail,
                            photoFiles,
                            gpxFile,
                        );
                        setFields(updatedTrail);
                    }
                    photoFiles = [];

                    savedAtLeastOnce = true;
                    show_toast({
                        type: "success",
                        icon: "check",
                        text: $_("trail-saved-successfully"),
                    });
                } catch (e) {
                    console.error(e);

                    show_toast({
                        type: "error",
                        icon: "close",
                        text: $_("error-saving-trail"),
                    });
                } finally {
                    loading = false;
                }
            },
        }),
    );

    onMount(async () => {
        clearAnchors();
        syncAnchorListData();
        clearRoute();
        clearUndoRedoStack();

        if ($formData.expand!.gpx_data) {
            $formData.id ??= cryptoRandomString({ length: 15 });
            const gpx = GPX.parse($formData.expand!.gpx_data);
            if (!(gpx instanceof Error)) {
                if (gpx.rte && !gpx.trk) {
                    gpx.trk = [
                        new Track({
                            trkseg: [
                                new TrackSegment({
                                    trkpt: gpx.rte?.at(0)?.rtept,
                                }),
                            ],
                        }),
                    ];
                    gpx.rte = undefined;
                }

                setRoute(gpx);
                await initRouteAnchors(gpx);
                refreshAnchorMetrics();

                updateTrailOnMap();
            }
        }
    });

    function openFileBrowser() {
        document.getElementById("fileInput")!.click();
    }

    async function handleFileSelection() {
        const selectedFile = (
            document.getElementById("fileInput") as HTMLInputElement
        ).files?.[0];

        if (!selectedFile) {
            return;
        }

        clearWaypoints();
        clearAnchors();
        syncAnchorListData();
        clearUndoRedoStack();
        clearRoute();
        mapTrail = [];
        drawingActive = false;
        overwriteGPX = false;

        const { gpxData, gpxFile: file } = await fromFile(selectedFile);
        gpxFile = file;

        try {
            const prevId = $formData.id;
            const parseResult = await gpx2trail(gpxData, selectedFile.name);       
            setFields(parseResult.trail);
            $formData.id = prevId ?? cryptoRandomString({ length: 15 });
            $formData.expand!.gpx_data = gpxData;

            setFields(
                "category",
                page.data.settings.category || $categories[0].id,
            );
            setFields(
                "public",
                page.data.settings?.privacy?.trails === "public",
            );

            // const log = new SummitLog(parseResult.trail.date as string, {
            //     distance: $formData.distance,
            //     elevation_gain: $formData.elevation_gain,
            //     elevation_loss: $formData.elevation_loss,
            //     duration: $formData.duration
            //         ? $formData.duration * 60
            //         : undefined,
            // });

            // log.expand!.gpx_data = gpxData;
            // const blob = new Blob([gpxData], { type: selectedFile.type });
            // log._gpx = new File([blob], selectedFile.name, {
            //     type: selectedFile.type,
            // });

            // $formData.expand!.summit_logs?.push(log);

            if (parseResult.gpx.rte?.length && !parseResult.gpx.trk) {
                parseResult.gpx.trk = [
                    new Track({
                        trkseg: [
                            new TrackSegment({
                                trkpt: parseResult.gpx.rte?.at(0)?.rtept,
                            }),
                        ],
                    }),
                ];
                parseResult.gpx.rte = undefined;
            }
            setRoute(parseResult.gpx);
            await initRouteAnchors(parseResult.gpx);
            refreshAnchorMetrics();

            updateTrailOnMap();
        } catch (e) {
            console.error(e);

            show_toast({
                icon: "close",
                type: "error",
                text: $_("error-reading-file"),
            });
            return;
        }
        const r = await searchLocationReverse($formData.lat!, $formData.lon!);

        if (r) {
            setFields("location", r);
        }
    }

    function clearWaypoints() {
        for (const waypoint of $formData.expand!.waypoints_via_trail ?? []) {
            waypoint.marker?.remove();
        }
        $formData.expand!.waypoints_via_trail = [];
    }

    async function initRouteAnchors(gpx: GPX, addToMap: boolean = false) {
        const segments = gpx.trk?.at(0)?.trkseg ?? [];

        for (let i = 0; i < segments.length; i++) {
            const segment = segments[i];
            const points = segment.trkpt ?? [];

            if (points.length > 0) {
                addAnchor(
                    points[0].$.lat!,
                    points[0].$.lon!,
                    valhallaStore.anchors.length,
                    addToMap,
                );
            }
            if (i == segments.length - 1) {
                addAnchor(
                    points[points.length - 1].$.lat!,
                    points[points.length - 1].$.lon!,
                    valhallaStore.anchors.length,
                    addToMap,
                );
            }
        }
    }

    let routePositions: Position[] = [];
    // caches for cumulative metrics aligned to flattened GPX points
    let flatPoints: any[] = [];
    let cumGain: number[] = [];
    let cumLoss: number[] = [];
    // running cumulative distance for each flattened point (mirrors smoothing metrics)
    let cumDistance: number[] = [];
    let pointIndexByCoord: Map<string, number> = new Map();

    function coordKey(lat?: number, lon?: number) {
        // normalize to reduce floating noise; anchors use exact route points after snap
        return `${(lon ?? 0).toFixed(6)}|${(lat ?? 0).toFixed(6)}`;
    }

    function initRoutePositions() {
        if (routePositions.length > 0) {
            return;
        }

        let trackJson = valhallaStore.route.toGeoJSON();       
        routePositions = geoJsonObjectToPositionsAndTimes(trackJson).positions;
    }
    function initCumulativeMetrics() {
        if (flatPoints.length > 0 && cumGain.length === flatPoints.length) {
            return;
        }
        flatPoints = valhallaStore.route.flatten();
        cumGain = new Array(flatPoints.length).fill(0);
        cumLoss = new Array(flatPoints.length).fill(0);
        cumDistance = new Array(flatPoints.length).fill(0);
        pointIndexByCoord = new Map();

        const metrics = new GpxMetricsComputation(5, 5);
        if (flatPoints.length > 0) {
            // seed index map for all points
            for (let i = 0; i < flatPoints.length; i++) {
                const pt = flatPoints[i];
                pointIndexByCoord.set(coordKey(pt.$.lat, pt.$.lon), i);
            }
            // run the same algorithm as GPX.getTotals across the entire flattened series
            for (let i = 1; i < flatPoints.length; i++) {
                metrics.addAndFilter(flatPoints[i]);
                cumGain[i] = metrics.totalElevationGainSmoothed;
                cumLoss[i] = metrics.totalElevationLossSmoothed;
                cumDistance[i] = metrics.totalDistance;
            }
        }
    }
    function resetRoutePositions() {
        routePositions = [];
        flatPoints = [];
        cumGain = [];
        cumLoss = [];
        cumDistance = [];
        pointIndexByCoord = new Map();
    }

    function openMarkerPopup(waypoint: Waypoint) {
        waypoint.marker?.togglePopup();
    }

    function handleWaypointMenuClick(
        currentWaypoint: Waypoint,
        index: number,
        item: DropdownItem,
    ) {
        if (item.value === "edit") {
            waypoint.set(currentWaypoint);
            waypointModal.openModal();
        } else if (item.value === "delete") {
            currentWaypoint.marker?.remove();
            deleteWaypoint(index);
        }
    }

    function beforeWaypointModalOpen(lat?: number, lon?: number) {
        if (!map) {
            return;
        }
        const mapCenter = map.getCenter();
        waypoint.set(new Waypoint(lat ?? mapCenter.lat, lon ?? mapCenter.lng));
        waypointModal.openModal();
    }

    function deleteWaypoint(index: number) {
        const wp = $formData.expand!.waypoints_via_trail?.splice(index, 1);

        if (!$formData.expand!.waypoints_via_trail?.length) {
            $formData.expand!.waypoints_via_trail = [];
        }
        $formData.expand!.waypoints_via_trail = $formData.expand!.waypoints_via_trail;

        // updateTrailOnMap();
    }

    function saveWaypoint(savedWaypoint: Waypoint) {
        let editedWaypointIndex =
            $formData.expand!.waypoints_via_trail?.findIndex(
                (s) => s.id == savedWaypoint.id,
            ) ?? -1;

        if (editedWaypointIndex >= 0) {
            $formData.expand!.waypoints_via_trail![editedWaypointIndex] = savedWaypoint;
        } else {
            savedWaypoint.id = cryptoRandomString({ length: 15 });
            $formData.expand!.waypoints_via_trail = [
                ...($formData.expand!.waypoints_via_trail ?? []),
                savedWaypoint,
            ];

            // updateTrailOnMap();
        }
    }

    function moveMarker(marker: M.Marker, wpId?: string) {
        const position = marker.getLngLat();
        const editableWaypointIndex =
            $formData.expand!.waypoints_via_trail?.findIndex((w) => w.id == wpId) ?? -1;
        const editableWaypoint =
            $formData.expand!.waypoints_via_trail![editableWaypointIndex];
        if (!editableWaypoint) {
            return;
        }
        editableWaypoint.lat = position.lat;
        editableWaypoint.lon = position.lng;
        $formData.expand!.waypoints_via_trail = [...($formData.expand!.waypoints_via_trail ?? [])];
        // updateTrailOnMap();
    }

    function beforeSummitLogModalOpen() {
        const newSummitLog = new SummitLog(
            new Date().toISOString().split("T")[0],
        );
        newSummitLog.author = $currentUser?.actor;
        summitLog.set(newSummitLog);
        summitLogModal.openModal();
    }

    function saveSummitLog(log: SummitLog) {
        let editedSummitLogIndex =
            $formData.expand!.summit_logs_via_trail?.findIndex(
                (s) => s.id == log.id,
            );
        if ((editedSummitLogIndex ?? -1) >= 0) {
            $formData.expand!.summit_logs_via_trail![editedSummitLogIndex!] =
                log;
        } else {
            log.id = cryptoRandomString({ length: 15 });
            $formData.expand!.summit_logs_via_trail = [
                ...($formData.expand!.summit_logs_via_trail ?? []),
                log,
            ];
        }
    }

    function handleSummitLogMenuClick(
        currentSummitLog: SummitLog,
        index: number,
        item: DropdownItem,
    ) {
        if (item.value === "edit") {
            summitLog.set(currentSummitLog);
            summitLogModal.openModal();
        } else if (item.value === "delete") {
            $formData.expand!.summit_logs_via_trail?.splice(index, 1);
            $formData.expand!.summit_logs_via_trail =
                $formData.expand!.summit_logs_via_trail;
        }
    }

    async function handleListSelection(list: List) {
        if (!$formData.id) {
            return;
        }
        try {
            if (list.trails?.includes($formData.id!)) {
                list = await lists_remove_trail(list, $formData as Trail);
            } else {
                list = await lists_add_trail(list, $formData as Trail);
            }
            const index = lists.items.findIndex((l: List) => l.id == list.id);
            if (index >= 0) {
                lists.items[index] = list;
            }
            // await lists_index({ q: "", author: $currentUser?.id ?? "" }, 1, -1);
        } catch (e) {
            console.error(e);
            show_toast({
                type: "error",
                icon: "close",
                text: "Error adding trail to list.",
            });
        }
    }

    async function startDrawing() {
        if (!map) {
            return;
        }
        drawingActive = true;
        
        for (const anchor of valhallaStore.anchors) {
            anchor.marker?.addTo(map);
        }
    }

    async function stopDrawing() {
        drawingActive = false;
        for (const anchor of valhallaStore.anchors) {
            anchor.marker?.remove();
        }
        toggleCropMarkers(false);
        clearUndoRedoStack();

        if (valhallaStore.route.trk?.at(0)?.trkseg?.at(0)?.trkpt?.at(0)) {
            $formData.lat = valhallaStore.route.trk
                ?.at(0)
                ?.trkseg?.at(0)
                ?.trkpt?.at(0)?.$.lat;
            $formData.lon = valhallaStore.route.trk
                ?.at(0)
                ?.trkseg?.at(0)
                ?.trkpt?.at(0)?.$.lon;
        }

        if ($formData.lat && $formData.lon) {
            const r = await searchLocationReverse($formData.lat, $formData.lon);
            if (r) {
                setFields("location", r);
            }
        }
    }

    async function handleMapClick(e: M.MapMouseEvent) {
        if (!drawingActive) {
            if (
                (
                    e.originalEvent.target as HTMLElement
                ).tagName.toLowerCase() !== "canvas"
            ) {
                return;
            }
            mapPopup?.remove();

            mapPopup = createEditTrailMapPopup(e.lngLat, () => {
                mapPopup?.remove();
                beforeWaypointModalOpen(e.lngLat.lat, e.lngLat.lng);
            });
            mapPopup.addTo(map!);
        } else {
            const anchorCount = valhallaStore.anchors.length;
            if (anchorCount == 0) {
                addAnchor(
                    e.lngLat.lat,
                    e.lngLat.lng,
                    valhallaStore.anchors.length,
                );
            } else {
                addAnchorAndRecalculate(e.lngLat.lat, e.lngLat.lng);
            };
        }
    }

    async function addAnchorAndRecalculate(lat: number, lon: number) {

        loadEditing = true;

        const previousAnchor =
            valhallaStore.anchors[valhallaStore.anchors.length - 1];
        let anchor = addAnchor(lat, lon, valhallaStore.anchors.length);
        const markerText = startAnchorLoading(anchor);
        let success = false;
        try {
            const routeWaypoints = await calculateRouteBetween(
                previousAnchor.lat,
                previousAnchor.lon,
                lat,
                lon,
                routingOptions,
            );

            // 'snap' anchor to route endpoint, if autrouting is enabled
            if (routingOptions.autoRouting && routeWaypoints.length > 0) {
                const snappedPoint = routeWaypoints.at(-1);
                const snappedLat = snappedPoint?.$?.lat;
                const snappedLon = snappedPoint?.$?.lon;
                if (
                    typeof snappedLat === "number" &&
                    typeof snappedLon === "number"
                ) {
                    const snappedAnchorIndex = valhallaStore.anchors.length - 1;
                    const updatedAnchor =
                        updateAnchorAt(snappedAnchorIndex, (existing) => ({
                            ...existing,
                            lat: snappedLat,
                            lon: snappedLon,
                        })) ?? anchor;
                    anchor = updatedAnchor;
                    updatedAnchor.marker?.setLngLat([snappedLon, snappedLat]);
                }
            }

            insertIntoRoute(routeWaypoints);

            updateTrailWithRouteData();
            normalizeRouteTime();
            success = true;
        } catch (e) {
            console.error(e);
            show_toast({
                text: "Error calculating route",
                icon: "close",
                type: "error",
            });
            rollbackAnchorInsertion(anchor);
        } finally {
            if (success) {
                stopAnchorLoading(anchor, markerText);
            }
        }

        loadEditing = false;
    }

    function findClosestRouteIndex(lat?: number, lon?: number) {
        if (lat == null || lon == null) return -1;
        initRoutePositions();
        initCumulativeMetrics();
        if (flatPoints.length === 0) return -1;

        let nearest = -1;
        let nearestDist = Number.POSITIVE_INFINITY;
        for (let i = 0; i < flatPoints.length; i++) {
            const pt = flatPoints[i];
            const ptLat = pt.$?.lat ?? pt.lat;
            const ptLon = pt.$?.lon ?? pt.lon;
            if (ptLat == null || ptLon == null) {
                continue;
            }
            const dist = haversineDistance(lat, lon, ptLat, ptLon);
            if (dist < nearestDist) {
                nearestDist = dist;
                nearest = i;
                // early exit for perfect match (within ~1mm)
                if (nearestDist < 1e-3) {
                    break;
                }
            }
        }
        return nearest;
    }

    function getDistanceAndElevationGainLossFromPreviousAnchor(anchor: ValhallaAnchor, index: number) {
        if (index == 0 || index >= valhallaStore.anchors.length) {
            return;
        }

        // ensure caches are ready
        initRoutePositions();
        initCumulativeMetrics();

        const a1 = valhallaStore.anchors[index - 1];
        const a2 = valhallaStore.anchors[index];

        // find indices in flattened points (anchors are snapped to route endpoints)
        let idxPrev = pointIndexByCoord.get(coordKey(a1.lat, a1.lon)) ?? -1;
        let idx = pointIndexByCoord.get(coordKey(a2.lat, a2.lon)) ?? -1;
        if (idxPrev < 0) {
            idxPrev = findClosestRouteIndex(a1.lat, a1.lon);
        }
        if (idx < 0) {
            idx = findClosestRouteIndex(a2.lat, a2.lon);
        }
        if (idxPrev < 0 || idx < 0 || idx <= idxPrev) {
            return;
        }

        // distances from GPX metrics (same as totals)
        const distance = (cumDistance[idx] ?? 0) - (cumDistance[idxPrev] ?? 0);

        // elevation gain/loss aligned to totals' smoothing and thresholds
        const gain = (cumGain[idx] ?? 0) - (cumGain[idxPrev] ?? 0);
        const loss = (cumLoss[idx] ?? 0) - (cumLoss[idxPrev] ?? 0);

        updateAnchorAt(index, (existing) => ({
            ...existing,
            distance: Math.max(0, distance),
            elevation_gain: Math.max(0, gain),
            elevation_loss: Math.max(0, loss),
        }));
    }

    function refreshAnchorMetrics() {
        resetRoutePositions();
        for (let i = 1; i < valhallaStore.anchors.length; i++) {
            getDistanceAndElevationGainLossFromPreviousAnchor(
                valhallaStore.anchors[i],
                i,
            );
        }
        syncAnchorListData();
        ensureAnchorLocationNames();
    }

    async function updateAnchorLocationName(anchor: ValhallaAnchor) {
        (async () => {
            try {
                const locationName = await searchLocationReverse(
                    anchor.lat,
                    anchor.lon,
                    LocationDetails.ALL,
                );
                if (!locationName) {
                    return;
                }

                const updated = updateAnchorById(anchor.id, (current) => ({
                    ...current,
                    locationName,
                }));
                if (updated) {
                    syncAnchorListData();
                }
            } catch (error) {
                console.error("Failed to resolve anchor location", error);
            }
        })();
    }

    function ensureAnchorLocationNames() {
        for (const anchor of valhallaStore.anchors) {
            if (!anchor.locationName) {
                updateAnchorLocationName(anchor);
            }
        }
    }

    function cloneAnchors() {
        return [...valhallaStore.anchors];
    }

    function insertAnchorAtIndex(index: number, anchor: ValhallaAnchor) {
        const nextAnchors = cloneAnchors();
        nextAnchors.splice(index, 0, anchor);
        valhallaStore.anchors = nextAnchors;
    }

    function removeAnchorAtIndex(index: number) {
        if (index < 0 || index >= valhallaStore.anchors.length) {
            return;
        }
        const nextAnchors = cloneAnchors();
        nextAnchors.splice(index, 1);
        valhallaStore.anchors = nextAnchors;
    }

    function updateAnchorAt(
        index: number,
        updater: (anchor: ValhallaAnchor) => ValhallaAnchor,
    ) {
        if (index < 0 || index >= valhallaStore.anchors.length) {
            return null;
        }
        const nextAnchors = cloneAnchors();
        const current = nextAnchors[index];
        if (!current) {
            return null;
        }
        nextAnchors[index] = updater({ ...current });
        valhallaStore.anchors = nextAnchors;
        return nextAnchors[index];
    }

    function updateAnchorById(
        anchorId: string,
        updater: (anchor: ValhallaAnchor) => ValhallaAnchor,
    ) {
        const index = valhallaStore.anchors.findIndex((a) => a.id === anchorId);
        if (index === -1) {
            return null;
        }
        return updateAnchorAt(index, updater);
    }

    function addAnchor(
        lat: number,
        lon: number,
        index: number,
        addtoMap: boolean = true,
    ) {
        const anchor: ValhallaAnchor = {
            id: cryptoRandomString({ length: 15 }),
            lat: lat,
            lon: lon,
        };
        const marker = createAnchorMarker(
            lat,
            lon,
            index + 1,
            () => {
                removeAnchor(
                    valhallaStore.anchors.findIndex((a) => a.id == anchor.id),
                );
            },
            () => {
                const thisAnchor = valhallaStore.anchors.find(
                    (a) => a.id == anchor.id,
                );
                addAnchorAndRecalculate(
                    thisAnchor?.lat ?? lat,
                    thisAnchor?.lon ?? lon,
                );
                marker.togglePopup();
            },
            (e) => {
                draggingMarker = true;
            },
            async (_) => {
                if (!drawingActive) {
                    return;
                }

                loadEditing = false;

                const anchorIndex = valhallaStore.anchors.findIndex(
                    (a) => a.id == anchor.id,
                );
                const position = marker.getLngLat();
                updateAnchorAt(anchorIndex, (existing) => ({
                    ...existing,
                    lat: position.lat,
                    lon: position.lng,
                }));

                await recalculateRoute(anchorIndex);

                draggingMarker = false;
                loadEditing = false;
            },
        );
        
        if (addtoMap && map) {
            marker.addTo(map);
        }
        
        anchor.marker = marker;
        insertAnchorAtIndex(index, anchor);

        const schedulePostAddUpdates = () => {
            const currentIndex = valhallaStore.anchors.findIndex(
                (a) => a.id === anchor.id,
            );
            if (
                currentIndex > 0 &&
                currentIndex < valhallaStore.anchors.length
            ) {
                const insertedAnchor = valhallaStore.anchors[currentIndex];
                getDistanceAndElevationGainLossFromPreviousAnchor(
                    insertedAnchor,
                    currentIndex,
                );
            }
            syncAnchorListData();
            updateAnchorLocationName(anchor);
        };

        setTimeout(schedulePostAddUpdates, 0);

        return anchor;
    }

    function startAnchorLoading(anchor: ValhallaAnchor) {
        const markerIcon = anchor.marker?.getElement();
        if (!markerIcon) {
            return null;
        }
        markerIcon.classList.add("spinner", "spinner-light", "spinner-small");
        const savedMarkerNumber = markerIcon.textContent;
        markerIcon.textContent = "";

        return savedMarkerNumber;
    }

    async function stopAnchorLoading(anchor: ValhallaAnchor, index: string | null) {
        const markerIcon = anchor.marker?.getElement();
        if (!markerIcon || !index) {
            return;
        }
        markerIcon.classList.remove(
            "spinner",
            "spinner-light",
            "spinner-small",
        );
        markerIcon.textContent = index;
        
        resetRoutePositions();
        const i = valhallaStore.anchors.findIndex((a) => a.id == anchor.id);
        if (i >= 0) {
            getDistanceAndElevationGainLossFromPreviousAnchor(valhallaStore.anchors[i], i);
        }
        syncAnchorListData();
    }

    async function removeAnchor(anchorIndex: number) {
        if (!drawingActive) {
            return;
        }

        loadEditing = true;

        resetRoutePositions();
        valhallaStore.anchors[anchorIndex]?.marker?.remove();
        removeAnchorAtIndex(anchorIndex);
        refreshAnchorLabels(anchorIndex);
        syncAnchorListData();
        if (anchorIndex == 0) {
            deleteFromRoute(anchorIndex);
            if ($formData.expand?.gpx_data) {
                updateTrailWithRouteData();
            }
        } else if (anchorIndex == valhallaStore.anchors.length) {
            deleteFromRoute(anchorIndex - 1);
            updateTrailWithRouteData();
        } else {
            deleteFromRoute(anchorIndex - 1);
            await recalculateRoute(anchorIndex);
        }

        syncAnchorListData();

        loadEditing = false;
    }

    async function recalculateRoute(anchorIndex: number) {
        const markerText = startAnchorLoading(
            valhallaStore.anchors[anchorIndex],
        );

        const anchor = valhallaStore.anchors[anchorIndex];
        if (!anchor) {
            return;
        }
        let nextRouteSegment;
        let previousRouteSegment;
        try {
            if (anchorIndex < valhallaStore.anchors.length - 1) {
                const nextAnchor = valhallaStore.anchors[anchorIndex + 1];

                nextRouteSegment = await calculateRouteBetween(
                    anchor.lat,
                    anchor.lon,
                    nextAnchor.lat,
                    nextAnchor.lon,
                    routingOptions,
                );
            }
            if (anchorIndex > 0) {
                const previousAnchor = valhallaStore.anchors[anchorIndex - 1];
                previousRouteSegment = await calculateRouteBetween(
                    previousAnchor.lat,
                    previousAnchor.lon,
                    anchor.lat,
                    anchor.lon,
                    routingOptions,
                );
            }

            if (nextRouteSegment) {
                editRoute(anchorIndex, nextRouteSegment);
            }
            if (previousRouteSegment) {
                editRoute(anchorIndex - 1, previousRouteSegment);
            }
            updateTrailWithRouteData();
            normalizeRouteTime();
        } catch (e) {
            console.error(e);
            show_toast({
                text: "Error calculating route",
                icon: "close",
                type: "error",
            });
        } finally {
            stopAnchorLoading(valhallaStore.anchors[anchorIndex], markerText);
        }
    }

    async function handleSegmentDragEnd(data: {
        segment: number;
        event: M.MapMouseEvent;
    }) {
        if (draggingMarker) {
            return;
        }

        await insertAnchorWithinRouteSegment(
            data.segment,
            data.event.lngLat.lat,
            data.event.lngLat.lng,
        );
    }

    function refreshAnchorLabels(startIndex: number) {
        for (let i = startIndex; i < valhallaStore.anchors.length; i++) {
            const anchor = valhallaStore.anchors[i];
            const markerIcon = anchor.marker?.getElement();
            if (markerIcon) {
                markerIcon.textContent = `${i + 1}`;
                anchor
                    .marker!.getPopup()
                    ._content.getElementsByTagName("h5")[0].textContent =
                    $_("route-point") + " #" + (i + 1);
            }
        }
    }

    function updateFollowingAnchors(segment: number) {
        refreshAnchorLabels(segment + 1);
    }

    function rollbackAnchorInsertion(anchor: ValhallaAnchor) {
        const index = valhallaStore.anchors.findIndex(
            (a) => a.id === anchor.id,
        );
        if (index === -1) {
            return;
        }

        anchor.marker?.remove();
        removeAnchorAtIndex(index);
        refreshAnchorLabels(index);
        syncAnchorListData();
    }

    async function handleSegmentClick(data: {
        segment: number;
        event: M.MapMouseEvent;
    }) {
        addAnchor(
            data.event.lngLat.lat,
            data.event.lngLat.lng,
            data.segment + 1,
        );

        splitSegment(data.segment, data.event.lngLat);
        updateFollowingAnchors(data.segment);
        updateTrailWithRouteData();
    }

    function reverseTrail() {
        reverseRoute();

        updateTrailWithRouteData();
        syncAnchorListData();
        refreshAnchorMetrics();
    }

    function resetTrail() {
        resetRoute();

        updateTrailWithRouteData();
        syncAnchorListData();
        refreshAnchorMetrics();
    }

    async function recalculateElevationData() {
        await recalculateHeight();

        updateTrailWithRouteData();
    }

    function toggleCropMarkers(active: boolean) {
        if (active) {
            cropStartMarker?.setOpacity("1");
            cropEndMarker?.setOpacity("1");
        } else {
            cropStartMarker?.setOpacity("0");
            cropEndMarker?.setOpacity("0");

            updateTotals(valhallaStore.route);
        }
    }

    function updateCropMarkers(range: [start: number, end: number]) {
        if (!cropStartMarker || !cropEndMarker) {
            cropStartMarker = new FontawesomeMarker(
                {
                    id: "crop-start-marker",
                    icon: "fa-regular fa-circle",
                    fontSize: "xs",
                    style: "w-6",
                    width: 4,
                    backgroundColor: "bg-primary",
                    fontColor: "white",
                },
                {},
            );
            cropEndMarker = new FontawesomeMarker(
                {
                    id: "crop-end-marker",
                    icon: "fa fa-flag-checkered",
                    fontSize: "xs",
                    style: "w-6",
                    width: 4,
                    backgroundColor: "bg-primary",
                    fontColor: "white",
                },
                {},
            );

            cropStartMarker.setLngLat([0, 0]).addTo(map!);
            cropEndMarker.setLngLat([0, 0]).addTo(map!);
        }
        const [start, end] = range;

        const flatRoute = valhallaStore.route.flatten();

        const targetStartDistance =
            valhallaStore.route.features.distance * (start / 100);
        const [startLon, startLat, startIndex] = getCoordinateAtDistance(
            flatRoute,
            valhallaStore.route.features.cumulativeDistance,
            targetStartDistance,
        );

        const targetEndDistance =
            valhallaStore.route.features.distance * (end / 100);
        const [endLon, endLat, endIndex] = getCoordinateAtDistance(
            flatRoute,
            valhallaStore.route.features.cumulativeDistance,
            targetEndDistance,
        );

        cropStartMarker.setLngLat([startLon, startLat]);
        cropEndMarker.setLngLat([endLon, endLat]);

        croppedGPX = cropGPX(
            flatRoute[startIndex],
            flatRoute[endIndex],
            valhallaStore.route,
        );

        updateTotals(croppedGPX);
    }

    async function confirmCrop() {
        if (!croppedGPX) {
            return;
        }
        setRoute(croppedGPX, true);
        updateTrailWithRouteData();
        clearAnchors();
        syncAnchorListData();
        await initRouteAnchors(croppedGPX, true);
        refreshAnchorMetrics();
    }

    function getCoordinateAtDistance(
        points: GPXWaypoint[],
        cumulative: number[],
        target: number,
    ) {
        let low = 0,
            high = cumulative.length - 1;

        while (low < high) {
            const mid = Math.floor((low + high) / 2);
            if (cumulative[mid] < target) low = mid + 1;
            else high = mid;
        }

        const i = Math.max(1, low);
        const prevDist = cumulative[i - 1];
        const nextDist = cumulative[i];
        const ratio = (target - prevDist) / (nextDist - prevDist);

        const prev = points[i - 1];
        const next = points[i];

        return [
            prev.$.lon! + (next.$.lon! - prev.$.lon!) * ratio,
            prev.$.lat! + (next.$.lat! - prev.$.lat!) * ratio,
            i,
        ];
    }

    function updateTrailWithRouteData() {
        overwriteGPX = true;
        updateTotals(valhallaStore.route);

        if (!$formData.id) {
            $formData.id = cryptoRandomString({ length: 15 });
        }
        updateTrailOnMap();
    }

    function updateTotals(gpx: GPX) {
        const totals = gpx.features;
        formData.set({
            ...$formData,
            distance: totals.distance,
            duration: totals.duration / 1000,
            elevation_gain: totals.elevationGain,
            elevation_loss: totals.elevationLoss,
        });
    }

    function updateTrailOnMap() {
        const {
            id,
            name,
            distance,
            duration,
            elevation_gain,
            elevation_loss,
            lat,
            lon,
            expand,
        } = $formData;

        const expandForMap: Trail["expand"] = {
            ...(expand ? { ...expand } : {}),
            gpx: valhallaStore.route,
        };
        if (expandForMap?.gpx_data) {
            delete expandForMap.gpx_data;
        }

        mapTrail = [
            {
                id,
                name,
                distance,
                duration,
                elevation_gain,
                elevation_loss,
                lat,
                lon,
                expand: expandForMap,
            } as Trail,
        ];
    }

    function handleSearchClick(item: SearchItem) {
        map?.flyTo({
            center: [item.value.lon, item.value.lat],
            zoom: 13,
            animate: false,
        });
        selectedSearchLocation = item as LocationSearchItem;
    }

    function clearSelectedSearchLocation() {
        selectedSearchLocation = null;
    }

    function findNearestRouteSegment(lat: number, lon: number) {
        const segments = valhallaStore.route.trk?.at(0)?.trkseg ?? [];
        let bestSegment = -1;
        let bestDistance = Infinity;

        for (let i = 0; i < segments.length; i++) {
            const points = segments[i].trkpt ?? [];
            for (const point of points) {
                const pointLat = point.$.lat;
                const pointLon = point.$.lon;
                if (
                    typeof pointLat !== "number" ||
                    typeof pointLon !== "number"
                ) {
                    continue;
                }
                const distance = haversineDistance(
                    lat,
                    lon,
                    pointLat,
                    pointLon,
                );
                if (distance < bestDistance) {
                    bestDistance = distance;
                    bestSegment = i;
                }
            }
        }

        return bestSegment;
    }

    async function insertAnchorWithinRouteSegment(
        segmentIndex: number,
        lat: number,
        lon: number,
    ) {
        if (
            segmentIndex < 0 ||
            segmentIndex >= valhallaStore.anchors.length - 1
        ) {
            return;
        }

        loadEditing = true;

        let anchor = addAnchor(lat, lon, segmentIndex + 1);

        console.log("Inserting anchor within segment", segmentIndex, anchor);

        const markerText = startAnchorLoading(anchor);
        let success = false;

        const previousAnchor = valhallaStore.anchors[segmentIndex];
        const nextAnchor = valhallaStore.anchors[segmentIndex + 2];

        console.log("Previous anchor:", previousAnchor);
        console.log("Next anchor:", nextAnchor);

        if (!previousAnchor || !nextAnchor) {
            rollbackAnchorInsertion(anchor);
            loadEditing = false;
            return;
        }

        try {
            const previousRouteSegment = await calculateRouteBetween(
                previousAnchor.lat,
                previousAnchor.lon,
                anchor.lat,
                anchor.lon,
                routingOptions,
            );
            const nextRouteSegment = await calculateRouteBetween(
                anchor.lat,
                anchor.lon,
                nextAnchor.lat,
                nextAnchor.lon,
                routingOptions,
            );

            console.log("Previous route segment:", previousRouteSegment);
            console.log("Next route segment:", nextRouteSegment);

            await editRoute(segmentIndex, previousRouteSegment);
            await insertIntoRoute(nextRouteSegment, segmentIndex + 1);
            normalizeRouteTime();
            updateTrailWithRouteData();
            updateFollowingAnchors(segmentIndex);
            //await recalculateTrailFromAnchors();
            success = true;
        } catch (e) {
            console.error(e);
            show_toast({
                text: "Error calculating route",
                icon: "close",
                type: "error",
            });
            rollbackAnchorInsertion(anchor);
        } finally {
            if (success) {
                stopAnchorLoading(anchor, markerText);
            }
            loadEditing = false;
        }
    }

    async function addCoordinatesAsAnchor(lat: number, lon: number) {
        if (!drawingActive) {
            return;
        }

        const segments = valhallaStore.route.trk?.at(0)?.trkseg ?? [];
        if (
            valhallaStore.anchors.length >= 2 &&
            segments.length >= valhallaStore.anchors.length - 1
        ) {
            const segmentIndex = findNearestRouteSegment(lat, lon);
            if (segmentIndex >= 0) {
                await insertAnchorWithinRouteSegment(segmentIndex, lat, lon);
                return;
            }
        }

        if (valhallaStore.anchors.length === 0) {
            addAnchor(lat, lon, valhallaStore.anchors.length);
        } else {
            await addAnchorAndRecalculate(lat, lon);
        }
    }

    async function useSelectedLocationAsAnchor() {
        if (!selectedSearchLocation) {
            return;
        }

        const { lat, lon } = selectedSearchLocation.value;
        if (typeof lat !== "number" || typeof lon !== "number") {
            return;
        }

        await addCoordinatesAsAnchor(lat, lon);
        selectedSearchLocation = null;
    }

    const buildPoiAnchorAction: OverpassPopupActionFactory = (
        _feature,
        coordinates,
    ) => {
        const [lon, lat] = coordinates;
        if (typeof lat !== "number" || typeof lon !== "number") {
            return null;
        }

        const action: OverpassPopupAction = {
            label: $_("use-as-route-anchor"),
            icon: "fa fa-route",
            onClick: () => {
                addCoordinatesAsAnchor(lat, lon);
            },
        };

        if (!drawingActive) {
            action.disabled = true;
            action.helperText = $_("start-route-editing-to-add-anchor");
        }

        return action;
    };

    async function searchCities(q: string) {
        const r = await searchLocations(q);
        searchDropdownItems = r.map((h) => ({
            text: h.name,
            description: h.description,
            value: h,
            icon: getIconForLocation(h),
        }));
    }

    function getTrailTags() {
        return (
            $formData.expand?.tags?.map((t) => ({
                text: t.name,
                value: t,
            })) ?? []
        );
    }

    function setTrailTags(items: ComboboxItem[]) {
        $formData.expand!.tags = items.map((i) =>
            i.value ? i.value : new Tag(i.text),
        );
    }

    async function searchTags(q: string) {
        const result = await tags_index(q);
        tagItems = result.items.map((t) => ({ text: t.name, value: t }));
    }

    function openPhotoBrowser() {
        document.getElementById("waypoint-photo-input")!.click();
    }

    class GPXCoord
    {
        longitude!: number;
        latitude!: number;
        photos: File[] | undefined;
    }

    async function handleWaypointPhotoSelection() {
        const files = (
            document.getElementById("waypoint-photo-input") as HTMLInputElement
        ).files;

        if (!files) {
            return;
        }

        const liCoords: GPXCoord[] = [];

        let mergeRadius = 50;
        if ($formData.category) {
            for (const cat of $categories) {
                if ($formData.category !== cat.id) {
                    continue;
                }

                break;
            }
        }

        for (const file of files) {
            const coords = await new Promise<GPXCoord | undefined>((resolve) => {
                EXIF.getData(file, function (p) {
                    const lat = EXIF.getTag(p, "GPSLatitude");
                    const latDir = EXIF.getTag(p, "GPSLatitudeRef");
                    const lon = EXIF.getTag(p, "GPSLongitude");
                    const lonDir = EXIF.getTag(p, "GPSLongitudeRef");

                    if (lat && lon) {
                        var c = new GPXCoord();

                        c.latitude = convertDMSToDD(lat, latDir);
                        c.longitude = convertDMSToDD(lon, lonDir);
                    
                        resolve(c);
                    } else {
                        resolve(undefined);
                    }
                });
            });

            if (!coords) {
                show_toast(
                    {
                        type: "warning",
                        icon: "warning",
                        text: `${file.name}: ${$_("no-gps-data-in-image")}`,
                    },
                    10000,
                );
                continue;
            }

            var found = false;
            if (liCoords.length > 0) {
                for (let refCoords of liCoords) {
                    const distance = haversineDistance(refCoords.latitude, refCoords.longitude, coords.latitude, coords.longitude);

                    if (distance < mergeRadius) {
                        found = true;
                        if (!refCoords.photos) {
                            refCoords.photos = [];
                        }
                        refCoords.photos.push(file);
                        break;
            }
        }
            } 
            
            if (found === false) {
                coords.photos = [file];
                liCoords.push(coords);
            }
        }

        for (const coords of liCoords) {
            if (!coords.photos) {
                continue;
            }

            const wp: Waypoint = new Waypoint(coords.latitude, coords.longitude, {
                icon: coords.photos.length > 1 ? "images" : "image",
            });
            wp._photos = coords.photos;
            saveWaypoint(wp);
        }
    }

    async function undoRouteEdit() {
        undo();
        clearAnchors();
        syncAnchorListData();
        await initRouteAnchors(valhallaStore.route, true);
        refreshAnchorMetrics();
        updateTrailWithRouteData();
    }

    async function redoRouteEdit() {
        redo();
        clearAnchors();
        syncAnchorListData();
        await initRouteAnchors(valhallaStore.route, true);
        refreshAnchorMetrics();
        updateTrailWithRouteData();
    }

    function buildAnchorSignature(
        anchors: ValhallaAnchor[],
        options: RoutingOptions,
    ): string {
        // Keep this stable and cheap. We include routing options so changing mode/costing
        // forces a full reroute even if anchors didn't move.
        const opts = JSON.stringify(options ?? {});
        const pts = (anchors ?? [])
            .map((a) => `${a.id}:${a.lat.toFixed(6)},${a.lon.toFixed(6)}`)
            .join("|");
        return `${opts}::${pts}`;
    }

    async function rebuildRouteFromAnchors() {
        // Full rebuild (slow): used as fallback or when we can't do a minimal reroute.
        if (valhallaStore.anchors.length < 2) {
            clearRoute();
            updateTrailWithRouteData();
            return;
        }

        try {
            // reset existing route so we can build a new one
            clearRoute();

            for (let i = 0; i < valhallaStore.anchors.length - 1; i++) {
                const a = valhallaStore.anchors[i];
                const b = valhallaStore.anchors[i + 1];

                // calculate route segment between consecutive anchors
                const segment = await calculateRouteBetween(
                    a.lat,
                    a.lon,
                    b.lat,
                    b.lon,
                    routingOptions,
                );

                if (i === 0) {
                    // initialize route with first segment
                    // calculateRouteBetween returns an array of waypoints (trkpt) — wrap into a GPX object
                    const gpxSegment = new GPX({
                        trk: [
                            new Track({
                                trkseg: [
                                    new TrackSegment({
                                        trkpt: segment,
                                    }),
                                ],
                            }),
                        ],
                    });
                    setRoute(gpxSegment);
                } else {
                    // append following segments
                    insertIntoRoute(segment);
                }
            }

            // update anchor distances (uses current route positions)
            for (let i = 0; i < valhallaStore.anchors.length; i++) {
                getDistanceAndElevationGainLossFromPreviousAnchor(valhallaStore.anchors[i], i);
            }

            normalizeRouteTime();
            updateTrailWithRouteData();
        } catch (e) {
            console.error(e);
            show_toast({
                text: "Error recalculating route",
                icon: "close",
                type: "error",
            });
        }

        refreshAnchorMetrics();
    }

    async function rerouteAffectedSegmentsAfterReorder(
        prevAnchors: ValhallaAnchor[],
        nextAnchors: ValhallaAnchor[],
    ) {
        // Minimal reroute for a reorder:
        // Only segments adjacent to moved anchors can change.
        // We compute which indices changed by comparing anchor IDs.
        const prevIds = prevAnchors.map((a) => a.id);
        const nextIds = nextAnchors.map((a) => a.id);

        // If lengths differ, fallback to full rebuild.
        if (prevIds.length !== nextIds.length) {
            await rebuildRouteFromAnchors();
            return;
        }

        // Identify indices where the anchor ID differs.
        const changed: number[] = [];
        for (let i = 0; i < nextIds.length; i++) {
            if (prevIds[i] !== nextIds[i]) changed.push(i);
        }

        // No change
        if (changed.length === 0) {
            return;
        }

        // If too many changes, full rebuild is simpler/safer.
        // (Drag-and-drop can move multiple anchors; this threshold avoids complexity.)
        if (changed.length > 3) {
            await rebuildRouteFromAnchors();
            return;
        }

        // A moved anchor at index k affects segments (k-1) and k (between anchors).
        const segIndexes = new Set<number>();
        for (const k of changed) {
            segIndexes.add(k - 1);
            segIndexes.add(k);
        }

        // Filter to valid segment indices [0 .. anchors-2]
        const maxSeg = nextAnchors.length - 2;
        const segsToUpdate = Array.from(segIndexes)
            .filter((s) => s >= 0 && s <= maxSeg)
            .sort((a, b) => a - b);

        // If we can't patch in-place (route not initialized), rebuild.
        const existingSegCount =
            valhallaStore.route.trk?.at(0)?.trkseg?.length ?? 0;
        if (existingSegCount !== maxSeg + 1) {
            await rebuildRouteFromAnchors();
            return;
        }

        try {
            // Recompute only affected segments and patch them into the existing GPX route.
            for (const s of segsToUpdate) {
                const a = nextAnchors[s];
                const b = nextAnchors[s + 1];

                const segment = await calculateRouteBetween(
                    a.lat,
                    a.lon,
                    b.lat,
                    b.lon,
                    routingOptions,
                );

                await editRoute(s, segment);
            }

            // Update anchor distances/elevation gain/loss after patching.
            for (let i = 0; i < nextAnchors.length; i++) {
                getDistanceAndElevationGainLossFromPreviousAnchor(
                    nextAnchors[i],
                    i,
                );
            }

            normalizeRouteTime();
            updateTrailWithRouteData();
        } catch (e) {
            console.error(e);
            show_toast({
                text: "Error recalculating affected segments",
                icon: "close",
                type: "error",
            });

            // Fallback to full rebuild to recover.
            await rebuildRouteFromAnchors();
        }

        refreshAnchorMetrics();
    }

    function updateAnchorIndices() {
        for (let i = 0; i < valhallaStore.anchors.length; i++) {
            const anchor = valhallaStore.anchors[i];
            const markerIcon = anchor.marker?.getElement();
            // don't overwrite a spinner state - keep loading indicator if present
            if (markerIcon && !markerIcon.classList.contains("spinner")) {
                markerIcon.textContent = (i + 1).toString();
            }
            // update popup header if present
            const popup = anchor.marker?.getPopup();
            if (popup && popup._content) {
                const h5 = popup._content.getElementsByTagName("h5")[0];
                if (h5) {
                    h5.textContent = $_("route-point") + " #" + (i + 1);
                }
            }
        }
    }

	async function handleAnchorDrop(newItems: ValhallaAnchor[]) {
        
        loadEditing = true;
        await tick();
        await new Promise((resolve) => setTimeout(resolve, 0));

        const prevAnchors = [...(valhallaStore.anchors ?? [])];
        const prevSignature = buildAnchorSignature(prevAnchors, routingOptions);

		// remove any existing markers from the map first
		for (const a of valhallaStore.anchors ?? []) {
			try {
				a.marker?.remove();
			} catch (e) {
				/* ignore */
			}
		}

		valhallaStore.anchors = newItems;

        // update marker labels / popups to reflect new order immediately
        updateAnchorIndices();

		// assign new anchor list (shallow copy of items to avoid unexpected refs)
		valhallaStore.anchors = (newItems ?? []).map((a: any) => ({ ...a }));

        // recreate and attach markers for the new anchor order so they are wired correctly
        for (let i = 0; i < valhallaStore.anchors.length; i++) {
            const anchor = valhallaStore.anchors[i];

            // remove any previous marker just in case
            anchor.marker?.remove();

            const marker = createAnchorMarker(
                anchor.lat,
                anchor.lon,
                i + 1,
                () => {
                    // delete handler
                    removeAnchor(
                        valhallaStore.anchors.findIndex(
                            (x) => x.id == anchor.id,
                        ),
                    );
                },
                () => {
                    // recalc / snap handler
                    const thisAnchor = valhallaStore.anchors.find(
                        (a) => a.id == anchor.id,
                    );
                    addAnchorAndRecalculate(
                        thisAnchor?.lat ?? anchor.lat,
                        thisAnchor?.lon ?? anchor.lon,
                    );
                },
                () => {
                    draggingMarker = true;
                },
                async () => {
                    if (!drawingActive) return;
                    const idx = valhallaStore.anchors.findIndex(
                        (x) => x.id == anchor.id,
                    );
                    const position = marker.getLngLat();
                    updateAnchorAt(idx, (existing) => ({
                        ...existing,
                        lat: position.lat,
                        lon: position.lng,
                    }));
                    await recalculateRoute(idx);
                    draggingMarker = false;
                },
            );

            if (map) marker.addTo(map);
            anchor.marker = marker;
        }

        syncAnchorListData();

        const nextAnchors = [...(valhallaStore.anchors ?? [])];
        const nextSignature = buildAnchorSignature(nextAnchors, routingOptions);

        // Force a map/form refresh after reorder regardless of whether we reroute.
        // This avoids stale map rendering / stale saved GPX when the reorder results
        // in a no-op reroute path (e.g. moved to end) but the underlying order changed.
        overwriteGPX = true;

        // If only the order changed, reroute only affected adjacent segments.
        // Otherwise (anchor moved/edited/options changed), fallback to full rebuild.
        if (prevSignature !== nextSignature) {
            await rerouteAffectedSegmentsAfterReorder(prevAnchors, nextAnchors);
        }

        // Always refresh map + totals after reordering anchors.
        // (Even if rerouteAffectedSegmentsAfterReorder() early-returns.)
        updateTrailWithRouteData();

        loadEditing = false;
    }
</script>

<svelte:head>
    <title
        >{page.params.id !== "new"
            ? `${$formData.name} | ${$_("edit")}`
            : $_("new-trail")} | wanderer</title
    >
</svelte:head>

<main class="grid grid-cols-1 md:grid-cols-[400px_1fr]">
    <form
        id="trail-form"
        class="overflow-y-auto overflow-x-hidden flex flex-col gap-4 px-8 order-1 md:order-none mt-8 md:mt-0"
        use:form
    >
        <Search
            onupdate={(q) => searchCities(q)}
            onclick={(item) => handleSearchClick(item)}
            placeholder="{$_('search-places')}..."
            items={searchDropdownItems}
        ></Search>
        {#if selectedSearchLocation && drawingActive}
            <div
                class="rounded-xl border border-input-border bg-menu-item-background px-4 py-3 flex flex-col gap-3"
            >
                <div class="flex items-start gap-3">
                    <button
                        type="button"
                        class="btn-primary shrink-0 rounded-full w-10 h-10 flex items-center justify-center !p-0 disabled:opacity-40 disabled:cursor-not-allowed"
                        aria-label={$_("use-as-route-anchor")}
                        title={$_("use-as-route-anchor")}
                        disabled={!drawingActive || loadEditing}
                        onclick={useSelectedLocationAsAnchor}
                    >
                        {#if loadEditing && drawingActive}
                            <span class="spinner spinner-small"></span>
                        {:else}
                            <i class="fa fa-route text-lg"></i>
                        {/if}
                    </button>
                    <div class="flex-1">
                        <p
                            class="text-xs font-semibold uppercase tracking-wide text-gray-500"
                        >
                            {$_("selected-place")}
                        </p>
                        <p class="font-semibold">
                            {selectedSearchLocation.text}
                        </p>
                        {#if selectedSearchLocation.description}
                            <p class="text-sm text-gray-500">
                                {selectedSearchLocation.description}
                            </p>
                        {/if}
                    </div>
                    <button
                        type="button"
                        class="btn-icon"
                        aria-label={$_("clear-all")}
                        onclick={clearSelectedSearchLocation}
                    >
                        <i class="fa fa-close text-sm"></i>
                    </button>
                </div>
                {#if !drawingActive}
                    <p class="text-sm text-gray-500 text-right">
                        {$_("start-route-editing-to-add-anchor")}
                    </p>
                {/if}
            </div>
        {/if}
        <hr class="border-input-border" />
        {#if !drawingActive}
        <h3 class="text-xl font-semibold">{$_("pick-a-trail")}</h3>
        <Button
            primary={true}
            type="button"
            disabled={drawingActive}
            onclick={openFileBrowser}
            >{$formData.expand?.gpx_data
                ? $_("upload-new-file")
                : $_("upload-file")}</Button
        >
        {/if}
        {#if env.PUBLIC_VALHALLA_URL}
            {#if !drawingActive}
            <div class="flex gap-4 items-center w-full">
                <hr class="basis-full border-input-border" />
                <span class="text-gray-500 uppercase">{$_("or")}</span>
                <hr class="basis-full border-input-border" />
            </div>
            {/if}
            <Button
                primary
                type="button"
                onclick={async () => {
                    loadEditing = true;

                    // allow the spinner to paint before heavy work
                    await new Promise((r) => setTimeout(r, 0));

                    try {
                    if (drawingActive) {
                        await stopDrawing();
                    } else {
                            await startDrawing();
                    }
                    } finally {
                        loadEditing = false;
                    }
                }}                
                loading={loadEditing}
            >
                {$formData.expand?.gpx_data
                    ? drawingActive
                        ? $_("stop-editing")
                        : $_("edit-route")
                    : drawingActive
                      ? $_("stop-drawing")
                      : $_("draw-a-route")}</Button
            >
            {#if drawingActive}
                <div class={loadEditing ? "relative pointer-events-none opacity-50" : "relative"}>
                    <TrailAnchorList itemsData={listData} onDrop={handleAnchorDrop}></TrailAnchorList>
                </div>
            {/if}    
        {/if}
        <input
            type="file"
            name="gpx"
            id="fileInput"
            accept=".gpx,.GPX,.tcx,.TCX,.kml,.KML,.kmz,.KMZ,.fit,.FIT"
            style="display: none;"
            onchange={handleFileSelection}
        />
        <hr class="border-separator" />
        <div class="flex gap-x-2">
            <h3 class="text-xl font-semibold">{$_("basic-info")}</h3>
            <button
                aria-label="Edit basic info"
                type="button"
                class="btn-icon"
                style="font-size: 0.9rem"
                onclick={() => (editingBasicInfo = !editingBasicInfo)}
                ><i class="fa fa-{editingBasicInfo ? 'check' : 'pen'}"
                ></i></button
            >
        </div>

        <fieldset
            class="grid grid-cols-2 gap-4 justify-around"
            data-felte-keep-on-remove
        >
            {#if editingBasicInfo}
                <TextField
                    bind:value={$formData.distance}
                    name="distance"
                    label={$_("distance")}
                ></TextField>
                <TextField
                    bind:value={$formData.duration}
                    name="duration"
                    label={$_("est-duration")}
                ></TextField><TextField
                    bind:value={$formData.elevation_gain}
                    name="elevation_gain"
                    label={$_("elevation-gain")}
                ></TextField>
                <TextField
                    bind:value={$formData.elevation_loss}
                    name="elevation_loss"
                    label={$_("elevation-loss")}
                ></TextField>
            {:else}
                <div>
                    <p>{$_("distance")}</p>
                    <span class="font-medium"
                        >{formatDistance($formData.distance)}</span
                    >
                    <input
                        type="hidden"
                        name="distance"
                        value={$formData.distance}
                    />
                </div>
                <div>
                    <p>{$_("est-duration")}</p>
                    <span class="font-medium"
                        >{formatTimeHHMM($formData.duration)}</span
                    >
                    <input
                        type="hidden"
                        name="duration"
                        value={$formData.duration}
                    />
                </div>
                <div>
                    <p>{$_("elevation-gain")}</p>
                    <span class="font-medium"
                        >{formatElevation($formData.elevation_gain)}</span
                    >
                    <input
                        type="hidden"
                        name="elevation_gain"
                        value={$formData.elevation_gain}
                    />
                </div>
                <div>
                    <p>{$_("elevation-loss")}</p>
                    <span class="font-medium"
                        >{formatElevation($formData.elevation_loss)}</span
                    >
                    <input
                        type="hidden"
                        name="elevation_gain"
                        value={$formData.elevation_gain}
                    />
                </div>
            {/if}
        </fieldset>
        <TextField name="name" label={$_("name")} error={$errors.name}
        ></TextField>
        <TextField
            name="location"
            label={$_("location")}
            error={$errors.location}
        ></TextField>
        <Datepicker label={$_("date")} bind:value={$formData.date}></Datepicker>
        <Editor
            extraClasses="min-h-24"
            bind:value={$formData.description}
            label={$_("describe-your-trail")}
        ></Editor>
        <Combobox
            bind:value={getTrailTags, setTrailTags}
            onupdate={searchTags}
            items={tagItems}
            label={$_("tags")}
            multiple
            chips
        ></Combobox>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-y-4">
            <Select
                name="difficulty"
                label={$_("difficulty")}
                items={[
                    { text: $_("easy"), value: "easy" },
                    { text: $_("moderate"), value: "moderate" },
                    { text: $_("difficult"), value: "difficult" },
                ]}
            ></Select>
            <Select
                name="category"
                label={$_("category")}
                items={$categories.map((c) => ({
                    text: $_(c.name),
                    value: c.id,
                }))}
            ></Select>
        </div>

        <Toggle
            name="public"
            label={$formData.public ? $_("public") : $_("private")}
            icon={$formData.public ? "globe" : "lock"}
        ></Toggle>
        <hr class="border-separator" />
        <h3 class="text-xl font-semibold">
            {$_("waypoints", { values: { n: 2 } })}
        </h3>
        <ul>
            {#each $formData.expand?.waypoints_via_trail ?? [] as waypoint, i}
                <li
                    onmouseenter={() => openMarkerPopup(waypoint)}
                    onmouseleave={() => openMarkerPopup(waypoint)}
                >
                    <WaypointCard
                        {waypoint}
                        mode="edit"
                        onchange={(item) =>
                            handleWaypointMenuClick(waypoint, i, item)}
                    ></WaypointCard>
                </li>
            {/each}
        </ul>
        <button
            class="btn-secondary"
            type="button"
            onclick={() => beforeWaypointModalOpen()}
            ><i class="fa fa-plus mr-2"></i>{$_("add-waypoint")}</button
        >
        <button
            class="btn-secondary"
            type="button"
            onclick={() => openPhotoBrowser()}
            ><i class="fa fa-image mr-2"></i>{$_("from-photos")}</button
        >
        <input
            type="file"
            id="waypoint-photo-input"
            accept="image/*"
            multiple={true}
            style="display: none;"
            onchange={() => handleWaypointPhotoSelection()}
        />
        <hr class="border-separator" />
        <h3 class="text-xl font-semibold">{$_("photos")}</h3>
        <PhotoPicker
            id="trail"
            parent={$formData}
            bind:photos={$formData.photos}
            bind:thumbnail={$formData.thumbnail}
            bind:photoFiles
        ></PhotoPicker>
        <hr class="border-separator" />
        <h3 class="text-xl font-semibold">{$_("summit-book")}</h3>
        <ul>
            {#each $formData.expand?.summit_logs_via_trail ?? [] as log, i}
                <li>
                    <SummitLogCard
                        {log}
                        mode={log.author == $currentUser?.actor
                            ? "edit"
                            : "show"}
                        onchange={(item) =>
                            handleSummitLogMenuClick(log, i, item)}
                    ></SummitLogCard>
                </li>
            {/each}
        </ul>
        <button
            class="btn-secondary"
            type="button"
            onclick={beforeSummitLogModalOpen}
            ><i class="fa fa-plus mr-2"></i>{$_("add-entry")}</button
        >
        {#if lists.items.length}
            <hr class="border-separator" />
            <h3 class="text-xl font-semibold">
                {$_("list", { values: { n: 2 } })}
            </h3>
            <div class="flex gap-4 flex-wrap">
                {#each lists.items as list}
                    {#if $formData.id && list.trails?.includes($formData.id)}
                        <div
                            class="flex gap-2 items-center border border-input-border rounded-xl p-2"
                        >
                            <img
                                class="w-8 aspect-square rounded-full object-cover"
                                src={list.avatar
                                    ? getFileURL(list, list.avatar)
                                    : $theme === "light"
                                      ? emptyStateTrailLight
                                      : emptyStateTrailDark}
                                alt="avatar"
                            />

                            <span class="text-sm">{list.name}</span>
                        </div>
                    {/if}
                {/each}
            </div>
            <Button
                secondary={true}
                tooltip={$_("save-your-trail-first")}
                disabled={page.params.id == "new" && !savedAtLeastOnce}
                type="button"
                onclick={() => listSelectModal.openModal()}
                ><i class="fa fa-plus mr-2"></i>{$_("add-to-list")}</Button
            >
        {/if}
        <hr class="border-separator" />
        <Button
            primary={true}
            large={true}
            type="submit"
            extraClasses="mb-2"
            {loading}>{$_("save-trail")}</Button
        >
    </form>
    <div class="relative">
        {#if drawingActive}
            <div
                in:fly={{ easing: backInOut, x: -30 }}
                out:fly={{ easing: backInOut, x: -30 }}
                class="absolute top-8 left-2 z-50"
            >
                <RouteEditor
                    bind:options={routingOptions}
                    onReverse={reverseTrail}
                    onReset={resetTrail}
                    onCropToggle={toggleCropMarkers}
                    onCrop={confirmCrop}
                    onUpdateCropRange={updateCropMarkers}
                    onRecalculateElevationData={recalculateElevationData}
                    onUndo={undoRouteEdit}
                    onRedo={redoRouteEdit}
                ></RouteEditor>
            </div>
        {/if}
        <div id="trail-map">
            <MapWithElevationMaplibre
                trails={mapTrail}
                waypoints={$formData.expand?.waypoints_via_trail}
                drawing={drawingActive}
                showTerrain={true}
                autoGeolocateOnDrawing={page.params.id === "new"}
                onmarkerdragend={moveMarker}
                activeTrail={0}
                bind:map
                onclick={(target) => handleMapClick(target)}
                onsegmentclick={(data) => handleSegmentClick(data)}
                onsegmentdragend={(data) => handleSegmentDragEnd(data)}
                mapOptions={{
                    preserveDrawingBuffer: true,
                }}
                {buildPoiAnchorAction}
            ></MapWithElevationMaplibre>
        </div>
    </div>
</main>
<WaypointModal bind:this={waypointModal} onsave={saveWaypoint}></WaypointModal>
<SummitLogModal bind:this={summitLogModal} onsave={(log) => saveSummitLog(log)}
></SummitLogModal>
<ListSelectModal
    lists={lists.items}
    bind:this={listSelectModal}
    onchange={(e) => handleListSelection(e)}
></ListSelectModal>

<style>
    #trail-map {
        height: calc(50vh);
    }
    @media only screen and (min-width: 768px) {
        #trail-map,
        form {
            height: calc(100vh - 124px);
        }
    }
</style>
