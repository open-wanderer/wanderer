<script lang="ts">
    import { env } from "$env/dynamic/public";
    import Button from "$lib/components/base/button.svelte";
    import Datepicker from "$lib/components/base/datepicker.svelte";
    import Select from "$lib/components/base/select.svelte";
    import TextField from "$lib/components/base/text_field.svelte";
    import Toggle from "$lib/components/base/toggle.svelte";
    import ListSearchModal from "$lib/components/list/list_search_modal.svelte";
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
    import { convertDMSToDD } from "$lib/models/gpx/utils.js";
    import { Tag } from "$lib/models/tag.js";
    import {
        searchLocationReverse,
        searchLocations,
    } from "$lib/stores/search_store.js";
    import { tags_index } from "$lib/stores/tag_store.js";
    import { theme } from "$lib/stores/theme_store.js";
    import { currentUser } from "$lib/stores/user_store.js";
    import { getIconForLocation } from "$lib/util/icon_util.js";
    import {
        createAnchorMarker,
        createEditTrailMapPopup,
        FontawesomeMarker,
    } from "$lib/util/maplibre_util";
    import EXIF from "$lib/vendor/exif-js/exif.js";
    import { validator } from "@felte/validator-zod";
    import cryptoRandomString from "crypto-random-string";
    import { createForm } from "felte";
    import * as M from "maplibre-gl";
    import { onMount, tick, untrack } from "svelte";
    import { _ } from "svelte-i18n";
    import { backInOut } from "svelte/easing";
    import { fly, slide } from "svelte/transition";
    import { z } from "zod";
    import Track from "$lib/models/gpx/track.js";
    import TrackSegment from "$lib/models/gpx/track-segment.js";

    let { data } = $props();

    let map: M.Map | undefined = $state();
    let mapPopup: M.Popup | undefined;
    let mapTrail: Trail[] = $state([]);
    let lists = $state(data.lists);

    let waypointModal: WaypointModal;
    let summitLogModal: SummitLogModal;
    let listSelectModal: ListSearchModal;

    let loading = $state(false);

    let editingBasicInfo: boolean = $state(false);

    let photoFiles: File[] = $state([]);

    let gpxFile: File | Blob | null = null;

    let drawingActive = $state(false);
    let overwriteGPX = false;
    let draggingMarker = false;

    let searchDropdownItems: SearchItem[] = $state([]);

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
        modeOfTransport: "pedestrian",
    });

    let savedAtLeastOnce = $state(false);

    let tagItems: ComboboxItem[] = $state([]);

    const {
        form,
        errors,
        data: formData,
        setFields,
    } = createForm<z.infer<typeof ClientTrailCreateSchema>>({
        initialValues: {
            ...data.trail,
            public: data.trail.id
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
                    valhallaStore.route.trk?.at(0)?.trkseg?.at(0)?.trkpt?.at(0)
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
    });

    onMount(async () => {
        clearAnchors();
        clearRoute();
        clearUndoRedoStack();

        if ($formData.expand!.gpx_data) {
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
                initRouteAnchors(gpx);

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
            initRouteAnchors(parseResult.gpx);

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

    function initRouteAnchors(gpx: GPX, addToMap: boolean = false) {
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
            const index = lists.items.findIndex((l) => l.id == list.id);
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

    function startDrawing() {
        if (!map) {
            return;
        }
        drawingActive = true;
        if (!valhallaStore.route.trk?.at(0)?.trkseg?.at(0)?.trkpt?.length) {
        }
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
                await addAnchorAndRecalculate(e.lngLat.lat, e.lngLat.lng);
            }
        }
    }

    async function addAnchorAndRecalculate(lat: number, lon: number) {
        const previousAnchor =
            valhallaStore.anchors[valhallaStore.anchors.length - 1];
        const anchor = addAnchor(lat, lon, valhallaStore.anchors.length);
        const markerText = startAnchorLoading(anchor);
        try {
            const routeWaypoints = await calculateRouteBetween(
                previousAnchor.lat,
                previousAnchor.lon,
                lat,
                lon,
                routingOptions,
            );
            insertIntoRoute(routeWaypoints);
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
            stopAnchorLoading(anchor, markerText);
        }
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
                const anchorIndex = valhallaStore.anchors.findIndex(
                    (a) => a.id == anchor.id,
                );
                const thisAnchor = valhallaStore.anchors[anchorIndex];
                const position = marker.getLngLat();
                thisAnchor.lat = position.lat;
                thisAnchor.lon = position.lng;

                await recalculateRoute(anchorIndex);

                draggingMarker = false;
            },
        );
        if (addtoMap && map) {
            marker.addTo(map);
        }
        anchor.marker = marker;
        valhallaStore.anchors.splice(index, 0, anchor);

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

    function stopAnchorLoading(anchor: ValhallaAnchor, index: string | null) {
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
    }

    async function removeAnchor(anchorIndex: number) {
        if (!drawingActive) {
            return;
        }
        valhallaStore.anchors[anchorIndex]?.marker?.remove();
        valhallaStore.anchors.splice(anchorIndex, 1);
        for (let i = anchorIndex; i < valhallaStore.anchors.length; i++) {
            const anchor = valhallaStore.anchors[i];
            const markerIcon = anchor.marker?.getElement();
            if (markerIcon) {
                const markerText = markerIcon.textContent ?? "0";
                const markerIndex = parseInt(markerText);
                const newIndex = markerIndex - 1;
                markerIcon.textContent = newIndex + "";
                anchor
                    .marker!.getPopup()
                    ._content.getElementsByTagName("h5")[0].textContent =
                    $_("route-point") + " #" + newIndex;
            }
        }
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
        const anchor = addAnchor(
            data.event.lngLat.lat,
            data.event.lngLat.lng,
            data.segment + 1,
        );
        const markerText = startAnchorLoading(anchor);
        updateFollowingAnchors(data.segment);

        const previousAnchor = valhallaStore.anchors[data.segment];
        const nextAnchor = valhallaStore.anchors[data.segment + 2];

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

            editRoute(data.segment, previousRouteSegment);
            insertIntoRoute(nextRouteSegment, data.segment + 1);
            normalizeRouteTime();
            updateTrailWithRouteData();
        } catch (e) {
            console.error(e);
            show_toast({
                text: "Error calculating route",
                icon: "close",
                type: "error",
            });
        } finally {
            stopAnchorLoading(anchor, markerText);
        }
    }

    function updateFollowingAnchors(segment: number) {
        for (let i = segment + 2; i < valhallaStore.anchors.length; i++) {
            const anchor = valhallaStore.anchors[i];
            const markerIcon = anchor.marker?.getElement();
            if (markerIcon) {
                const markerText = markerIcon.textContent ?? "0";
                const markerIndex = parseInt(markerText);
                const newIndex = markerIndex + 1;
                markerIcon.textContent = newIndex + "";
                anchor
                    .marker!.getPopup()
                    ._content.getElementsByTagName("h5")[0].textContent =
                    $_("route-point") + " #" + newIndex;
            }
        }
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
    }

    function resetTrail() {
        resetRoute();

        updateTrailWithRouteData();
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

    function confirmCrop() {
        if (!croppedGPX) {
            return;
        }
        setRoute(croppedGPX, true);
        updateTrailWithRouteData();
        clearAnchors();
        initRouteAnchors(croppedGPX, true);
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
        const t: Trail = JSON.parse(JSON.stringify($formData));
        t.expand!.gpx = valhallaStore.route;
        mapTrail = [t];
    }

    function handleSearchClick(item: SearchItem) {
        map?.flyTo({
            center: [item.value.lon, item.value.lat],
            zoom: 13,
            animate: false,
        });
    }

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

    async function handleWaypointPhotoSelection() {
        const files = (
            document.getElementById("waypoint-photo-input") as HTMLInputElement
        ).files;

        if (!files) {
            return;
        }

        for (const file of files) {
            const coords = await new Promise<number[]>((resolve) => {
                EXIF.getData(file, function (p) {
                    const lat = EXIF.getTag(p, "GPSLatitude");
                    const latDir = EXIF.getTag(p, "GPSLatitudeRef");
                    const lon = EXIF.getTag(p, "GPSLongitude");
                    const lonDir = EXIF.getTag(p, "GPSLongitudeRef");

                    if (lat && lon) {
                        resolve([
                            convertDMSToDD(lat, latDir),
                            convertDMSToDD(lon, lonDir),
                        ]);
                    } else {
                        resolve([]);
                    }
                });
            });
            if (coords.length) {
                const wp: Waypoint = new Waypoint(coords[0], coords[1], {
                    icon: "image",
                });
                wp._photos = [file];
                saveWaypoint(wp);
            } else {
                show_toast(
                    {
                        type: "warning",
                        icon: "warning",
                        text: `${file.name}: ${$_("no-gps-data-in-image")}`,
                    },
                    10000,
                );
            }
        }
    }

    function undoRouteEdit() {
        undo();
        clearAnchors();
        initRouteAnchors(valhallaStore.route, true);
        updateTrailWithRouteData();
    }

    function redoRouteEdit() {
        redo();
        clearAnchors();
        initRouteAnchors(valhallaStore.route, true);
        updateTrailWithRouteData();
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
        <hr class="border-input-border" />
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
        {#if env.PUBLIC_VALHALLA_URL}
            <div class="flex gap-4 items-center w-full">
                <hr class="basis-full border-input-border" />
                <span class="text-gray-500 uppercase">{$_("or")}</span>
                <hr class="basis-full border-input-border" />
            </div>
            <button
                class="btn-primary"
                type="button"
                onclick={async () => {
                    if (drawingActive) {
                        await stopDrawing();
                    } else {
                        startDrawing();
                    }
                }}
            >
                {$formData.expand?.gpx_data
                    ? drawingActive
                        ? $_("stop-editing")
                        : $_("edit-route")
                    : drawingActive
                      ? $_("stop-drawing")
                      : $_("draw-a-route")}</button
            >
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
                onmarkerdragend={moveMarker}
                activeTrail={0}
                bind:map
                onclick={(target) => handleMapClick(target)}
                onsegmentclick={(data) => handleSegmentClick(data)}
                onsegmentdragend={(data) => handleSegmentDragEnd(data)}
                mapOptions={{ preserveDrawingBuffer: true }}
            ></MapWithElevationMaplibre>
        </div>
    </div>
</main>
<WaypointModal bind:this={waypointModal} onsave={saveWaypoint}></WaypointModal>
<SummitLogModal bind:this={summitLogModal} onsave={(log) => saveSummitLog(log)}
></SummitLogModal>
<ListSearchModal
    lists={lists.items}
    bind:this={listSelectModal}
    onchange={(e) => handleListSelection(e)}
></ListSearchModal>

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
