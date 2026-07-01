<script lang="ts">
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
    import TrailAnchorList from "$lib/components/trail/trail_anchor_list.svelte";
    import WaypointCard from "$lib/components/waypoint/waypoint_card.svelte";
    import WaypointMergeModal, {
        type WaypointMergeOptions,
    } from "$lib/components/waypoint/waypoint_merge_modal.svelte";
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
    import type { OverpassPopupActionFactory } from "$lib/vendor/maplibre-layer-manager/overpass-layer";
    import { type OverpassPopupAction } from "$lib/util/maplibre_util";
    import { Waypoint } from "$lib/models/waypoint";
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
        revertRouteChange,
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
    import { designSelectableCategories } from "$lib/util/category_util";
    import { getIconForLocation } from "$lib/util/icon_util.js";
    import {
        createAnchorMarker,
        createEditTrailMapPopup,
        FontawesomeMarker,
    } from "$lib/util/maplibre_util";
    import {
        renderValhallaAnchorMarker,
        valhallaAnchorTitle,
    } from "$lib/util/valhalla_anchor_util";
    import EXIF from "$lib/vendor/exif-js/exif.js";
    import { validator } from "@felte/validator-zod";
    import cryptoRandomString from "crypto-random-string";
    import { createForm } from "felte";
    import * as M from "maplibre-gl";
    import { onMount, untrack } from "svelte";
    import { _, locale } from "svelte-i18n";
    import { backInOut } from "svelte/easing";
    import { fly } from "svelte/transition";
    import { z } from "zod";
    import Track from "$lib/models/gpx/track.js";
    import TrackSegment from "$lib/models/gpx/track-segment.js";
    import ConfirmModal from "$lib/components/confirm_modal.svelte";
    import CategoryPicker from "$lib/components/trail/category_picker.svelte";

    let { data } = $props();

    let map: M.Map | undefined = $state();
    let mapPopup: M.Popup | undefined;
    let mapTrail: Trail[] = $state([]);
    let lists = $state(untrack(() => data.lists));

    let waypointModal: WaypointModal;
    let waypointMergeModal: WaypointMergeModal;
    let summitLogModal: SummitLogModal;
    let listSelectModal: ListSearchModal;
    let markTrailAsCompletedModal: ConfirmModal;
    let replaceRouteModal: ConfirmModal;

    let loading = $state(false);

    let editingBasicInfo: boolean = $state(false);

    let photoFiles: File[] = $state([]);

    let gpxFile: File | Blob | null = null;

    let drawingActive = $state(false);
    let showWaypointsWhileDrawing = $state(true);
    let replacingRoute = $state(false);
    let isNewTrail = $derived(page.params.id === "new");

    function routeCalculationErrorText(error: unknown) {
        if (error instanceof Error && error.message) {
            return error.message;
        }
        return "Error calculating route";
    }
    let overwriteGPX = false;
    let draggingMarker = false;
    
    let pendingWaypointMerge:
        | { incoming: Waypoint; existing: Waypoint }
        | undefined = $state();

    let searchDropdownItems: SearchItem[] = $state([]);
    let selectedSearchLocation: SearchItem | null = $state(null);
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
    let routeAnchorListUpdating = $state(false);
    let routeSegments = $state<TrackSegment[]>([]);

    let savedAtLeastOnce = $state(false);

    let tagItems: ComboboxItem[] = $state([]);

    function defaultCategoryId() {
        const existingCategory = data.trail.category;
        if (existingCategory) {
            return existingCategory;
        }

        // Pre-select the highest-priority visible category for new trails.
        return (
            designSelectableCategories(
                data.categories,
                data.categoryPreferences,
                $locale,
            )[0]?.id ?? data.categories[0]?.id ?? ""
        );
    }

    const getInitialFormValues = () => ({
        ...data.trail,
        public: data.trail.id
            ? data.trail.public
            : page.data.settings?.privacy?.trails === "public",
        category: defaultCategoryId(),
        subcategory: data.trail.subcategory || "",
    });

    const {
        form,
        errors,
        data: formData,
        setFields,
    } = createForm<z.infer<typeof ClientTrailCreateSchema>>({
        initialValues: getInitialFormValues(),
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

    let categorySelectValue = $derived(
        $formData.subcategory
            ? `subcategory:${$formData.subcategory}`
            : $formData.category
              ? `category:${$formData.category}`
              : "",
    );

    function handleCategoryChange(selection: {
        category: string;
        subcategory: string;
    }) {
        setFields("category", selection.category);
        setFields("subcategory", selection.subcategory);
    }

    onMount(async () => {
        clearAnchors();
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
                initRouteAnchors(gpx);

                updateTrailOnMap();

                if (!isNewTrail) {
                    startDrawing();
                }
            }
        }
    });

    function fitCurrentRoute(initializedMap: M.Map) {
        const bounds = valhallaStore.route.toGeoJSON().bbox;
        if (!bounds) {
            return;
        }

        initializedMap.fitBounds(bounds as M.LngLatBoundsLike, {
            animate: false,
            padding: {
                top: 16,
                left: 16,
                right: 16,
                bottom: 16,
            },
        });
    }

    function handleMapInit(initializedMap: M.Map) {
        if (drawingActive) {
            for (const anchor of valhallaStore.anchors) {
                anchor.marker?.addTo(initializedMap);
            }
        }
        if (!isNewTrail) {
            fitCurrentRoute(initializedMap);
        }
    }

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

        const replaceExistingRoute = replacingRoute && !isNewTrail;
        if (!replaceExistingRoute) {
            clearWaypoints();
        }
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
            if (replaceExistingRoute) {
                setFields("lat", parseResult.trail.lat);
                setFields("lon", parseResult.trail.lon);
                setFields("distance", parseResult.trail.distance);
                setFields("duration", parseResult.trail.duration);
                setFields("elevation_gain", parseResult.trail.elevation_gain);
                setFields("elevation_loss", parseResult.trail.elevation_loss);
            } else {
                setFields(parseResult.trail);
            }
            $formData.id = prevId ?? cryptoRandomString({ length: 15 });
            $formData.expand!.gpx_data = gpxData;

            if (!replaceExistingRoute) {
                setFields(
                    "category",
                    defaultCategoryId(),
                );
                setFields("subcategory", "");
                setFields(
                    "public",
                    page.data.settings?.privacy?.trails === "public",
                );
            }

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
            replacingRoute = false;
            if (!isNewTrail) {
                startDrawing();
                if (map) {
                    fitCurrentRoute(map);
                }
            }

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
        $formData.expand!.waypoints_via_trail =
            $formData.expand!.waypoints_via_trail;

        // updateTrailOnMap();
    }

    function commitWaypoint(savedWaypoint: Waypoint) {
        let editedWaypointIndex =
            $formData.expand!.waypoints_via_trail?.findIndex(
                (s) => s.id == savedWaypoint.id,
            ) ?? -1;

        if (editedWaypointIndex >= 0) {
            $formData.expand!.waypoints_via_trail![editedWaypointIndex] =
                savedWaypoint;
        } else {
            savedWaypoint.id = cryptoRandomString({ length: 15 });
            $formData.expand!.waypoints_via_trail = [
                ...($formData.expand!.waypoints_via_trail ?? []),
                savedWaypoint,
            ];

            // updateTrailOnMap();
        }
    }

    function getExistingWaypointClusterInputs() {
        return (
            $formData.expand?.waypoints_via_trail
                ?.filter((wp) => wp.id)
                .map((wp) => ({
                    id: wp.id!,
                    lat: wp.lat,
                    lon: wp.lon,
                })) ?? []
        );
    }

    async function saveWaypoint(savedWaypoint: Waypoint) {
        const editedWaypointIndex =
            $formData.expand!.waypoints_via_trail?.findIndex(
                (s) => s.id == savedWaypoint.id,
            ) ?? -1;

        if (editedWaypointIndex >= 0) {
            commitWaypoint(savedWaypoint);
            return true;
        }

        const matchingWaypoint = await findMergeableWaypoint(savedWaypoint);
        if (matchingWaypoint) {
            pendingWaypointMerge = {
                incoming: savedWaypoint,
                existing: matchingWaypoint,
            };
            waypointModal.closeModal();
            waypointMergeModal.openModal();
            return false;
        }

        commitWaypoint(savedWaypoint);
        return true;
    }

    async function findMergeableWaypoint(savedWaypoint: Waypoint) {
        const existingWaypoints = getExistingWaypointClusterInputs();

        if (!existingWaypoints.length) {
            return;
        }

        try {
            const clusterResponse = await clusterWaypointPhotos({
                category: $formData.category,
                photos: [
                    {
                        id: waypointMergeCheckPhotoId,
                        lat: savedWaypoint.lat,
                        lon: savedWaypoint.lon,
                    },
                ],
                waypoints: existingWaypoints,
            });

            const matchingCluster = clusterResponse.clusters.find(
                (cluster) =>
                    cluster.waypoint &&
                    cluster.photos.includes(waypointMergeCheckPhotoId),
            );

            if (!matchingCluster?.waypoint) {
                return;
            }

            return $formData.expand?.waypoints_via_trail?.find(
                (wp) => wp.id === matchingCluster.waypoint,
            );
        } catch (e) {
            show_toast(
                {
                    type: "error",
                    icon: "warning",
                    text: $_("waypoint-cluster-error"),
                },
                10000,
            );
        }
    }

    function createPendingWaypointAnyway() {
        if (!pendingWaypointMerge) {
            return;
        }

        commitWaypoint(pendingWaypointMerge.incoming);
        closeWaypointMergeModal();
    }

    function addPendingWaypointToExisting(options: WaypointMergeOptions) {
        if (!pendingWaypointMerge) {
            return;
        }

        const { incoming, existing } = pendingWaypointMerge;
        const mergedWaypoint = {
            ...existing,
            icon: options.icon ? incoming.icon : existing.icon,
            name: options.title
                ? appendDistinctText(existing.name, incoming.name, " / ")
                : existing.name,
            description: options.description
                ? appendDistinctText(
                      existing.description,
                      incoming.description,
                      "\n\n",
                  )
                : existing.description,
            photos: existing.photos ?? [],
            _photos: options.photos
                ? [
                      ...((existing as Waypoint)._photos ?? []),
                      ...(incoming._photos ?? []),
                  ]
                : (existing as Waypoint)._photos,
        } as Waypoint;

        closeWaypointMergeModal();
        waypoint.set(mergedWaypoint);
        waypointModal.openModal();
    }

    function appendDistinctText(
        existing: string | undefined,
        incoming: string | undefined,
        separator: string,
    ) {
        const existingText = existing?.trim() ?? "";
        const incomingText = incoming?.trim() ?? "";

        if (!incomingText || existingText === incomingText) {
            return existing ?? "";
        }

        if (!existingText) {
            return incomingText;
        }

        return `${existingText}${separator}${incomingText}`;
    }

    function closeWaypointMergeModal() {
        pendingWaypointMerge = undefined;
        waypointMergeModal.closeModal();
    }

    function cancelPendingWaypointMerge() {
        if (pendingWaypointMerge) {
            waypoint.set(pendingWaypointMerge.incoming);
        }

        closeWaypointMergeModal();
        waypointModal.openModal();
    }

    function moveMarker(marker: M.Marker, wpId?: string) {
        const position = marker.getLngLat();
        const editableWaypointIndex =
            $formData.expand!.waypoints_via_trail?.findIndex(
                (w) => w.id == wpId,
            ) ?? -1;
        const editableWaypoint =
            $formData.expand!.waypoints_via_trail![editableWaypointIndex];
        if (!editableWaypoint) {
            return;
        }
        editableWaypoint.lat = position.lat;
        editableWaypoint.lon = position.lng;
        $formData.expand!.waypoints_via_trail = [
            ...($formData.expand!.waypoints_via_trail ?? []),
        ];
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

        if (
            $formData.expand?.summit_logs_via_trail?.length == 1 &&
            !$formData.completed
        ) {
            markTrailAsCompletedModal.openModal();
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

    function startDrawing() {
        drawingActive = true;
        routeSegments = [...(valhallaStore.route.trk?.at(0)?.trkseg ?? [])];

        if (!map) {
            return;
        }

        for (const anchor of valhallaStore.anchors) {
            anchor.marker?.addTo(map);
        }
    }

    function startReplacementDrawing() {
        replacingRoute = false;
        startDrawing();
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

    function openWaypointActionPopup(lngLat: M.LngLat) {
        mapPopup?.remove();

        mapPopup = createEditTrailMapPopup(lngLat, () => {
            mapPopup?.remove();
            beforeWaypointModalOpen(lngLat.lat, lngLat.lng);
        });
        mapPopup.addTo(map!);
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
            openWaypointActionPopup(e.lngLat);
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

    function handleMapContextMenu(e: M.MapMouseEvent) {
        if (!drawingActive || !showWaypointsWhileDrawing) {
            return;
        }
        if (
            (e.originalEvent.target as HTMLElement).tagName.toLowerCase() !==
            "canvas"
        ) {
            return;
        }
        e.preventDefault();
        openWaypointActionPopup(e.lngLat);
    }

    async function addAnchorAndRecalculate(lat: number, lon: number) {
        const previousAnchor =
            valhallaStore.anchors[valhallaStore.anchors.length - 1];
        if (!previousAnchor) {
            addAnchor(lat, lon, 0);
            return;
        }

        const anchor = addAnchor(lat, lon, valhallaStore.anchors.length);
        startAnchorLoading(anchor);
        try {
            const routeWaypoints = await calculateRouteBetween(
                previousAnchor.lat,
                previousAnchor.lon,
                lat,
                lon,
                routingOptions,
            );
            await insertIntoRoute(routeWaypoints);
            normalizeRouteTime();
            updateTrailWithRouteData();
        } catch (e) {
            console.error(e);
            show_toast({
                text: routeCalculationErrorText(e),
                icon: "close",
                type: "error",
            });
        } finally {
            stopAnchorLoading(anchor);
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
        refreshAnchorLabels(Math.max(0, index - 1));

        return anchor;
    }

    function startAnchorLoading(anchor: ValhallaAnchor) {
        const markerIcon = anchor.marker?.getElement();
        if (!markerIcon) {
            return;
        }
        markerIcon.classList.add("spinner", "spinner-light", "spinner-small");
        markerIcon.replaceChildren();
    }

    function stopAnchorLoading(anchor: ValhallaAnchor) {
        const markerIcon = anchor.marker?.getElement();
        if (!markerIcon) {
            return;
        }
        markerIcon.classList.remove(
            "spinner",
            "spinner-light",
            "spinner-small",
        );
        refreshAnchorLabel(valhallaStore.anchors.findIndex((a) => a.id === anchor.id));
    }

    function refreshAnchorLabel(index: number) {
        if (index < 0) {
            return;
        }

        const anchor = valhallaStore.anchors[index];
        const markerIcon = anchor.marker?.getElement();
        if (markerIcon) {
            renderValhallaAnchorMarker(
                markerIcon,
                index,
                valhallaStore.anchors.length,
            );
            anchor
                .marker!.getPopup()
                ._content.getElementsByTagName("h5")[0].textContent =
                valhallaAnchorTitle(index, valhallaStore.anchors.length, $_);
        }
    }

    function refreshAnchorLabels(startIndex: number = 0) {
        for (let i = startIndex; i < valhallaStore.anchors.length; i++) {
            refreshAnchorLabel(i);
        }
    }

    function highlightAnchorMarker(index: number | null) {
        for (const anchor of valhallaStore.anchors) {
            anchor.marker?.getElement().classList.remove("anchor-list-highlight");
        }

        if (index === null) {
            return;
        }

        valhallaStore.anchors[index]?.marker
            ?.getElement()
            .classList.add("anchor-list-highlight");
    }

    async function removeAnchor(anchorIndex: number) {
        if (!drawingActive) {
            return;
        }
        valhallaStore.anchors[anchorIndex]?.marker?.remove();
        valhallaStore.anchors.splice(anchorIndex, 1);
        refreshAnchorLabels(anchorIndex);
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
            await recalculateRoute(anchorIndex, [anchorIndex - 1, anchorIndex]);
        }
    }

    async function recalculateRouteFromAnchors(fromIndex: number, toIndex: number) {
        const anchors = valhallaStore.anchors;
        const N = anchors.length;

        if (N < 2) {
            setRoute(new GPX({ trk: [new Track({ trkseg: [] })] }), true);
            updateTrailWithRouteData();
            return;
        }

        // Segments not touching the moved anchor are reused (shifted by ±1); only the 2–3 boundary segments are recalculated.
        const oldSegments = valhallaStore.route.trk?.at(0)?.trkseg ?? [];
        const newSegments: (TrackSegment | null)[] = new Array(N - 1).fill(null);
        const toRecalc: number[] = [];

        if (fromIndex < toIndex) {
            for (let i = 0; i < fromIndex - 1; i++) newSegments[i] = oldSegments[i] ?? null;
            for (let i = fromIndex; i <= toIndex - 2; i++) newSegments[i] = oldSegments[i + 1] ?? null;
            for (let i = toIndex + 1; i < N - 1; i++) newSegments[i] = oldSegments[i] ?? null;
            if (fromIndex > 0) toRecalc.push(fromIndex - 1);
            toRecalc.push(toIndex - 1);
            if (toIndex < N - 1) toRecalc.push(toIndex);
        } else {
            for (let i = 0; i < toIndex - 1; i++) newSegments[i] = oldSegments[i] ?? null;
            for (let i = toIndex + 1; i <= fromIndex - 1; i++) newSegments[i] = oldSegments[i - 1] ?? null;
            for (let i = fromIndex + 1; i < N - 1; i++) newSegments[i] = oldSegments[i] ?? null;
            if (toIndex > 0) toRecalc.push(toIndex - 1);
            toRecalc.push(toIndex);
            if (fromIndex < N - 1) toRecalc.push(fromIndex);
        }

        const loadingAnchorIndexes = [...new Set(toRecalc.flatMap((i) => [i, i + 1]))];
        for (const index of loadingAnchorIndexes) {
            startAnchorLoading(anchors[index]);
        }
        try {
            const recalcResults = await Promise.all(
                toRecalc.map((i) =>
                    calculateRouteBetween(
                        anchors[i].lat,
                        anchors[i].lon,
                        anchors[i + 1].lat,
                        anchors[i + 1].lon,
                        routingOptions,
                    ).then((pts) => ({ i, segment: new TrackSegment({ trkpt: pts }) })),
                ),
            );

            for (const { i, segment } of recalcResults) {
                newSegments[i] = segment;
            }

            setRoute(
                new GPX({ trk: [new Track({ trkseg: newSegments.filter((s): s is TrackSegment => s !== null) })] }),
                true,
            );
            normalizeRouteTime();
            updateTrailWithRouteData();
        } finally {
            for (const index of loadingAnchorIndexes) {
                stopAnchorLoading(anchors[index]);
            }
        }
    }

    async function moveAnchor(fromIndex: number, toIndex: number) {
        if (
            routeAnchorListUpdating ||
            !drawingActive ||
            fromIndex === toIndex ||
            fromIndex < 0 ||
            toIndex < 0 ||
            fromIndex >= valhallaStore.anchors.length ||
            toIndex >= valhallaStore.anchors.length
        ) {
            return;
        }

        const previousAnchors = [...valhallaStore.anchors];
        const previousUndoStackLength = valhallaStore.undoStack.length;
        const [anchor] = valhallaStore.anchors.splice(fromIndex, 1);
        valhallaStore.anchors.splice(toIndex, 0, anchor);
        refreshAnchorLabels(Math.min(fromIndex, toIndex));

        routeAnchorListUpdating = true;
        try {
            await recalculateRouteFromAnchors(fromIndex, toIndex);
            const lastEntry = valhallaStore.undoStack.at(-1);
            if (lastEntry && valhallaStore.undoStack.length > previousUndoStackLength) {
                lastEntry.anchorsBefore = previousAnchors;
                lastEntry.anchorsAfter = [...valhallaStore.anchors];
            }
        } catch (e) {
            while (valhallaStore.undoStack.length > previousUndoStackLength) {
                revertRouteChange();
            }
            routeSegments = [...(valhallaStore.route.trk?.at(0)?.trkseg ?? [])];
            valhallaStore.anchors = previousAnchors;
            refreshAnchorLabels(Math.min(fromIndex, toIndex));
            console.error(e);
            show_toast({
                text: routeCalculationErrorText(e),
                icon: "close",
                type: "error",
            });
        } finally {
            routeAnchorListUpdating = false;
        }
    }

    async function recalculateRoute(anchorIndex: number, loadingAnchorIndexes = [anchorIndex]) {
        const anchor = valhallaStore.anchors[anchorIndex];
        if (!anchor) {
            return;
        }
        const anchors = valhallaStore.anchors;
        const loadingAnchors = [
            ...new Set(
                loadingAnchorIndexes
                    .map((index) => anchors[index])
                    .filter((anchor): anchor is ValhallaAnchor => Boolean(anchor)),
            ),
        ];
        for (const loadingAnchor of loadingAnchors) {
            startAnchorLoading(loadingAnchor);
        }
        let nextRouteSegment;
        let previousRouteSegment;
        try {
            if (anchorIndex < anchors.length - 1) {
                const nextAnchor = anchors[anchorIndex + 1];

                nextRouteSegment = await calculateRouteBetween(
                    anchor.lat,
                    anchor.lon,
                    nextAnchor.lat,
                    nextAnchor.lon,
                    routingOptions,
                );
            }
            if (anchorIndex > 0) {
                const previousAnchor = anchors[anchorIndex - 1];
                previousRouteSegment = await calculateRouteBetween(
                    previousAnchor.lat,
                    previousAnchor.lon,
                    anchor.lat,
                    anchor.lon,
                    routingOptions,
                );
            }

            if (nextRouteSegment) {
                await editRoute(anchorIndex, nextRouteSegment);
            }
            if (previousRouteSegment) {
                await editRoute(anchorIndex - 1, previousRouteSegment);
            }
            normalizeRouteTime();
            updateTrailWithRouteData();
        } catch (e) {
            console.error(e);
            show_toast({
                text: routeCalculationErrorText(e),
                icon: "close",
                type: "error",
            });
        } finally {
            for (const loadingAnchor of loadingAnchors) {
                stopAnchorLoading(loadingAnchor);
            }
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
        startAnchorLoading(anchor);

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

            await editRoute(data.segment, previousRouteSegment);
            await insertIntoRoute(nextRouteSegment, data.segment + 1);
            normalizeRouteTime();
            updateTrailWithRouteData();
        } catch (e) {
            console.error(e);
            show_toast({
                text: routeCalculationErrorText(e),
                icon: "close",
                type: "error",
            });
        } finally {
            stopAnchorLoading(anchor);
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

        await splitSegment(data.segment, data.event.lngLat);
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

    function requestReplaceRoute() {
        replaceRouteModal.openModal();
    }

    function replaceRoute() {
        resetRoute();
        clearUndoRedoStack();
        gpxFile = null;
        overwriteGPX = true;
        replacingRoute = true;
        drawingActive = false;
        routeSegments = [];
        $formData.expand!.gpx_data = undefined;
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
        routeSegments = [...(valhallaStore.route.trk?.at(0)?.trkseg ?? [])];
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
        selectedSearchLocation = item;
    }

    function clearSelectedSearchLocation() {
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
        if (!drawingActive) {
            return null;
        }
        return {
            label: $_("add-as-endpoint"),
            icon: "fa fa-flag-checkered",
            onClick: () => addAnchorAndRecalculate(lat, lon),
        } satisfies OverpassPopupAction;
    };

    async function addSelectedLocationAsEndpoint() {
        if (!selectedSearchLocation) {
            return;
        }
        const { lat, lon } = selectedSearchLocation.value;
        if (valhallaStore.anchors.length === 0) {
            addAnchor(lat, lon, 0);
        } else {
            await addAnchorAndRecalculate(lat, lon);
        }
        selectedSearchLocation = null;
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

    interface GPXCoord {
        id: string;
        longitude: number;
        latitude: number;
        file: File;
    }

    interface WaypointPhotoCluster {
        lat: number;
        lon: number;
        waypoint?: string;
        photos: string[];
    }

    interface WaypointPhotoClusterResponse {
        mergeEnabled: boolean;
        mergeRadius: number;
        clusters: WaypointPhotoCluster[];
    }

    interface WaypointClusterPoint {
        id: string;
        lat: number;
        lon: number;
    }

    interface WaypointPhotoClusterRequest {
        category?: string;
        photos: WaypointClusterPoint[];
        waypoints: WaypointClusterPoint[];
    }

    async function clusterWaypointPhotos(
        data: WaypointPhotoClusterRequest,
    ): Promise<WaypointPhotoClusterResponse> {
        const response = await fetch("/api/v1/waypoint/cluster", {
            method: "POST",
            headers: {
                "content-type": "application/json",
            },
            body: JSON.stringify(data),
        });

        if (!response.ok) {
            throw await response.json();
        }

        return (await response.json()) as WaypointPhotoClusterResponse;
    }

    const waypointMergeCheckPhotoId = "__waypoint_merge_check__";

    async function handleWaypointPhotoSelection() {
        const files = (
            document.getElementById("waypoint-photo-input") as HTMLInputElement
        ).files;

        if (!files) {
            return;
        }

        const photoCoords: GPXCoord[] = [];

        for (const [index, file] of Array.from(files).entries()) {
            const coords = await new Promise<GPXCoord | undefined>((resolve) => {
                EXIF.getData(file, function (p) {
                    const lat = EXIF.getTag(p, "GPSLatitude");
                    const latDir = EXIF.getTag(p, "GPSLatitudeRef");
                    const lon = EXIF.getTag(p, "GPSLongitude");
                    const lonDir = EXIF.getTag(p, "GPSLongitudeRef");

                    if (lat && lon) {
                        resolve({
                            id: index.toString(),
                            latitude: convertDMSToDD(lat, latDir),
                            longitude: convertDMSToDD(lon, lonDir),
                            file,
                        });
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

            photoCoords.push(coords);
        }

        let clusterResponse: WaypointPhotoClusterResponse;
        try {
            clusterResponse = await clusterWaypointPhotos({
                category: $formData.category,
                photos: photoCoords.map((coords) => ({
                    id: coords.id,
                    lat: coords.latitude,
                    lon: coords.longitude,
                })),
                waypoints: getExistingWaypointClusterInputs(),
            });
        } catch (e) {
            show_toast(
                {
                    type: "error",
                    icon: "warning",
                    text: $_("waypoint-cluster-error"),
                },
                10000,
            );
            return;
        }

        const fileMap = new Map(photoCoords.map((coords) => [coords.id, coords.file]));

        for (const cluster of clusterResponse.clusters) {
            const photos = cluster.photos
                .map((id) => fileMap.get(id))
                .filter((file): file is File => file != null);

            if (!photos.length) {
                continue;
            }

            if (cluster.waypoint) {
                const existingWaypoint =
                    $formData.expand?.waypoints_via_trail?.find(
                        (wp) => wp.id === cluster.waypoint,
                    );

                if (existingWaypoint) {
                    const existingWaypointPhotos =
                        (existingWaypoint as Waypoint)._photos ?? [];

                    commitWaypoint({
                        ...existingWaypoint,
                        photos: existingWaypoint.photos ?? [],
                        _photos: [...existingWaypointPhotos, ...photos],
                    } as Waypoint);
                    continue;
                }
            }

            const wp: Waypoint = new Waypoint(
                cluster.lat,
                cluster.lon,
                {
                    icon: photos.length > 1 ? "images" : "image",
                },
            );
            wp._photos = photos;
            commitWaypoint(wp);
        }
    }

    function undoRouteEdit() {
        const entry = undo();
        if (entry?.anchorsBefore) {
            valhallaStore.anchors = entry.anchorsBefore;
            refreshAnchorLabels();
        } else {
            clearAnchors();
            initRouteAnchors(valhallaStore.route, true);
        }
        updateTrailWithRouteData();
    }

    function redoRouteEdit() {
        const entry = redo();
        if (entry?.anchorsAfter) {
            valhallaStore.anchors = entry.anchorsAfter;
            refreshAnchorLabels();
        } else {
            clearAnchors();
            initRouteAnchors(valhallaStore.route, true);
        }
        updateTrailWithRouteData();
    }

    function markTrailAsCompleted() {
        setFields("completed", true);
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
        class="overflow-y-auto overflow-x-hidden flex flex-col gap-4 px-8 order-1 md:order-0 mt-8 md:mt-0"
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
                        class="flex h-9 w-9 shrink-0 items-center justify-center self-start rounded-full p-0 text-xl text-content hover:bg-secondary-hover"
                        aria-label={$_("add-as-endpoint")}
                        title={$_("add-as-endpoint")}
                        onclick={addSelectedLocationAsEndpoint}
                    >
                        <i class="fa fa-flag-checkered"></i>
                    </button>
                    <div class="flex-1">
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
            </div>
        {/if}
        <hr class="border-input-border" />
        {#if isNewTrail || replacingRoute || drawingActive || $formData.expand?.gpx_data}
            {#if isNewTrail || replacingRoute}
                <h3 class="text-xl font-semibold">{$_("pick-a-trail")}</h3>
            {/if}
            <button
                class="btn-primary"
                type="button"
                onclick={async () => {
                    if (drawingActive) {
                        await stopDrawing();
                    } else if (replacingRoute) {
                        startReplacementDrawing();
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
        {#if drawingActive && valhallaStore.anchors.length}
            <TrailAnchorList
                anchors={valhallaStore.anchors}
                segments={routeSegments}
                disabled={routeAnchorListUpdating}
                onMove={moveAnchor}
                onDelete={removeAnchor}
                onHover={highlightAnchorMarker}
            ></TrailAnchorList>
        {/if}
        {#if !drawingActive && (isNewTrail || replacingRoute)}
        <div class="flex gap-4 items-center w-full">
            <hr class="basis-full border-input-border" />
            <span class="text-gray-500 uppercase">{$_("or")}</span>
            <hr class="basis-full border-input-border" />
        </div>
        <Button
            primary={true}
            type="button"
            onclick={openFileBrowser}
            >{$formData.expand?.gpx_data
                ? $_("upload-new-file")
                : $_("upload-file")}</Button
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
            <CategoryPicker
                value={categorySelectValue}
                hiddenInputs
                currentCategoryId={data.trail.category}
                fixedDropdown
                onchange={handleCategoryChange}
            ></CategoryPicker>
        </div>

        <Toggle
            name="completed"
            label={$formData.completed ? $_("completed") : $_("not-completed")}
            icon={$formData.completed ? "flag-checkered" : "compass-drafting"}
        ></Toggle>
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
                    onReset={isNewTrail ? resetTrail : requestReplaceRoute}
                    resetLabel="reset-route"
                    resetAriaLabel="reset-route"
                    onCropToggle={toggleCropMarkers}
                    onCrop={confirmCrop}
                    onUpdateCropRange={updateCropMarkers}
                    onRecalculateElevationData={recalculateElevationData}
                    onUndo={undoRouteEdit}
                    onRedo={redoRouteEdit}
                    bind:showWaypoints={showWaypointsWhileDrawing}
                ></RouteEditor>
            </div>
        {/if}
        <div id="trail-map">
            <MapWithElevationMaplibre
                trails={mapTrail}
                waypoints={$formData.expand?.waypoints_via_trail}
                drawing={drawingActive}
                displayWaypoints={!drawingActive || showWaypointsWhileDrawing}
                showTerrain={true}
                autoGeolocateOnDrawing={page.params.id === "new"}
                onmarkerdragend={moveMarker}
                activeTrail={0}
                bind:map
                oninit={handleMapInit}
                onclick={(target) => handleMapClick(target)}
                oncontextmenu={(target) => handleMapContextMenu(target)}
                onsegmentclick={(data) => handleSegmentClick(data)}
                onsegmentdragend={(data) => handleSegmentDragEnd(data)}
                mapOptions={{ canvasContextAttributes: { preserveDrawingBuffer: true } }}
                {buildPoiAnchorAction}
            ></MapWithElevationMaplibre>
        </div>
    </div>
</main>
<WaypointModal bind:this={waypointModal} onsave={saveWaypoint}></WaypointModal>
<WaypointMergeModal
    merge={pendingWaypointMerge}
    bind:this={waypointMergeModal}
    oncreate={createPendingWaypointAnyway}
    onmerge={addPendingWaypointToExisting}
    oncancel={cancelPendingWaypointMerge}
></WaypointMergeModal>
<SummitLogModal bind:this={summitLogModal} onsave={(log) => saveSummitLog(log)}
></SummitLogModal>
<ListSearchModal
    lists={lists.items}
    bind:this={listSelectModal}
    onchange={(e) => handleListSelection(e)}
></ListSearchModal>
<ConfirmModal
    id="mark-trail-as-completed-modal"
    title={$_("mark-trail-as-completed")}
    text={$_("mark-trail-as-completed-modal-text")}
    action={$_("yes")}
    deny={$_("no")}
    bind:this={markTrailAsCompletedModal}
    onconfirm={markTrailAsCompleted}
></ConfirmModal>
<ConfirmModal
    id="replace-route-modal"
    title={$_("reset-route")}
    text={$_("reset-route-confirm")}
    action="reset-route"
    deny="cancel"
    bind:this={replaceRouteModal}
    onconfirm={replaceRoute}
></ConfirmModal>

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

    :global(.route-anchor.anchor-list-highlight) {
        border-color: rgb(255 255 255);
        box-shadow:
            0 0 0 4px rgba(var(--primary), 0.35),
            0 0 0 8px rgba(var(--primary), 0.16);
        z-index: 1;
    }
</style>
