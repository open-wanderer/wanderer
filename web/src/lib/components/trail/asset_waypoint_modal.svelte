<script lang="ts">
    import { photoLibraryPluginLinks, type PhotoLibraryCandidate } from "$lib/models/photo_library";
    import type { PluginProvider } from "$lib/models/plugin_provider";
    import { Waypoint } from "$lib/models/waypoint";
    import { _ } from "svelte-i18n";
    import PhotoLibraryPickerModal from "../photo/photo_library_picker_modal.svelte";

    interface Props {
        trailId: string;
        category?: string;
        trailData?: string;
        trailPolyline?: string;
        assetPluginIds?: string[];
        assetPluginProviders?: PluginProvider[];
        existingWaypoints?: Waypoint[];
        onsave?: (waypoints: Waypoint[]) => void;
    }

    let {
        trailId,
        category,
        trailData,
        trailPolyline,
        assetPluginIds = [],
        assetPluginProviders = [],
        existingWaypoints = [],
        onsave,
    }: Props = $props();

    interface WaypointPhotoCluster {
        lat: number;
        lon: number;
        waypoint?: string;
        name?: string;
        photos: string[];
    }

    interface WaypointPhotoClusterResponse {
        clusters: WaypointPhotoCluster[];
    }

    interface CandidateClusterInput {
        id: string;
        lat: number;
        lon: number;
        candidate: PhotoLibraryCandidate;
    }

    let picker: PhotoLibraryPickerModal = $state()!;

    export function openModal() {
        picker.openModal();
    }

    async function importSelected(candidates: PhotoLibraryCandidate[]) {
        const waypoints = await candidatesToWaypoints(candidates);
        onsave?.(waypoints);
    }

    async function candidatesToWaypoints(candidates: PhotoLibraryCandidate[]) {
        if (!candidates.length) {
            return [];
        }

        const inputs = candidates
            .map(candidateClusterInput)
            .filter((input): input is CandidateClusterInput => Boolean(input));
        const inputsById = new Map(inputs.map((input) => [input.id, input]));

        if (!inputs.length) {
            return [];
        }

        const response = await clusterSelectedCandidates(inputs);
        return response.clusters
            .map((cluster) => {
                const clusterCandidates = cluster.photos
                    .map((id) => inputsById.get(id)?.candidate)
                    .filter(isPhotoLibraryCandidate);
                if (!clusterCandidates.length) {
                    return undefined;
                }
                return clusterToWaypoint(cluster, clusterCandidates);
            })
            .filter((waypoint): waypoint is Waypoint => Boolean(waypoint));
    }

    async function clusterSelectedCandidates(inputs: CandidateClusterInput[]): Promise<WaypointPhotoClusterResponse> {
        const r = await fetch("/api/v1/waypoint/cluster", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                category,
                resolveNames: true,
                photos: inputs.map((input) => ({
                    id: input.id,
                    lat: input.lat,
                    lon: input.lon,
                })),
                waypoints: existingWaypoints
                    .filter((waypoint) => waypoint.id)
                    .map((waypoint) => ({
                        id: waypoint.id!,
                        lat: waypoint.lat,
                        lon: waypoint.lon,
                    })),
            }),
        });
        if (!r.ok) {
            const data = await r.json().catch(() => ({}));
            throw new Error(data.message ?? $_("error-generic"));
        }
        return await r.json();
    }

    function candidateClusterInput(candidate: PhotoLibraryCandidate): CandidateClusterInput | undefined {
        const lat = waypointLat(candidate);
        const lon = waypointLon(candidate);
        if (!hasCoordinate(lat) || !hasCoordinate(lon)) {
            return;
        }
        return {
            id: candidateKey(candidate),
            lat,
            lon,
            candidate,
        };
    }

    function clusterToWaypoint(cluster: WaypointPhotoCluster, candidates: PhotoLibraryCandidate[]) {
        const existing = cluster.waypoint
            ? existingWaypoints.find((waypoint) => waypoint.id === cluster.waypoint)
            : undefined;
        const wandererCandidates = candidates.filter((candidate) => candidate.source === "wanderer");
        const pluginCandidates = candidates.filter((candidate) => candidate.source !== "wanderer");
        const waypoint = new Waypoint(existing?.lat ?? cluster.lat, existing?.lon ?? cluster.lon, {
            id: existing?.id,
            name: existing ? existing.name : cluster.name,
            icon: existing?.icon ?? "camera",
            photos: Array.from(new Set([
                ...(existing?.photos ?? []),
                ...wandererCandidates
                    .map((candidate) => candidate.thumbnailUrl)
                    .filter((url): url is string => Boolean(url)),
            ])),
        });
        waypoint.description = existing?.description ?? waypoint.description;
        waypoint.distance_from_start = existing?.distance_from_start ?? Math.min(
            ...candidates.map((candidate) => candidate.distanceFromStart ?? 0),
        );
        waypoint._assetLinks = wandererCandidates.map((candidate) => candidate.assetId);
        waypoint._assetCandidates = pluginCandidates.map((candidate) => ({
            pluginId: candidate.pluginId ?? candidate.providerId ?? "",
            assetId: candidate.assetId,
            lat: candidate.lat,
            lon: candidate.lon,
            originalFileName: candidate.originalFileName,
            takenAt: candidate.takenAt,
        }));
        waypoint._assetPluginLinks = photoLibraryPluginLinks(pluginCandidates);
        return waypoint;
    }

    function candidateKey(candidate: PhotoLibraryCandidate) {
        return `${candidate.source ?? "plugin"}:${candidate.pluginId ?? candidate.providerId ?? ""}:${candidate.assetId}`;
    }

    function waypointLat(candidate: PhotoLibraryCandidate) {
        return candidate.pointLat ?? candidate.lat;
    }

    function waypointLon(candidate: PhotoLibraryCandidate) {
        return candidate.pointLon ?? candidate.lon;
    }

    function hasCoordinate(value: unknown): value is number {
        return typeof value === "number" && Number.isFinite(value);
    }

    function isPhotoLibraryCandidate(candidate: PhotoLibraryCandidate | undefined): candidate is PhotoLibraryCandidate {
        return Boolean(candidate);
    }
</script>

<PhotoLibraryPickerModal
    bind:this={picker}
    id="asset-waypoint-modal"
    {trailId}
    {trailData}
    {trailPolyline}
    {assetPluginIds}
    {assetPluginProviders}
    title={$_("waypoints-from-photos")}
    actionLabel={$_("create-waypoints")}
    onselect={importSelected}
/>
