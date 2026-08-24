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
    import PhotoPicker from "$lib/components/photo/photo_picker.svelte";
    import PhotoLibraryPickerModal from "$lib/components/photo/photo_library_picker_modal.svelte";
    import TrailAnchorList from "$lib/components/trail/trail_anchor_list.svelte";
    import RoundTripControls from "$lib/components/trail/round_trip_controls.svelte";
    import WaypointCard from "$lib/components/waypoint/waypoint_card.svelte";
    import WaypointMergeModal, {
        type WaypointMergeOptions,
    } from "$lib/components/waypoint/waypoint_merge_modal.svelte";
    import WaypointModal from "$lib/components/waypoint/waypoint_modal.svelte";
    import AssetWaypointModal from "$lib/components/trail/asset_waypoint_modal.svelte";
    import { SummitLogCreateSchema } from "$lib/models/api/summit_log_schema.js";
    import { TrailCreateSchema } from "$lib/models/api/trail_schema.js";
    import { WaypointCreateSchema } from "$lib/models/api/waypoint_schema.js";
    import GPX from "$lib/models/gpx/gpx";
    import GPXWaypoint from "$lib/models/gpx/waypoint";
    import type { List } from "$lib/models/list";
    import { SummitLog } from "$lib/models/summit_log";
    import { Trail } from "$lib/models/trail";
    import type { Asset } from "$lib/models/asset";
    import {
        mergePhotoLibraryPluginLinks,
        photoLibraryCandidateKey,
        photoLibraryPluginLinks,
        type PhotoLibraryCandidate,
    } from "$lib/models/photo_library";
    import type {
        RoutingOptions,
        RoutingAnchor,
        RoutingEffectiveControls,
        RoutingEngine,
        RoutingSettings,
        RoutingCandidate,
        RoutingSegmentProvenance,
    } from "$lib/models/routing";
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
    import { markGeneratedAssetFile } from "$lib/stores/asset_store";
    import { APIError } from "$lib/util/api_util";
    import {
        ROUTING_MAX_VARIANT_ANCHORS,
        ROUTING_MAX_VARIANTS,
    } from "$lib/util/routing_variant_util";
    import {
        applyRouteEditorRoutingEngineSelection,
        applyRoutingEffectiveControlDefaults,
        routingEngineSupportsVia,
        routingModeToTransport,
        selectEnabledRoutingPlugin,
    } from "$lib/util/routing_engine_util";
    import {
        anchorsForClosedLoopEdit,
        candidateForClosedLoop,
        closedLoopEditAction,
        isPersistedClosedLoop,
        roundTripLoopProvenanceForSegments,
        selectRoundTripRoutingEngine,
        shouldShowRoundTripControls,
    } from "$lib/util/routing_round_trip_util";
    import {
        routeCalculationErrorText as formatRouteCalculationError,
        routingPlanningKey,
        routingPreparationOptions,
    } from "$lib/util/trail_editor_routing_util";

    import {
        routingStore,
        calculateRouteBetween,
        calculateRouteForAnchors,
        calculateRoundTrip,
        calculateRouteVariants,
        applyRoutingCandidate,
        routingCandidateToGPX,
        routingProvenanceMatchesOptions,
        clearRoutingCandidates,
        markRoutingCandidatesStale,
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
        routingSettings as loadRoutingSettings,
        routingEffectiveControls as loadRoutingEffectiveControls,
        routingEngines as loadRoutingEngines,
        prepareRoutingProfile,
    } from "$lib/stores/routing_store.svelte.js";
    import { waypoint } from "$lib/stores/waypoint_store";
    import { getFileURL } from "$lib/util/file_util";
    import {
        formatDistance,
        formatElevation,
        formatTimeHHMM,
    } from "$lib/util/format_util";
    import { extractGPSCoordinates } from "$lib/util/exif_util";
    import { cropGPX, fromFile, gpx2trail } from "$lib/util/gpx_util";

    import { page } from "$app/state";
    import emptyStateTrailDark from "$lib/assets/svgs/empty_states/empty_state_trail_dark.svg";
    import emptyStateTrailLight from "$lib/assets/svgs/empty_states/empty_state_trail_light.svg";
    import Combobox, {
        type ComboboxItem,
    } from "$lib/components/base/combobox.svelte";
    import Dropdown, {
        type DropdownItem,
    } from "$lib/components/base/dropdown.svelte";
    import Editor from "$lib/components/base/editor.svelte";
    import Search, {
        type SearchItem,
    } from "$lib/components/base/search.svelte";
    import RouteEditor from "$lib/components/trail/route_editor.svelte";
    import RouteVariants from "$lib/components/trail/route_variants.svelte";
    import { TagCreateSchema } from "$lib/models/api/tag_schema.js";
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
        markerElement,
        syncMarkerHighlightClass,
    } from "$lib/util/maplibre_util";
    import {
        renderRoutingAnchorMarker,
        routingAnchorTitle,
    } from "$lib/util/routing_anchor_util";
    import { validator } from "@felte/validator-zod";
    import cryptoRandomString from "crypto-random-string";
    import { createForm } from "felte";
    import * as M from "maplibre-gl";
    import { onMount, tick, untrack } from "svelte";
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
    let mapWithElevation: MapWithElevationMaplibre | undefined = $state();
    let mapPopup: M.Popup | undefined;
    let mapTrail: Trail[] = $state([]);
    let lists = $state(untrack(() => data.lists));

    let waypointModal: WaypointModal;
    let assetWaypointModal: AssetWaypointModal;
    let trailPhotoLibraryModal: PhotoLibraryPickerModal = $state()!;
    let waypointMergeModal: WaypointMergeModal;
    let summitLogModal: SummitLogModal;
    let listSelectModal: ListSearchModal;
    let markTrailAsCompletedModal: ConfirmModal;
    let replaceRouteModal: ConfirmModal;
    let roundTripReplaceModal: ConfirmModal;
    let publishConfirmModal: ConfirmModal | undefined = $state();
    let publishConfirmed = false;
    let pendingLinkedPhotoCount = $state(0);

    let loading = $state(false);
    let pendingTrailPhotoCandidates: PhotoLibraryCandidate[] = $state([]);

    // Counts photos still stored as remote links (link_private). Publishing copies
    // them into wanderer, because a remote link cannot be served to public viewers.
    function countLinkedPrivatePhotos(): number {
        const expand = $formData.expand;
        if (!expand) return 0;
        const isLinked = (a: { storage_mode?: string }) =>
            a.storage_mode === "link_private";
        let count = (expand.assets_via_trail ?? []).filter(isLinked).length;
        for (const w of expand.waypoints_via_trail ?? []) {
            count += (w.expand?.assets_via_waypoint ?? []).filter(isLinked).length;
        }
        for (const s of expand.summit_logs_via_trail ?? []) {
            count += (s.expand?.assets_via_summit_log ?? []).filter(isLinked).length;
        }
        return count;
    }

    function confirmPublishWithLinkedPhotos() {
        publishConfirmed = true;
        (
            document.getElementById("trail-form") as HTMLFormElement | null
        )?.requestSubmit();
    }

    let editingBasicInfo: boolean = $state(false);

    let photoFiles: File[] = $state([]);

    let gpxFile: File | Blob | null = null;

    let drawingActive = $state(false);
    let showWaypointsWhileDrawing = $state(true);
    let replacingRoute = $state(false);
    let isNewTrail = $derived(page.params.id === "new");
    let shouldStartDrawingOnLoad = $derived(
        Boolean(data.duplicateOptions) || (!isNewTrail && !data.trail.completed),
    );

    function routeCalculationErrorText(error: unknown) {
        return formatRouteCalculationError(error, $_);
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

    // Assets are not edited by this form; carry them through untouched.
    const ClientAssetSchema = z.custom<Asset>();
    const PhotoLibraryPluginLinkSchema = z.object({
        pluginId: z.string(),
        assetIds: z.array(z.string()),
    });

    const ClientSummitLogCreateSchema = SummitLogCreateSchema.extend({
        photos: z.array(z.string()).default([]),
        _photos: z.array(z.instanceof(File)).optional(),
        _gpx: z.instanceof(Blob).optional().nullable(),
        _assetLinks: z.array(z.string()).optional(),
        _assetPluginLinks: z
            .array(PhotoLibraryPluginLinkSchema)
            .optional(),
        expand: z
            .object({
                gpx_data: z.string().optional(),
                assets_via_summit_log: z.array(ClientAssetSchema).optional(),
                summit_log_assets_via_summit_log: z.any().optional(),
            })
            .optional(),
    });

    const ClientWaypointCreateSchema = WaypointCreateSchema.extend({
        photos: z.array(z.string()).default([]),
        _photos: z.array(z.instanceof(File)).optional(),
        _assetLinks: z.array(z.string()).optional(),
        _assetPluginLinks: z
            .array(PhotoLibraryPluginLinkSchema)
            .optional(),
        expand: z
            .object({
                assets_via_waypoint: z.array(ClientAssetSchema).optional(),
                waypoint_assets_via_waypoint: z.any().optional(),
            })
            .optional(),
    });

    const ClientTrailCreateSchema = TrailCreateSchema.extend({
        photos: z.array(z.string()).default([]),
        _assetLinks: z.array(z.string()).optional(),
        _assetPluginLinks: z
            .array(PhotoLibraryPluginLinkSchema)
            .optional(),
        expand: z
            .object({
                gpx_data: z.string().optional(),
                assets_via_trail: z.array(ClientAssetSchema).optional(),
                trail_assets_via_trail: z.any().optional(),
                summit_logs_via_trail: z
                    .array(ClientSummitLogCreateSchema)
                    .optional(),
                waypoints_via_trail: z
                    .array(
                        ClientWaypointCreateSchema.extend({
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
    let hoveredRoutingVariantId: string | null | undefined = $state();
    function routingVariantMapTrailId(candidate: RoutingCandidate, index: number) {
        return `routing-variant-${index}-${candidate.id}`;
    }
    let routingVariantGPXById = $derived.by(() => {
        if (routingStore.routeCandidatesStale) return new Map();
        return new Map(
            routingStore.routeCandidates.map((candidate) => [
                candidate.id,
                routingCandidateToGPX(candidate),
            ]),
        );
    });
    let routingVariantMapTrails = $derived.by(() => {
        if (!drawingActive || routingStore.routeCandidatesStale) return [];
        return routingStore.routeCandidates.map((candidate, index) => {
            const preview = new Trail(`${$_("routing-variant")} ${index + 1}`);
            preview.id = routingVariantMapTrailId(candidate, index);
            preview.expand!.gpx = routingVariantGPXById.get(candidate.id);
            return preview;
        });
    });
    let routingVariantMapFocusId = $derived.by(() => {
        if (
            !drawingActive ||
            routingStore.routeCandidatesStale ||
            hoveredRoutingVariantId === undefined
        ) {
            return undefined;
        }
        if (hoveredRoutingVariantId === null) {
            return mapTrail[0]?.id;
        }
        const index = routingStore.routeCandidates.findIndex(
            (candidate) => candidate.id === hoveredRoutingVariantId,
        );
        const candidate = routingStore.routeCandidates[index];
        return candidate ? routingVariantMapTrailId(candidate, index) : undefined;
    });
    let routingVariantElevationPreview = $derived.by(() => {
        if (
            !drawingActive ||
            routingStore.routeCandidatesStale ||
            !routingStore.routeCandidateOrigin
        ) {
            return undefined;
        }
        const previewId =
            hoveredRoutingVariantId !== undefined
                ? hoveredRoutingVariantId
                : routingStore.selectedRouteCandidateId;
        if (previewId === null) {
            return routingStore.routeCandidateOrigin.route;
        }
        if (previewId === undefined) return undefined;
        const candidate = routingStore.routeCandidates.find(
            (routeCandidate) => routeCandidate.id === previewId,
        );
        return candidate ? routingVariantGPXById.get(candidate.id) : undefined;
    });
    let routeRoutingSettings: RoutingSettings | undefined = $state();
    let routeRoutingEngines: RoutingEngine[] = $state([]);
    let routeEffectiveControls: RoutingEffectiveControls | undefined = $state();
    let routeEffectiveControlsKey = "";
    let routeEffectiveControlsRequest = 0;
    let routeAnchorListUpdating = $state(false);
    let routeCalculationPendingCount = $state(0);
    let roundTripGenerating = $state(false);
    let pendingRoundTripRequest: { targetDistanceMeters: number; direction?: number } | undefined = $state();
    let originalRouteCalculating = $derived(
        routeCalculationPendingCount > 0 || routeAnchorListUpdating,
    );
    let routeSegments = $state<TrackSegment[]>([]);
    let dismissedRoutingMismatchKey = $state("");
    let lastRoutingCandidatePlanningKey = $state("");
    let routingEnabled = $derived(
        routeRoutingSettings !== undefined &&
            routeRoutingSettings.exposedFeatures?.routing !== false,
    );
    let routingPluginAvailable = $derived(
        routingEnabled &&
            !!routingOptions.routingPluginId &&
            routeRoutingEngines.some(
                (engine) =>
                    engine.pluginId === routingOptions.routingPluginId &&
                    engine.enabled,
            ),
    );
    let routingVariantsAvailable = $derived(
        routingEnabled &&
            routeRoutingSettings?.exposedFeatures?.variants !== false,
    );
    let parallelRoutingAvailable = $derived(
        routingEnabled &&
            routeRoutingSettings?.exposedFeatures?.parallelRouting !== false,
    );
    let viaRoutingAvailable = $derived(selectedRoutingEnginesSupportVia());
    let roundTripEngine = $derived(selectedRoundTripRoutingEngine());
    let roundTripAvailable = $derived(routingEnabled && roundTripEngine !== undefined);
    let roundTripControlsAvailable = $derived.by(() =>
        shouldShowRoundTripControls({
            capabilityAvailable: roundTripAvailable,
            anchorCount: routingStore.anchors.length,
            hasRoute: routeHasTrackPoints(),
            provenance: routingStore.segmentProvenance,
            segmentCount: routingStore.route.trk?.at(0)?.trkseg?.length ?? 0,
        }),
    );
    let routeEditorRoutingEngines = $derived(
        routeRoutingEngines.filter(
            (engine) => engine.enabled && engine.roles?.includes("route"),
        ),
    );
    let routingVariantAnchors = $derived.by(() =>
        routingStore.closedLoop && routingStore.anchors.length > 0
            ? [...routingStore.anchors, { ...routingStore.anchors[0] }]
            : routingStore.anchors,
    );
    let routingVariantMaxCount = $derived.by(() => {
        if (routingVariantAnchors.length > ROUTING_MAX_VARIANT_ANCHORS) return 0;
		return ROUTING_MAX_VARIANTS;
    });
    let routingVariantRequestError = $derived.by(() => {
        if (routingVariantAnchors.length > ROUTING_MAX_VARIANT_ANCHORS) {
            return $_("routing-error-anchor-limit", {
                values: { max: ROUTING_MAX_VARIANT_ANCHORS },
            });
        }
        return undefined;
    });
    let routingProvenanceMismatchKey = $derived.by(() => {
        const mismatching = routingStore.segmentProvenance
            .map((provenance, index) => ({ provenance, index }))
            .filter(
                (entry) =>
                    entry.provenance !== null &&
                    !routingProvenanceMatchesOptions(entry.provenance, routingOptions),
            );
        if (!mismatching.length) return "";
        return JSON.stringify({
            category: routingOptions.category ?? "",
            subcategory: routingOptions.subcategory ?? "",
            routingMode: routingOptions.routingMode ?? "segment",
            pluginId: routingOptions.routingPluginId ?? "",
            instanceId: routingOptions.routingInstanceId ?? "",
            nativeProfileKey: routingOptions.nativeProfileKey ?? "",
            profileRevisions: routingOptions.profileRevisions ?? {},
            preferences: routingOptions.preferences,
            nativeConfig: routingOptions.nativeConfig,
            mismatchingSegments: mismatching.map((entry) => entry.index),
        });
    });
    let showRoutingProvenanceMismatch = $derived(
        Boolean(routingProvenanceMismatchKey) &&
            routingProvenanceMismatchKey !== dismissedRoutingMismatchKey,
    );

    $effect(() => {
        const key = routingPlanningKey(routingOptions);
        if (lastRoutingCandidatePlanningKey && key !== lastRoutingCandidatePlanningKey) {
            clearRoutingCandidates();
        }
        lastRoutingCandidatePlanningKey = key;
    });

    $effect(() => {
        if (
            !drawingActive ||
            !routingOptions.autoRouting ||
            !routingOptions.routingPluginId ||
            !routingPluginAvailable
        ) {
            return;
        }
        const preparationOptions = routingPreparationOptions(routingOptions);
        const timeout = window.setTimeout(() => {
            // Drawing starts with ordinary auto-routing. Additional engines are
            // prepared only by an actual variant request.
            void prepareRoutingProfile(preparationOptions, false).catch(() => undefined);
        }, 500);
        return () => window.clearTimeout(timeout);
    });
    let drawRouteDisabled = $derived(
        !drawingActive &&
            (routeRoutingSettings === undefined ||
                (routingEnabled && !routingPluginAvailable)),
    );
    let drawRouteTooltip = $derived(
        drawRouteDisabled ? $_("no-routing-plugin-available") : undefined,
    );

    let savedAtLeastOnce = $state(false);

    let assetPluginIds = $derived(data.assetPluginIds);
    let canImportPhotosFromLibrary = $derived(!isNewTrail || savedAtLeastOnce);
    let photoImportDropdownItems: DropdownItem[] = $derived([
        {
            text: $_("upload-photos-from-device"),
            value: "device",
            icon: "image",
        },
        {
            text: $_("choose-photos-from-library"),
            value: "library",
            icon: "images",
            disabled: !canImportPhotosFromLibrary,
            tooltip: canImportPhotosFromLibrary
                ? undefined
                : $_("save-your-trail-first"),
        },
    ]);

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
            if (!publishConfirmed) {
                const publishForm = document.getElementById(
                    "trail-form",
                ) as HTMLFormElement | null;
                const willBePublic = !!(
                    publishForm && new FormData(publishForm).get("public")
                );
                const wasPublic = !!(data.trail.id && data.trail.public);
                if (!wasPublic && willBePublic) {
                    const linkedCount = countLinkedPrivatePhotos();
                    if (linkedCount > 0) {
                        pendingLinkedPhotoCount = linkedCount;
                        publishConfirmModal?.openModal();
                        return;
                    }
                }
            }
            publishConfirmed = false;
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
                applyTrailPhotoLibrarySelection(form as Trail);

                if (
                    !form.photos?.length &&
                    !photoFiles.length &&
                    !pendingTrailPhotoCandidates.length &&
                    !hasPendingAssetLinks(form as Trail)
                ) {
                    const canvas = document.querySelector(
                        "#map .maplibregl-canvas",
                    ) as HTMLCanvasElement;

                    const dataURL = canvas.toDataURL("image/webp", 0.3);
                    const response = await fetch(dataURL);
                    const blob = await response.blob();
                    photoFiles = [
                        markGeneratedAssetFile(
                            new File([blob], "wanderer-route-preview.webp", { type: "image/webp" }),
                            "route-preview",
                        ),
                    ];
                }

                form.expand!.gpx_data = routingStore.route.toString();
                form.routing_provenance = routingStore.segmentProvenance;
                if (
                    form.expand!.gpx_data &&
                    (overwriteGPX || (isNewTrail && !savedAtLeastOnce && !gpxFile && routeHasTrackPoints()))
                ) {
                    gpxFile = new Blob([form.expand!.gpx_data], {
                        type: "text/xml",
                    });
                }

                if (
                    (!form.lat || !form.lon) &&
                    routingStore.route.trk?.at(0)?.trkseg?.at(0)?.trkpt?.at(0)
                ) {
                    form.lat = routingStore.route.trk
                        ?.at(0)
                        ?.trkseg?.at(0)
                        ?.trkpt?.at(0)?.$.lat;
                    form.lon = routingStore.route.trk
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
                    savedAtLeastOnce = true;
                } else {
                    const updatedTrail = await trails_update(
                        $trail,
                        form as Trail,
                        photoFiles,
                        gpxFile,
                    );
                    setFields(updatedTrail);
                    savedAtLeastOnce = true;
                }
                photoFiles = [];
                pendingTrailPhotoCandidates = [];

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
        await initRoutingPhaseThreeState();

        const initialGpxData = $formData.expand?.gpx_data;
        if (initialGpxData) {
            $formData.id ??= cryptoRandomString({ length: 15 });
            const gpx = GPX.parse(initialGpxData);
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

                setRoute(gpx, false, $formData.routing_provenance ?? []);
                initRouteAnchors(gpx);

                updateTrailOnMap();

                if (shouldStartDrawingOnLoad) {
                    startDrawing();
                }
            }
        }
    });

    $effect(() => {
        const selection = selectedRoutingCategory();
        const pluginId = routingOptions.routingPluginId;
        const instanceId = routingOptions.routingInstanceId;
        if (!routingEnabled || !selection.category || !pluginId) {
            return;
        }
        if (routingOptions.category !== selection.category) {
            routingOptions.category = selection.category;
        }
        if (routingOptions.subcategory !== selection.subcategory) {
            routingOptions.subcategory = selection.subcategory;
        }
        void refreshRoutingEffectiveControls(
            selection.category,
            selection.subcategory,
            pluginId,
            instanceId,
        );
    });

    async function initRoutingPhaseThreeState() {
        try {
            const [settings, engines] = await Promise.all([
                loadRoutingSettings(),
                loadRoutingEngines(),
            ]);
            routeRoutingSettings = settings;
            routeRoutingEngines = engines;
            const enabled = settings.exposedFeatures?.routing !== false;
            if (!enabled) {
                routingOptions.autoRouting = false;
                routeEffectiveControls = undefined;
                clearRoutingCandidates();
            }

            routingOptions.routingPluginId = selectEnabledRoutingPlugin(
                routeRoutingEngines,
                settings.primaryRoutePluginId,
                "route",
            );
            routingOptions.routingInstanceId = routeRoutingEngines.find(
                (engine) => engine.pluginId === routingOptions.routingPluginId && engine.enabled,
            )?.instanceId;
            routingOptions.routingElevationPluginId = selectEnabledRoutingPlugin(
                routeRoutingEngines,
                settings.elevationPluginId,
                "elevation",
            );
			routingOptions.routingElevationInstanceId = routeRoutingEngines.find(
				(engine) =>
					engine.pluginId === routingOptions.routingElevationPluginId &&
					engine.enabled,
			)?.instanceId;
            routingOptions.routingMode = settings.defaultRoutingMode ?? "segment";
            routingOptions.routingModeExplicit = false;
            const parallelEnabled = settings.exposedFeatures?.parallelRouting !== false;
            routingOptions.engineMode = parallelEnabled ? "parallel" : "single";
            routingOptions.desiredVariants = Math.min(
                ROUTING_MAX_VARIANTS,
                Math.max(1, settings.defaultVariantCount ?? 3),
            );
            if (settings.exposedFeatures?.variants === false) {
                clearRoutingCandidates();
            }
            const selection = selectedRoutingCategory();
            routingOptions.category = selection.category;
            routingOptions.subcategory = selection.subcategory;
            if (enabled && selection.category) {
                await refreshRoutingEffectiveControls(
                    selection.category,
                    selection.subcategory,
                    routingOptions.routingPluginId,
                    routingOptions.routingInstanceId,
                );
            }
        } catch (e) {
            console.error(e);
        }
    }

    function selectedRoundTripRoutingEngine() {
        return selectRoundTripRoutingEngine(
            routeRoutingEngines,
            routingOptions.routingPluginId,
            routingOptions.routingInstanceId,
        );
    }

    function selectedRoutingEnginesSupportVia() {
        return selectedRoutingVariantEngines().some(routingEngineSupportsVia);
    }

    function selectedRoutingVariantEngines() {
        const engine = routeRoutingEngines.find(
            (candidate) =>
                candidate.pluginId === routingOptions.routingPluginId &&
                candidate.enabled &&
                candidate.roles?.includes("route"),
        );
        return engine ? [engine] : [];
    }

    async function selectRouteEditorRoutingEngine(pluginId: string) {
        const engine = applyRouteEditorRoutingEngineSelection(
            routingOptions,
            routeEditorRoutingEngines,
            pluginId,
        );
        if (!engine) return;
        routeEffectiveControls = undefined;
        clearRoutingCandidates();

        const selection = selectedRoutingCategory();
        if (selection.category) {
            await refreshRoutingEffectiveControls(
                selection.category,
                selection.subcategory,
                engine.pluginId,
                engine.instanceId,
            );
        }
    }

    function selectedRoutingCategory() {
        const category = data.categories.find((item) => item.id === $formData.category);
        const subcategory = data.subcategories.find(
            (item) => item.id === $formData.subcategory && item.category === category?.id,
        );
        return {
            category: category?.name ?? "",
            subcategory: subcategory?.name ?? "",
        };
    }

    async function refreshRoutingEffectiveControls(
        category: string,
        subcategory: string,
        pluginId: string,
        instanceId?: string,
    ) {
		const requestNumber = ++routeEffectiveControlsRequest;
		const requestKey = [pluginId, instanceId ?? "", category, subcategory].join(":");
        try {
            const request = {
                category,
                subcategory,
                routing: {
                    engines: [{ pluginId, instanceId }],
                },
            };
			const controls = await loadRoutingEffectiveControls({
                ...request,
            });
			if (requestNumber !== routeEffectiveControlsRequest) return;
			if (routeEffectiveControlsKey && routeEffectiveControlsKey !== requestKey) {
				routingOptions.preferences = {};
				routingOptions.nativeConfig = {};
			}
			routeEffectiveControlsKey = requestKey;
			routeEffectiveControls = controls;
            routingOptions.profileRevisions = routeEffectiveControls.profileRevisions ?? {};
            const transport = routingModeToTransport(routeEffectiveControls.mode);
            if (transport) {
                routingOptions.modeOfTransport = transport;
            }
            applyRoutingEffectiveControlDefaults(routingOptions, routeEffectiveControls);
        } catch (e) {
			if (requestNumber !== routeEffectiveControlsRequest) return;
            console.error(e);
            routeEffectiveControls = undefined;
        }
    }

    async function requestRouteVariants(desiredVariants = routingOptions.desiredVariants ?? 1) {
        if (routingStore.anchors.length < 2 || originalRouteCalculating) return;
        if (!routingVariantsAvailable) {
            show_toast({
                text: $_("routing-variants-disabled"),
                icon: "close",
                type: "error",
            });
            return;
        }
        if (routingVariantRequestError) {
            show_toast({ text: routingVariantRequestError, icon: "close", type: "error" });
            return;
        }
        const effectiveVariantCount = Math.max(
            1,
            Math.min(desiredVariants, routingVariantMaxCount),
        );
        try {
            await calculateRouteVariants(
                routingVariantAnchors,
                parallelRoutingAvailable
                    ? {
                          ...routingOptions,
                          desiredVariants: effectiveVariantCount,
                      }
                    : {
                          ...routingOptions,
                          engineMode: "single",
                          desiredVariants: effectiveVariantCount,
                      },
            );
            await tick();
            mapWithElevation?.fitToAllBounds();
        } catch (error) {
            console.error(error);
            show_toast({ text: routeCalculationErrorText(error), icon: "close", type: "error" });
        }
    }

    function applyRouteCandidate(candidate: RoutingCandidate) {
        const wasClosedLoop = routingStore.closedLoop;
        const previous = wasClosedLoop ? persistedClosedLoopMetadata() : undefined;
        if (wasClosedLoop && !previous) leaveInvalidClosedLoopState();
        const candidateToApply = previous
            ? candidateForClosedLoop(candidate, previous)
            : candidate;
        const snappedAnchors = applyRoutingCandidate(candidateToApply);
        routingStore.closedLoop = Boolean(previous) || candidate.compositionMode === "round_trip";
        refreshAnchorLabels();
        applySnappedAnchors(
            0,
            routingStore.closedLoop && snappedAnchors?.length === routingStore.anchors.length + 1
                ? snappedAnchors.slice(0, -1)
                : snappedAnchors,
        );
        normalizeRouteTime();
        updateTrailWithRouteData();
        hoveredRoutingVariantId = undefined;
        clearRoutingCandidates();
    }

    function selectRouteCandidate(candidate: RoutingCandidate) {
        routingStore.selectedRouteCandidateId = candidate.id;
    }

    function selectOriginalRoute() {
        routingStore.selectedRouteCandidateId = null;
    }

    function applySelectedRouteCandidate() {
        const candidate = routingStore.routeCandidates.find(
            (routeCandidate) =>
                routeCandidate.id === routingStore.selectedRouteCandidateId,
        );
        if (candidate) applyRouteCandidate(candidate);
    }

    function previewRouteCandidate(
        candidate: RoutingCandidate | null | undefined,
    ) {
        if (routingStore.routeCandidatesStale || candidate === undefined) {
            hoveredRoutingVariantId = undefined;
            return;
        }
        hoveredRoutingVariantId = candidate === null ? null : candidate.id;
    }

    async function recalculateEntireRoute() {
        const closedLoop = routingStore.closedLoop;
        const loopMetadata = closedLoop ? persistedClosedLoopMetadata() : undefined;
        const action = closedLoopEditAction({
            closedLoop,
            anchorCount: routingStore.anchors.length,
            hasTopology: Boolean(loopMetadata),
            autoRouting: routingOptions.autoRouting,
        });
        if (action === "clear") {
            setRoute(new GPX({ trk: [new Track({ trkseg: [] })] }), true);
            refreshAnchorLabels();
            updateTrailWithRouteData();
            return;
        }
        const loadingAnchors = [...routingStore.anchors];
        const useClosedLoop = action === "routed";
        if (action === "invalid") {
            leaveInvalidClosedLoopState();
        }
        loadingAnchors.forEach(startAnchorLoading);
        try {
            if (action === "manual") {
                await recalculateManualClosedLoop(loopMetadata!);
                return;
            }
            const response = await calculateRouteForAnchors(
                anchorsForClosedLoopEdit(routingStore.anchors, action),
                routingOptions,
                false,
            );
            const candidate = response.candidates?.[0];
            if (!candidate) {
                throw new APIError(502, "No route candidate returned");
            }
            if (response.warnings?.includes("routing_mode_fallback")) {
                routingOptions.routingMode = "segment";
                routingOptions.routingModeExplicit = false;
                show_toast({
                    text: $_("routing-mode-fallback"),
                    icon: "triangle-exclamation",
                    type: "warning",
                });
            }
            const candidateToApply = loopMetadata
                ? candidateForClosedLoop(candidate, loopMetadata)
                : candidate;
            const snappedAnchors = applyRoutingCandidate(candidateToApply);
            routingStore.closedLoop = useClosedLoop;
            applySnappedAnchors(
                0,
                useClosedLoop && snappedAnchors?.length === routingStore.anchors.length + 1
                    ? snappedAnchors.slice(0, -1)
                    : snappedAnchors,
            );
            normalizeRouteTime();
            updateTrailWithRouteData();
        } finally {
            loadingAnchors.forEach(stopAnchorLoading);
        }
    }

    function persistedClosedLoopMetadata() {
        const segmentCount = routingStore.route.trk?.at(0)?.trkseg?.length ?? 0;
        if (!isPersistedClosedLoop(routingStore.segmentProvenance, segmentCount)) {
            return undefined;
        }
        return routingStore.segmentProvenance.find(
            (entry): entry is RoutingSegmentProvenance =>
                entry?.routeTopology === "closed_loop",
        );
    }

    function leaveInvalidClosedLoopState() {
        routingStore.closedLoop = false;
        refreshAnchorLabels();
        show_toast({
            text: $_("routing-round-trip-state-lost"),
            icon: "triangle-exclamation",
            type: "warning",
        });
    }

    async function recalculateManualClosedLoop(previous: RoutingSegmentProvenance) {
        const anchors = routingStore.anchors;
        if (anchors.length < 2) {
            routingStore.closedLoop = false;
            return;
        }
        const results = await Promise.all(
            anchors.map((start, index) => {
                const end = anchors[(index + 1) % anchors.length];
                return calculateRouteBetween(
                    start.lat,
                    start.lon,
                    end.lat,
                    end.lon,
                    { ...routingOptions, autoRouting: false },
                );
            }),
        );
        const segments = results.map(
            (result) => new TrackSegment({ trkpt: result.waypoints }),
        );
        const route = new GPX({ trk: [new Track({ trkseg: segments })] });
        const actualDistance = route.getTotals().distance;
        setRoute(
            route,
            true,
            roundTripLoopProvenanceForSegments(
                previous,
                Array.from({ length: segments.length }),
                actualDistance,
            ),
        );
        routingStore.closedLoop = true;
        normalizeRouteTime();
        updateTrailWithRouteData();
    }

    async function generateRoundTrip(targetDistanceMeters: number, direction?: number) {
        if (routeHasTrackPoints() || routingStore.anchors.length > 1) {
            pendingRoundTripRequest = { targetDistanceMeters, direction };
            roundTripReplaceModal.openModal();
            return;
        }
        await performRoundTripGeneration(targetDistanceMeters, direction);
    }

    async function confirmRoundTripReplacement() {
        const request = pendingRoundTripRequest;
        pendingRoundTripRequest = undefined;
        if (!request) return;
        await performRoundTripGeneration(request.targetDistanceMeters, request.direction);
    }

    async function performRoundTripGeneration(targetDistanceMeters: number, direction?: number) {
        const engine = roundTripEngine;
        const start = routingStore.anchors[0];
        if (!engine || !start || roundTripGenerating) return;

        roundTripGenerating = true;
        startAnchorLoading(start);
        try {
            const response = await calculateRoundTrip(
                start,
                targetDistanceMeters,
                {
                    ...routingOptions,
                    routingPluginId: engine.pluginId,
                    routingInstanceId: engine.instanceId,
                },
                direction,
                undefined,
                engine.pluginId !== routingOptions.routingPluginId ||
                    engine.instanceId !== routingOptions.routingInstanceId,
            );
            const candidate = response.candidates?.[0];
            if (!candidate) {
                throw new APIError(502, "No round-trip candidate returned");
            }
            const snappedAnchors = candidate.snappedAnchors;
            if (!snappedAnchors?.length) {
                throw new APIError(502, "Round-trip candidate has no edit anchors");
            }
            applyRoutingCandidate(candidate, true, false);
            clearAnchors();
            for (const anchor of snappedAnchors) {
                addAnchor(anchor.lat, anchor.lon, routingStore.anchors.length);
            }
            routingStore.closedLoop = true;
            normalizeRouteTime();
            updateTrailWithRouteData();
            await tick();
            fitCurrentRoute();
            const warnings = new Set([
                ...(candidate.warnings ?? []),
                ...(response.warnings ?? []),
            ]);
            if (warnings.has("round_trip_target_tolerance_not_met")) {
                show_toast({
                    text: $_("routing-round-trip-tolerance-warning"),
                    icon: "triangle-exclamation",
                    type: "warning",
                });
            }
            if (warnings.has("round_trip_distance_adjustment_incomplete")) {
                show_toast({
                    text: $_("routing-round-trip-adjustment-warning"),
                    icon: "triangle-exclamation",
                    type: "warning",
                });
            }
            if (warnings.has("round_trip_candidates_truncated")) {
                show_toast({
                    text: $_("routing-round-trip-provider-warning"),
                    icon: "triangle-exclamation",
                    type: "warning",
                });
            }
        } catch (error) {
            console.error(error);
            show_toast({ text: routeCalculationErrorText(error), icon: "close", type: "error" });
        } finally {
            stopAnchorLoading(start);
            roundTripGenerating = false;
        }
    }

    function keepExistingRoutingSegments() {
        dismissedRoutingMismatchKey = routingProvenanceMismatchKey;
    }

    function fitCurrentRoute() {
        const bounds = routingStore.route.toGeoJSON().bbox;
        if (!bounds) {
            return;
        }

        mapWithElevation?.fitToBounds(bounds as M.LngLatBoundsLike);
    }

    function handleMapInit(initializedMap: M.Map) {
        if (drawingActive) {
            for (const anchor of routingStore.anchors) {
                anchor.marker?.addTo(initializedMap);
            }
        }
        if ($formData.expand?.gpx_data) {
            fitCurrentRoute();
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
                fitCurrentRoute();
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
        routingStore.closedLoop = isPersistedClosedLoop(
            routingStore.segmentProvenance,
            segments.length,
        );

        for (let i = 0; i < segments.length; i++) {
            const segment = segments[i];
            const points = segment.trkpt ?? [];

            if (points.length > 0) {
                addAnchor(
                    points[0].$.lat!,
                    points[0].$.lon!,
                    routingStore.anchors.length,
                    addToMap,
                );
            }
            if (i == segments.length - 1 && !routingStore.closedLoop && points.length > 0) {
                addAnchor(
                    points[points.length - 1].$.lat!,
                    points[points.length - 1].$.lon!,
                    routingStore.anchors.length,
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

    function onAssetPluginImport(importedWaypoints: Waypoint[]) {
        if (!importedWaypoints.length) return;
        const waypoints = importedWaypoints;
        const nextWaypoints = mergeAssetPluginWaypoints(
            $formData.expand!.waypoints_via_trail ?? [],
            waypoints,
        );
        $formData.expand!.waypoints_via_trail = nextWaypoints;
    }

    function onTrailPhotoLibrarySelect(candidates: PhotoLibraryCandidate[]) {
        pendingTrailPhotoCandidates = [
            ...pendingTrailPhotoCandidates,
            ...candidates.filter(
                (candidate) =>
                    !pendingTrailPhotoCandidates.some(
                        (pending) =>
                            photoLibraryCandidateKey(pending) ===
                            photoLibraryCandidateKey(candidate),
                    ),
            ),
        ];
    }

    function removePendingTrailPhotoCandidate(assetId: string) {
        pendingTrailPhotoCandidates = pendingTrailPhotoCandidates.filter(
            (candidate) => candidate.assetId !== assetId,
        );
    }

    function applyTrailPhotoLibrarySelection(target: Trail) {
        if (!pendingTrailPhotoCandidates.length) {
            return;
        }
        const pluginCandidates = pendingTrailPhotoCandidates.filter(
            (candidate) => candidate.source !== "wanderer",
        );
        const wandererCandidates = pendingTrailPhotoCandidates.filter(
            (candidate) => candidate.source === "wanderer",
        );
        target._assetLinks = Array.from(new Set([
            ...(target._assetLinks ?? []),
            ...wandererCandidates.map((candidate) => candidate.assetId),
        ]));
        target._assetPluginLinks = mergePhotoLibraryPluginLinks(
            target._assetPluginLinks,
            photoLibraryPluginLinks(pluginCandidates),
        );
        target.photos = Array.from(
            new Set([
                ...(target.photos ?? []),
                ...wandererCandidates
                    .map((candidate) => candidate.thumbnailUrl)
                    .filter((url): url is string => Boolean(url)),
            ]),
        );
    }

    function hasPendingAssetLinks(target: Trail) {
        return Boolean(
            target._assetLinks?.length ||
            target._assetPluginLinks?.some((link) => link.assetIds.length),
        );
    }

    function mergeAssetPluginWaypoints(existing: Waypoint[], imported: Waypoint[]) {
        const merged = existing.map(normalizeAssetPluginWaypoint);
        const indexById = new Map<string, number>();
        for (const [index, waypoint] of merged.entries()) {
            if (waypoint.id) {
                indexById.set(waypoint.id, index);
            }
        }

        for (const rawWaypoint of imported) {
            const waypoint = normalizeAssetPluginWaypoint(rawWaypoint);
            const existingIndex = waypoint.id ? indexById.get(waypoint.id) : undefined;
            if (existingIndex === undefined) {
                if (waypoint.id) {
                    indexById.set(waypoint.id, merged.length);
                }
                merged.push(waypoint);
                continue;
            }
            merged[existingIndex] = mergeAssetPluginWaypoint(merged[existingIndex], waypoint);
        }
        return sortWaypointsByDistanceFromStart(merged);
    }

    function normalizeAssetPluginWaypoint(waypoint: Waypoint): Waypoint {
        return { ...waypoint, photos: waypoint.photos ?? [] };
    }

    function mergeAssetPluginWaypoint(existing: Waypoint, incoming: Waypoint): Waypoint {
        const candidates = uniqueAssetCandidates([
            ...(existing._assetCandidates ?? []),
            ...(incoming._assetCandidates ?? []),
        ]);
        const assetLinks = Array.from(new Set([
            ...(existing._assetLinks ?? []),
            ...(incoming._assetLinks ?? []),
        ]));
        const assetPluginLinks = mergePhotoLibraryPluginLinks(
            existing._assetPluginLinks,
            incoming._assetPluginLinks,
        );
        return {
            ...existing,
            ...incoming,
            marker: existing.marker,
            photos: incoming.photos?.length ? incoming.photos : (existing.photos ?? []),
            _photos: [
                ...(existing._photos ?? []),
                ...(incoming._photos ?? []),
            ],
            _assetCandidates: candidates.length ? candidates : undefined,
            _assetLinks: assetLinks.length ? assetLinks : undefined,
            _assetPluginLinks: assetPluginLinks,
        };
    }

    function uniqueAssetCandidates(candidates: NonNullable<Waypoint["_assetCandidates"]>) {
        const seen = new Set<string>();
        return candidates.filter((candidate) => {
            const key = `${candidate.pluginId ?? ""}:${candidate.assetId}`;
            if (seen.has(key)) {
                return false;
            }
            seen.add(key);
            return true;
        });
    }

    function sortWaypointsByDistanceFromStart(waypoints: Waypoint[]) {
        return [...waypoints].sort(
            (a, b) => (a.distance_from_start ?? 0) - (b.distance_from_start ?? 0),
        );
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
        routeSegments = [...(routingStore.route.trk?.at(0)?.trkseg ?? [])];

        if (!map) {
            return;
        }

        for (const anchor of routingStore.anchors) {
            anchor.marker?.addTo(map);
        }
    }

    function startReplacementDrawing() {
        replacingRoute = false;
        startDrawing();
    }

    async function stopDrawing() {
        drawingActive = false;
        for (const anchor of routingStore.anchors) {
            anchor.marker?.remove();
        }
        toggleCropMarkers(false);
        clearUndoRedoStack();

        if (routingStore.route.trk?.at(0)?.trkseg?.at(0)?.trkpt?.at(0)) {
            $formData.lat = routingStore.route.trk
                ?.at(0)
                ?.trkseg?.at(0)
                ?.trkpt?.at(0)?.$.lat;
            $formData.lon = routingStore.route.trk
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
            const anchorCount = routingStore.anchors.length;
            if (anchorCount == 0) {
                addAnchor(
                    e.lngLat.lat,
                    e.lngLat.lng,
                    routingStore.anchors.length,
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
            routingStore.anchors[routingStore.anchors.length - 1];
        if (!previousAnchor) {
            addAnchor(lat, lon, 0);
            return;
        }

        const anchor = addAnchor(lat, lon, routingStore.anchors.length);
        if (
            routingStore.closedLoop ||
            (routingOptions.autoRouting && routingOptions.routingMode === "via")
        ) {
            try {
                await recalculateEntireRoute();
            } catch (e) {
                console.error(e);
                show_toast({ text: routeCalculationErrorText(e), icon: "close", type: "error" });
            }
            return;
        }
        startAnchorLoading(anchor);
        try {
            const routeResult = await calculateRouteBetween(
                previousAnchor.lat,
                previousAnchor.lon,
                lat,
                lon,
                routingOptions,
            );
            applySnappedAnchors(routingStore.anchors.length - 2, routeResult.snappedAnchors);
            await insertIntoRoute(routeResult.waypoints, undefined, routeResult.provenance);
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
        const anchor: RoutingAnchor = {
            id: cryptoRandomString({ length: 15 }),
            lat: lat,
            lon: lon,
        };
        const marker = createAnchorMarker(
            lat,
            lon,
            () => {
                removeAnchor(
                    routingStore.anchors.findIndex((a) => a.id == anchor.id),
                );
            },
            () => {
                if (routingStore.closedLoop) {
                    marker.togglePopup();
                    return;
                }
                const thisAnchor = routingStore.anchors.find(
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
                const anchorIndex = routingStore.anchors.findIndex(
                    (a) => a.id == anchor.id,
                );
                const thisAnchor = routingStore.anchors[anchorIndex];
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
        routingStore.anchors.splice(index, 0, anchor);
        refreshAnchorLabels(Math.max(0, index - 1));

        return anchor;
    }

    function startAnchorLoading(anchor: RoutingAnchor) {
        if (routeCalculationPendingCount === 0) {
            hoveredRoutingVariantId = undefined;
            markRoutingCandidatesStale();
        }
        routeCalculationPendingCount += 1;
        const markerIcon = anchor.marker?.getElement();
        if (!markerIcon) {
            return;
        }
        markerIcon.classList.add("spinner", "spinner-light", "spinner-small");
        markerIcon.replaceChildren();
    }

    function stopAnchorLoading(anchor: RoutingAnchor) {
        routeCalculationPendingCount = Math.max(
            0,
            routeCalculationPendingCount - 1,
        );
        const markerIcon = anchor.marker?.getElement();
        if (!markerIcon) {
            return;
        }
        markerIcon.classList.remove(
            "spinner",
            "spinner-light",
            "spinner-small",
        );
        refreshAnchorLabel(routingStore.anchors.findIndex((a) => a.id === anchor.id));
    }

    function refreshAnchorLabel(index: number) {
        if (index < 0) {
            return;
        }

        const anchor = routingStore.anchors[index];
        const markerIcon = anchor.marker?.getElement();
        if (markerIcon) {
            const popupContent = anchor.marker!.getPopup()._content;
            renderRoutingAnchorMarker(
                markerIcon,
                index,
                routingStore.anchors.length,
                routingStore.closedLoop,
            );
            popupContent.getElementsByTagName("h5")[0].textContent =
                routingAnchorTitle(
                    index,
                    routingStore.anchors.length,
                    $_,
                    routingStore.closedLoop,
                );
            popupContent
                .querySelector(".route-end-action")
                ?.classList.toggle("hidden", routingStore.closedLoop);
        }
    }

    function refreshAnchorLabels(startIndex: number = 0) {
        for (let i = startIndex; i < routingStore.anchors.length; i++) {
            refreshAnchorLabel(i);
        }
    }

    function applySnappedAnchors(startIndex: number, snappedAnchors?: { lat: number; lon: number }[]) {
        if (!snappedAnchors?.length) {
            return;
        }
        for (let i = 0; i < snappedAnchors.length; i++) {
            const anchor = routingStore.anchors[startIndex + i];
            const snapped = snappedAnchors[i];
            if (!anchor || !Number.isFinite(snapped.lat) || !Number.isFinite(snapped.lon)) {
                continue;
            }
            anchor.lat = snapped.lat;
            anchor.lon = snapped.lon;
            anchor.marker?.setLngLat([snapped.lon, snapped.lat]);
        }
    }

    function highlightAnchorMarker(index: number | null) {
        syncMarkerHighlightClass(
            routingStore.anchors.map((anchor) => markerElement(anchor.marker)),
            index === null ? undefined : markerElement(routingStore.anchors[index]?.marker),
            "anchor-list-highlight",
        );
    }

    async function removeAnchor(anchorIndex: number) {
        if (!drawingActive) {
            return;
        }
        routingStore.anchors[anchorIndex]?.marker?.remove();
        routingStore.anchors.splice(anchorIndex, 1);
        refreshAnchorLabels(anchorIndex);
        if (routingStore.closedLoop) {
            const action = closedLoopEditAction({
                closedLoop: true,
                anchorCount: routingStore.anchors.length,
                hasTopology: Boolean(persistedClosedLoopMetadata()),
                autoRouting: routingOptions.autoRouting,
            });
            if (action === "clear") {
                setRoute(new GPX({ trk: [new Track({ trkseg: [] })] }), true);
                refreshAnchorLabels();
                updateTrailWithRouteData();
            } else {
                await recalculateEntireRoute();
            }
            return;
        }
        if (routingOptions.autoRouting && routingOptions.routingMode === "via") {
            await recalculateEntireRoute();
            return;
        }
        if (anchorIndex == 0) {
            deleteFromRoute(anchorIndex);
            if ($formData.expand?.gpx_data) {
                updateTrailWithRouteData();
            }
        } else if (anchorIndex == routingStore.anchors.length) {
            deleteFromRoute(anchorIndex - 1);
            updateTrailWithRouteData();
        } else {
            deleteFromRoute(anchorIndex - 1);
            await recalculateRoute(anchorIndex, [anchorIndex - 1, anchorIndex]);
        }
    }

    async function recalculateRouteFromAnchors(fromIndex: number, toIndex: number) {
        const anchors = routingStore.anchors;
        const N = anchors.length;

        if (N < 2) {
            setRoute(new GPX({ trk: [new Track({ trkseg: [] })] }), true);
            updateTrailWithRouteData();
            return;
        }
        if (routingStore.closedLoop) {
            await recalculateEntireRoute();
            return;
        }
        if (routingOptions.autoRouting && routingOptions.routingMode === "via") {
            await recalculateEntireRoute();
            return;
        }

        // Segments not touching the moved anchor are reused (shifted by ±1); only the 2–3 boundary segments are recalculated.
        const oldSegments = routingStore.route.trk?.at(0)?.trkseg ?? [];
        const oldProvenance = routingStore.segmentProvenance;
        const newSegments: (TrackSegment | null)[] = new Array(N - 1).fill(null);
        const newProvenance = new Array(N - 1).fill(null);
        const toRecalc: number[] = [];

        if (fromIndex < toIndex) {
            for (let i = 0; i < fromIndex - 1; i++) {
                newSegments[i] = oldSegments[i] ?? null;
                newProvenance[i] = oldProvenance[i] ?? null;
            }
            for (let i = fromIndex; i <= toIndex - 2; i++) {
                newSegments[i] = oldSegments[i + 1] ?? null;
                newProvenance[i] = oldProvenance[i + 1] ?? null;
            }
            for (let i = toIndex + 1; i < N - 1; i++) {
                newSegments[i] = oldSegments[i] ?? null;
                newProvenance[i] = oldProvenance[i] ?? null;
            }
            if (fromIndex > 0) toRecalc.push(fromIndex - 1);
            toRecalc.push(toIndex - 1);
            if (toIndex < N - 1) toRecalc.push(toIndex);
        } else {
            for (let i = 0; i < toIndex - 1; i++) {
                newSegments[i] = oldSegments[i] ?? null;
                newProvenance[i] = oldProvenance[i] ?? null;
            }
            for (let i = toIndex + 1; i <= fromIndex - 1; i++) {
                newSegments[i] = oldSegments[i - 1] ?? null;
                newProvenance[i] = oldProvenance[i - 1] ?? null;
            }
            for (let i = fromIndex + 1; i < N - 1; i++) {
                newSegments[i] = oldSegments[i] ?? null;
                newProvenance[i] = oldProvenance[i] ?? null;
            }
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
                    ).then((result) => ({ i, result, segment: new TrackSegment({ trkpt: result.waypoints }) })),
                ),
            );

            for (const { i, result, segment } of recalcResults) {
                applySnappedAnchors(i, result.snappedAnchors);
                newSegments[i] = segment;
                newProvenance[i] = result.provenance ?? null;
            }

            setRoute(
                new GPX({ trk: [new Track({ trkseg: newSegments.filter((s): s is TrackSegment => s !== null) })] }),
                true,
                newProvenance,
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
            fromIndex >= routingStore.anchors.length ||
            toIndex >= routingStore.anchors.length
        ) {
            return;
        }

        const previousAnchors = [...routingStore.anchors];
        const previousUndoStackLength = routingStore.undoStack.length;
        const [anchor] = routingStore.anchors.splice(fromIndex, 1);
        routingStore.anchors.splice(toIndex, 0, anchor);
        refreshAnchorLabels(Math.min(fromIndex, toIndex));

        routeAnchorListUpdating = true;
        try {
            await recalculateRouteFromAnchors(fromIndex, toIndex);
            const lastEntry = routingStore.undoStack.at(-1);
            if (lastEntry && routingStore.undoStack.length > previousUndoStackLength) {
                lastEntry.anchorsBefore = previousAnchors;
                lastEntry.anchorsAfter = [...routingStore.anchors];
            }
        } catch (e) {
            while (routingStore.undoStack.length > previousUndoStackLength) {
                revertRouteChange();
            }
            routeSegments = [...(routingStore.route.trk?.at(0)?.trkseg ?? [])];
            routingStore.anchors = previousAnchors;
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
        const anchor = routingStore.anchors[anchorIndex];
        if (!anchor) {
            return;
        }
        if (routingStore.closedLoop) {
            try {
                await recalculateEntireRoute();
            } catch (e) {
                console.error(e);
                show_toast({ text: routeCalculationErrorText(e), icon: "close", type: "error" });
            }
            return;
        }
        if (routingOptions.autoRouting && routingOptions.routingMode === "via") {
            try {
                await recalculateEntireRoute();
            } catch (e) {
                console.error(e);
                show_toast({ text: routeCalculationErrorText(e), icon: "close", type: "error" });
            }
            return;
        }
        const anchors = routingStore.anchors;
        const loadingAnchors = [
            ...new Set(
                loadingAnchorIndexes
                    .map((index) => anchors[index])
                    .filter((anchor): anchor is RoutingAnchor => Boolean(anchor)),
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
                applySnappedAnchors(anchorIndex, nextRouteSegment.snappedAnchors);
                await editRoute(anchorIndex, nextRouteSegment.waypoints, nextRouteSegment.provenance);
            }
            if (previousRouteSegment) {
                applySnappedAnchors(anchorIndex - 1, previousRouteSegment.snappedAnchors);
                await editRoute(anchorIndex - 1, previousRouteSegment.waypoints, previousRouteSegment.provenance);
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
        if (
            routingStore.closedLoop ||
            (routingOptions.autoRouting && routingOptions.routingMode === "via")
        ) {
            try {
                await recalculateEntireRoute();
            } catch (e) {
                console.error(e);
                show_toast({ text: routeCalculationErrorText(e), icon: "close", type: "error" });
            }
            return;
        }
        startAnchorLoading(anchor);

        const previousAnchor = routingStore.anchors[data.segment];
        const nextAnchor = routingStore.anchors[data.segment + 2];

        try {
            const previousRouteResult = await calculateRouteBetween(
                previousAnchor.lat,
                previousAnchor.lon,
                anchor.lat,
                anchor.lon,
                routingOptions,
            );
            applySnappedAnchors(data.segment, previousRouteResult.snappedAnchors);
            const nextRouteResult = await calculateRouteBetween(
                anchor.lat,
                anchor.lon,
                nextAnchor.lat,
                nextAnchor.lon,
                routingOptions,
            );
            applySnappedAnchors(data.segment + 1, nextRouteResult.snappedAnchors);

            await editRoute(data.segment, previousRouteResult.waypoints, previousRouteResult.provenance);
            await insertIntoRoute(nextRouteResult.waypoints, data.segment + 1, nextRouteResult.provenance);
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

    async function reverseTrail() {
        reverseRoute();
        if (routingOptions.autoRouting && routingOptions.routingMode === "via") {
            try {
                await recalculateEntireRoute();
            } catch (error) {
                show_toast({ text: routeCalculationErrorText(error), icon: "close", type: "error" });
            }
        }
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

            updateTotals(routingStore.route);
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

        const flatRoute = routingStore.route.flatten();

        const targetStartDistance =
            routingStore.route.features.distance * (start / 100);
        const [startLon, startLat, startIndex] = getCoordinateAtDistance(
            flatRoute,
            routingStore.route.features.cumulativeDistance,
            targetStartDistance,
        );

        const targetEndDistance =
            routingStore.route.features.distance * (end / 100);
        const [endLon, endLat, endIndex] = getCoordinateAtDistance(
            flatRoute,
            routingStore.route.features.cumulativeDistance,
            targetEndDistance,
        );

        cropStartMarker.setLngLat([startLon, startLat]);
        cropEndMarker.setLngLat([endLon, endLat]);

        croppedGPX = cropGPX(
            flatRoute[startIndex],
            flatRoute[endIndex],
            routingStore.route,
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
        routeSegments = [...(routingStore.route.trk?.at(0)?.trkseg ?? [])];
        updateTotals(routingStore.route);

        if (!$formData.id) {
            $formData.id = cryptoRandomString({ length: 15 });
        }
        updateTrailOnMap();
    }

    function routeHasTrackPoints() {
        return Boolean(
            routingStore.route.trk?.some((track) =>
                track.trkseg?.some((segment) => (segment.trkpt?.length ?? 0) > 0),
            ),
        );
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
        t.expand!.gpx = routingStore.route;
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
        if (routingStore.anchors.length === 0) {
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

    function handlePhotoImportMenuClick(item: DropdownItem) {
        if (item.value === "device") {
            openPhotoBrowser();
        } else if (item.value === "library") {
            assetWaypointModal.openModal();
        }
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
        name?: string;
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
        resolveNames?: boolean;
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
            const coordinates = await extractGPSCoordinates(file);
            if (!coordinates) {
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

            photoCoords.push({
                id: index.toString(),
                latitude: coordinates.lat,
                longitude: coordinates.lon,
                file,
            });
        }

        let clusterResponse: WaypointPhotoClusterResponse;
        try {
            clusterResponse = await clusterWaypointPhotos({
                category: $formData.category,
                resolveNames: true,
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
                    name: cluster.name,
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
            routingStore.anchors = entry.anchorsBefore;
            refreshAnchorLabels();
        } else {
            clearAnchors();
            initRouteAnchors(routingStore.route, true);
        }
        updateTrailWithRouteData();
    }

    function redoRouteEdit() {
        const entry = redo();
        if (entry?.anchorsAfter) {
            routingStore.anchors = entry.anchorsAfter;
            refreshAnchorLabels();
        } else {
            clearAnchors();
            initRouteAnchors(routingStore.route, true);
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
            <Button
                primary={true}
                disabled={drawRouteDisabled}
                tooltip={drawRouteTooltip}
                type="button"
                onclick={async () => {
                    if (drawingActive) {
                        await stopDrawing();
                    } else if (drawRouteDisabled) {
                        return;
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
                        : $_("draw-a-route")}</Button
            >
        {/if}
        {#if drawingActive && roundTripControlsAvailable}
            <RoundTripControls
                loading={roundTripGenerating}
                disabled={originalRouteCalculating}
                onGenerate={generateRoundTrip}
            ></RoundTripControls>
        {/if}
        {#if drawingActive &&
            routingOptions.autoRouting &&
            routingVariantsAvailable &&
            routingStore.anchors.length >= 2}
            <RouteVariants
                bind:options={routingOptions}
                onFindVariants={requestRouteVariants}
                onSelectOriginal={selectOriginalRoute}
                onSelectCandidate={selectRouteCandidate}
                onApplySelected={applySelectedRouteCandidate}
                onPreviewCandidate={previewRouteCandidate}
                disabled={originalRouteCalculating}
                {parallelRoutingAvailable}
                variantRequestError={routingVariantRequestError}
                maxVariants={routingVariantMaxCount}
            ></RouteVariants>
        {/if}
        {#if drawingActive && routingStore.anchors.length}
            <TrailAnchorList
                anchors={routingStore.anchors}
                segments={routeSegments}
                closedLoop={routingStore.closedLoop}
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
        <Dropdown
            items={photoImportDropdownItems}
            onchange={handlePhotoImportMenuClick}
            matchToggleWidth={true}
        >
            {#snippet children({ toggleMenu: openDropdown })}
                <Button
                    secondary={true}
                    type="button"
                    extraClasses="w-full"
                    onclick={openDropdown}
                    ><i class="fa fa-images mr-2"></i>{$_("from-photos")}<i
                        class="fa fa-caret-down ml-2 text-xs"
                    ></i></Button
                >
            {/snippet}
        </Dropdown>
        <input
            type="file"
            id="waypoint-photo-input"
            accept="image/jpeg,image/png,image/heic,image/heif,image/heic-sequence,image/heif-sequence,image/webp,image/gif,image/avif,.jpg,.jpeg,.png,.heic,.heif,.webp,.gif,.avif"
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
            onassetplugin={canImportPhotosFromLibrary
                ? () => trailPhotoLibraryModal.openModal()
                : undefined}
            assetPluginPreviews={pendingTrailPhotoCandidates.map((candidate) => ({
                pluginId: candidate.pluginId,
                assetId: candidate.assetId,
                filename: candidate.originalFileName,
                takenAt: candidate.takenAt,
                thumbnailUrl: candidate.thumbnailUrl,
            }))}
            onassetplugindelete={removePendingTrailPhotoCandidate}
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
                    effectiveControls={routeEffectiveControls}
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
                    provenanceMismatch={showRoutingProvenanceMismatch}
                    onRerouteExisting={recalculateEntireRoute}
                    onKeepExisting={keepExistingRoutingSegments}
                    routingEngines={routeEditorRoutingEngines}
                    onRoutingEngineChange={selectRouteEditorRoutingEngine}
                    viaAvailable={viaRoutingAvailable}
                    {routingEnabled}
                    bind:showWaypoints={showWaypointsWhileDrawing}
                ></RouteEditor>
            </div>
        {/if}
        <div id="trail-map">
            <MapWithElevationMaplibre
                trails={mapTrail}
                previewTrails={routingVariantMapTrails}
                focusedTrailId={routingVariantMapFocusId}
                elevationPreviewGpx={routingVariantElevationPreview}
                waypoints={$formData.expand?.waypoints_via_trail}
                drawing={drawingActive}
                displayWaypoints={!drawingActive || showWaypointsWhileDrawing}
                showTerrain={true}
                autoGeolocateOnDrawing={page.params.id === "new"}
                onmarkerdragend={moveMarker}
                activeTrail={0}
                bind:map
                bind:this={mapWithElevation}
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
<WaypointModal
    bind:this={waypointModal}
    onsave={saveWaypoint}
    assetPluginActive={data.assetPluginActive}
    {assetPluginIds}
    assetPluginProviders={data.assetPluginProviders}
></WaypointModal>
<WaypointMergeModal
    merge={pendingWaypointMerge}
    bind:this={waypointMergeModal}
    oncreate={createPendingWaypointAnyway}
    onmerge={addPendingWaypointToExisting}
    oncancel={cancelPendingWaypointMerge}
></WaypointMergeModal>
<AssetWaypointModal
    bind:this={assetWaypointModal}
    trailId={$formData.id ?? ""}
    category={$formData.category}
    trailData={routingStore.route.toString()}
    {assetPluginIds}
    assetPluginProviders={data.assetPluginProviders}
    existingWaypoints={$formData.expand?.waypoints_via_trail ?? []}
    onsave={onAssetPluginImport}
></AssetWaypointModal>
<PhotoLibraryPickerModal
    bind:this={trailPhotoLibraryModal}
    id="trail-photo-library-modal"
    trailId={$formData.id ?? ""}
    trailData={routingStore.route.toString()}
    {assetPluginIds}
    assetPluginProviders={data.assetPluginProviders}
    onselect={onTrailPhotoLibrarySelect}
/>
<SummitLogModal
    bind:this={summitLogModal}
    onsave={(log) => saveSummitLog(log)}
    {assetPluginIds}
    assetPluginProviders={data.assetPluginProviders}
    trailId={$formData.id ?? ""}
    trailData={routingStore.route.toString()}
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
<ConfirmModal
    id="round-trip-replace-modal"
    title={$_("routing-round-trip-replace-title")}
    text={$_("routing-round-trip-replace-confirm")}
    action="routing-round-trip-replace-action"
    deny="cancel"
    bind:this={roundTripReplaceModal}
    onconfirm={confirmRoundTripReplacement}
    oncancel={() => (pendingRoundTripRequest = undefined)}
></ConfirmModal>
<ConfirmModal
    id="publish-linked-photos-modal"
    title={$_("publish-trail-confirm-title")}
    text={$_("publish-trail-confirm-text", {
        values: { count: pendingLinkedPhotoCount },
    })}
    action="publish-and-copy"
    deny="cancel"
    bind:this={publishConfirmModal}
    onconfirm={confirmPublishWithLinkedPhotos}
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
