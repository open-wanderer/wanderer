<script lang="ts">
    import { goto } from "$app/navigation";
    import type { List } from "$lib/models/list";
    import type { Trail } from "$lib/models/trail";
    import {
        lists_add_trail,
        lists_index,
        lists_remove_trail,
    } from "$lib/stores/list_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { trails_delete, trails_show, fetchGPX, trails_update } from "$lib/stores/trail_store";
    import { currentUser } from "$lib/stores/user_store";
    import { getFileURL, saveAs } from "$lib/util/file_util";
    import { trail2gpx } from "$lib/util/gpx_util";
    import { gpx } from "$lib/vendor/toGeoJSON/toGeoJSON";
    import JSZip from "jszip";
    import { _ } from "svelte-i18n";
    import Dropdown, { type DropdownItem } from "../base/dropdown.svelte";
    import ConfirmModal from "../confirm_modal.svelte";
    import ListSelectModal from "../list/list_select_modal.svelte";
    import TrailExportModal from "./trail_export_modal.svelte";
    import TrailShareModal from "./trail_share_modal.svelte";
    import { handleFromRecordWithIRI } from "$lib/util/activitypub_util";
    import type { Snippet } from "svelte";
    import { page } from "$app/state";
	import { SummitLog } from "$lib/models/summit_log";
    import { summit_logs_create } from "$lib/stores/summit_log_store";
    import type { Comment as TrailComment } from "$lib/models/comment";
    import { comments_create, comments_index } from "$lib/stores/comment_store";
    import TrailMergeModal from "./trail_merge_modal.svelte";
    import type { MergeSettings } from "./trail_merge_modal.svelte";
    import { processMergeQueue, mergeStore, type Merge } from "$lib/stores/trail_merge_store.svelte";
    import MergeDialog from "$lib/components/trail/trail_merge_dialog.svelte";
  import { tags_create } from "$lib/stores/tag_store";
  import { Tag } from "$lib/models/tag";
  import { trail_like_create } from "$lib/stores/trail_like_store";
  import { TrailLike } from "$lib/models/trail_like";

    interface Props {
        trails?: Set<Trail> | undefined;
        mode: "overview" | "map" | "list" | "multi-select";
        toggle?: Snippet<[any]>;
        onDelete?: () => void;
        onShare?: () => void;
        onMerge?: () => void;
    }

    let { trails, mode, toggle, onDelete, onShare, onMerge }: Props = $props();

    let confirmModal: ConfirmModal;
    let listSelectModal: ListSelectModal;
    let trailExportModal: TrailExportModal;
    let trailShareModal: TrailShareModal;
    let trailMergeModal: TrailMergeModal;

    let lists: List[] = $state([]);

    function allowEdit(): boolean {
        return (
            hasTrail() &&
            !isMultiselectMode() &&
            Boolean($currentUser) &&
            (trail()!.expand?.author?.id === $currentUser?.actor ||
                trail()!.expand?.trail_share_via_trail?.some(
                    (s) => s.permission == "edit",
                ))!
        );
    }

    function allowMerge(): boolean {
        return (
            hasTrail() &&
            isMultiselectMode() &&
            Boolean($currentUser) &&
            (trail()!.expand?.author?.id === $currentUser?.actor ||
                trail()!.expand?.trail_share_via_trail?.some(
                    (s) => s.permission == "edit",
                ))!
        );
    }

    function dropdownItems(): DropdownItem[] {
        return [
            ...(!isMultiselectMode()
                ? [
                      mode == "overview" || mode == "multi-select"
                          ? {
                                text: $_("show-on-map"),
                                value: "show",
                                icon: "map",
                            }
                          : {
                                text: $_("show-in-overview"),
                                value: "show",
                                icon: "table-columns",
                            },
                  ]
                : []),
            ...(!isMultiselectMode()
                ? [{ text: $_("directions"), value: "direction", icon: "car" }]
                : []),
            ...(canExport()
                ? [
                      {
                          text: $_("export"),
                          value: "download",
                          icon: "download",
                      },
                  ]
                : []),
            ...(!isMultiselectMode()
                ? [{ text: $_("print"), value: "print", icon: "print" }]
                : []),
            ...(!isFromCurrentUser()
                ? []
                : [
                      {
                          text: $_("add-to-list"),
                          value: "list",
                          icon: "bookmark",
                      },
                  ]),
            ...(isMultiselectMode() || !isFromCurrentUser()
                ? []
                : [{ text: $_("share"), value: "share", icon: "share" }]),
            ...(allowEdit()
                ? [{ text: $_("edit"), value: "edit", icon: "pen" }]
                : []),
            ...(allowDelete()
                ? [{ text: $_("delete"), value: "delete", icon: "trash" }]
                : []),
            ...(allowMerge()
                ? [{ text: $_("link"), value: "merge", icon: "link" }]
                : []),
        ];
    }

    function isMultiselectMode(): boolean {
        return trails !== undefined && trails.size > 1;
    }

    function hasTrail(): boolean {
        return (
            trails !== undefined &&
            trails.size > 0 &&
            [...trails][0] !== undefined
        );
    }

    function hasGpx(): boolean {
        if (!hasTrail()) return false;

        for (const gTrail of trails!) {
            if (gTrail.gpx) return true;
        }

        return false;
    }

    function canExport(): boolean {
        return hasGpx();
    }

    function trailId(): string | undefined {
        return trail()?.id;
    }

    function getTrails(): Set<Trail> | undefined {
        return trails;
    }

    function trail(): Trail | undefined {
        return hasTrail() ? [...trails!][0] : undefined;
    }

    function isFromCurrentUser(uTrail?: Trail): boolean {
        if (!$currentUser) {
            return false;
        }
        if (uTrail !== undefined) {
            return uTrail.expand?.author?.id === $currentUser?.actor;
        } else if (trails !== undefined && trails.size > 0) {
            for (const sTrail of trails) {
                if (sTrail.expand?.author?.id === $currentUser?.actor) {
                    return true;
                }
            }
        }

        return false;
    }

    function allowDelete(): boolean {
        return isFromCurrentUser();
    }

    function allowDeleteTrail(dTrail?: Trail): boolean {
        return isFromCurrentUser(dTrail);
    }

    async function handleDropdownClick(item: { text: string; value: any }) {
        if (!trail()) {
            return;
        }

        const handle = page.params.handle ?? handleFromRecordWithIRI(trail())

        if (item.value == "show") {
            if (hasTrail()) {
                const url = mode == "overview" || mode == "multi-select"
                        ? `/map/trail/${handle}/${trailId()}`
                        : `/trail/view/${handle}/${trailId()}`
                
                goto(
                    url + '?' + page.url.searchParams
                );
            }
        } else if (item.value == "list") {
            lists = (
                await lists_index(
                    { q: "", author: $currentUser?.actor ?? "" },
                    1,
                    -1,
                )
            ).items;
            listSelectModal.openModal();
        } else if (item.value == "direction") {
            if (hasTrail()) {
                window
                    .open(
                        `https://www.openstreetmap.org/directions?to=${trail()!.lat},${trail()!.lon}`,
                        "_blank",
                    )
                    ?.focus();
            }
        } else if (item.value == "print") {
            if (hasTrail()) {
                goto(`/map/trail/${handle}/${trailId()}/print?${page.url.searchParams}`);
            }
        } else if (item.value == "share") {
            trailShareModal.openModal();
        } else if (item.value == "download") {
            trailExportModal.openModal();
        } else if (item.value == "edit") {
            if (hasTrail()) {
                goto(`/trail/edit/${trailId()}`);
            }
        } else if (item.value == "delete") {
            confirmModal.openModal();
        } else if (item.value == "merge") {
            trailMergeModal.openModal();
        }
    }

    async function mergeTrails(settings: MergeSettings) {  
        if (!trails || trails.size < 2) return;

        let trailTarget: Trail | undefined  = undefined;
        for (const mTrail of trails) {
            if (mTrail.expand?.summit_logs_via_trail && mTrail.expand?.summit_logs_via_trail.length > 0) {
                trailTarget = mTrail;
                break;
            }
        }

        if (!trailTarget) {
            trailTarget = [...trails][0];
        }

        if (!trailTarget.expand) {
            return;
        }

        for (const t of trails) {
            if (t.id === trailTarget.id) continue;

            const u: Merge = {
                trailTarget: trailTarget,
                trailSource: t,
                progress: 0,
                status: "enqueued",
                settings: settings,
                function: trails_merge
            };
            mergeStore.enqueuedMerges.push(u);
        }

        await processMergeQueue();
    }

    async function trails_merge(trailTarget: Trail, trailSource: Trail, settings: MergeSettings, onProgress?: (progress: number) => void) {
        
        let summit: SummitLog = new SummitLog(trailSource.date!, {
            id: undefined,
            text: trailSource.description,
            distance: trailSource.distance,
            elevation_loss: trailSource.elevation_loss,
            elevation_gain: trailSource.elevation_gain,
            duration: trailSource.duration,
            photos: []
        });

        summit.author = trailSource.expand?.author?.id!;
        summit.trail = trailTarget.id;

        let fileData = await trail2gpx(trailSource, $currentUser);
        const blob = new Blob([fileData], {
                    type: "application/json",
                });

        summit._gpx = new File([blob], trailSource.gpx ?? "summit.gpx");

        const mTrailWithDetails = await trails_show(trailSource.id!, undefined, undefined, false);

        // part 1
        onProgress?.(1/6)

        if (settings.photos) {
            for (const photo of mTrailWithDetails.photos) {
                const photoURL = getFileURL(mTrailWithDetails, photo);
                const photoBlob = await fetch(photoURL).then(
                    (response) => response.blob(),
                );
                const photoData = new File([photoBlob], photo);
                
                if (!summit._photos) summit._photos = [];
                summit._photos.push(photoData);
            }
        }

        await summit_logs_create(summit);

        // part 2
        onProgress?.(2/6)

        const origTrail = await trails_show(trailTarget.id!, undefined, undefined, false);
        const updatedTrail = await trails_show(trailTarget.id!, undefined, undefined, false);
        let trailUpdated = false;

        if (settings.tags && mTrailWithDetails.expand?.tags && mTrailWithDetails.expand?.tags.length > 0)  {

            if (updatedTrail.expand?.tags) {
                updatedTrail.expand.tags = [...updatedTrail.expand.tags, ...mTrailWithDetails.expand.tags];
            } else if (updatedTrail.expand) {
                updatedTrail.expand.tags = mTrailWithDetails.expand.tags;
            } else {
                updatedTrail.expand = { tags: mTrailWithDetails.expand.tags };
            }

            trailUpdated = true;
        }

        if (settings.likes && mTrailWithDetails.expand?.trail_like_via_trail && mTrailWithDetails.expand?.trail_like_via_trail.length > 0) {
            
            for (const trailLike of mTrailWithDetails.expand.trail_like_via_trail) {
                let newTrailLike = new TrailLike(trailLike.actor, trailTarget.id!);
                await trail_like_create(newTrailLike);
            }

            updatedTrail.like_count += mTrailWithDetails.like_count;
            trailUpdated = true;
        }

        if (trailUpdated) {
            await trails_update(origTrail, updatedTrail);
        }

        // part 3
        onProgress?.(3/6)

        if (settings.summitLog) {
            if (mTrailWithDetails.expand?.summit_logs_via_trail) {

                for (const sourceSummit of mTrailWithDetails.expand?.summit_logs_via_trail) {
                    
                    if (sourceSummit.date == mTrailWithDetails.date) continue;

                    let summit2: SummitLog = new SummitLog(sourceSummit.date!, {
                        id: undefined,
                        text: sourceSummit.text,
                        distance: sourceSummit.distance,
                        elevation_loss: sourceSummit.elevation_loss,
                        elevation_gain: sourceSummit.elevation_gain,
                        duration: sourceSummit.duration,
                        photos: []
                    });

                    if (!sourceSummit.expand?.gpx_data) {
                        const gpxData: string = await fetchGPX(sourceSummit as any, fetch);
                        summit2.expand = {
                            ...(summit2.expand ?? {}),
                            gpx_data: gpxData,
                        };
                    } else {
                        summit2.expand = {
                                ...(summit2.expand ?? {}),
                                gpx_data: sourceSummit.expand.gpx_data,
                        }
                    }

                    if (summit2.expand.gpx_data) {
                        var gpxBlob = new Blob([summit2.expand.gpx_data], {
                            type: 'text/plain'
                        });
                        const gpxFile = new File([gpxBlob], sourceSummit.gpx ?? "summit_log.gpx");
                        summit2._gpx = gpxFile;
                    }

                    for (const photo2 of sourceSummit.photos) {
                        const photoURL2 = getFileURL(sourceSummit, photo2);
                        const photoBlob2 = await fetch(photoURL2).then(
                            (response) => response.blob(),
                        );
                        const photoData2 = new File([photoBlob2], photo2);
                        
                        if (!summit2._photos) summit2._photos = [];
                        summit2._photos.push(photoData2);
                    }

                    summit2.author = sourceSummit.author;
                    summit2.trail = trailTarget.id;

                    await summit_logs_create(summit2);
                }
            }
        }

        // part 5
        onProgress?.(4/6)

        if (settings.comments) {
            let commentsFetchResponse = await fetchComments(mTrailWithDetails);
            if (commentsFetchResponse && mTrailWithDetails.expand?.comments_via_trail) {
                for (const comment of mTrailWithDetails.expand.comments_via_trail) {
                    let newComment: TrailComment = { 
                        text: comment.text, 
                        author: comment.author,  
                        trail: trailTarget.id!,
                        created: comment.created,
                        updated: comment.updated,
                        iri: comment.iri,
                        expand: comment.expand
                    };

                    await comments_create(newComment);
                }
            }
        }

        // part 6
        onProgress?.(5/6)

        if (settings.delete) {
            await trails_delete(trailSource);
            onMerge?.();
        }

        // part 7
        onProgress?.(1)
    }


    async function fetchComments(cTrail: Trail) : Promise<boolean> {
        const trailId = cTrail.iri ? cTrail.iri : cTrail.id!;
        try {
            let trailComments = await comments_index(trailId);
            if (cTrail.expand && trailComments && trailComments.length > 0) {
                cTrail.expand.comments_via_trail = trailComments;
            }
            return true;
        } catch (e) {
            return false;
        }
    }

    async function exportTrails(exportSettings: {
        fileFormat: "gpx" | "json";
        photos: boolean;
        summitLog: boolean;
    }) {
        if (trails !== undefined && trails.size > 0) {
            for (const cTrail of trails) {
                await doExportTrail(exportSettings, cTrail);
            }
        }
    }

    async function doExportTrail(
        exportSettings: {
            fileFormat: "gpx" | "json";
            photos: boolean;
            summitLog: boolean;
        },
        eTrail: Trail,
    ) {
        try {
            if (eTrail !== undefined) {
                let fileData: string = await trail2gpx(eTrail, $currentUser);
                if (exportSettings.fileFormat == "json") {
                    fileData = JSON.stringify(
                        gpx(
                            new DOMParser().parseFromString(
                                fileData,
                                "application/gpx+xml" as any,
                            ),
                        ),
                    );
                }
                if (!exportSettings.photos && !exportSettings.summitLog) {
                    const blob = new Blob([fileData], {
                        type:
                            exportSettings.fileFormat == "json"
                                ? "application/json"
                                : "application/gpx+xml",
                    });
                    saveAs(blob, `${eTrail.name}.${exportSettings.fileFormat}`);
                } else {
                    const zip = new JSZip();
                    zip.file(
                        `${eTrail.name}.${exportSettings.fileFormat}`,
                        fileData,
                    );
                    if (exportSettings.photos) {
                        const photoFolder = zip.folder($_("photos"));
                        for (const photo of eTrail.photos) {
                            const photoURL = getFileURL(eTrail, photo);
                            const photoBlob = await fetch(photoURL).then(
                                (response) => response.blob(),
                            );
                            const photoData = new File([photoBlob], photo);
                            photoFolder?.file(photo, photoData, {
                                base64: true,
                            });
                        }
                    }
                    if (exportSettings.summitLog) {
                        let summitLogString = "";
                        for (const summitLog of eTrail.expand
                            ?.summit_logs_via_trail ?? []) {
                            summitLogString += `${summitLog.date},${summitLog.text}\n`;
                        }
                        zip.file(
                            `${eTrail.name} - ${$_("summit-book")}.csv`,
                            summitLogString,
                        );
                    }
                    const blob = await zip.generateAsync({ type: "blob" });
                    saveAs(blob, `${eTrail.name}.zip`);
                }
            }
        } catch (e) {
            console.error(e);
            show_toast({
                type: "error",
                icon: "close",
                text: $_("error-exporting-trail"),
            });
        }
    }

    async function deleteTrails() {
        if (hasTrail()) {
            for (const dTrail of trails!) {
                await doDeleteTrail(dTrail);
            }

            onDelete?.();
        }
    }

    async function doDeleteTrail(dTrail: Trail) {
        if (dTrail === undefined) return;

        if (!allowDeleteTrail(dTrail)) return;

        await trails_delete(dTrail);
    }

    async function handleShareUpdate() {
        onShare?.();
    }

    async function handleListSelection(list: List) {
        try {
            let deleted = false;
            let multiple = false;

            if (hasTrail()) {
                multiple = true;
                for (const lTrail of trails!) {
                    if (await doHandleListSelection(list, lTrail)) {
                        deleted = true;
                    }
                }
            }

            if (deleted) {
                show_toast({
                    type: "success",
                    icon: "check",
                    text: multiple
                        ? `${$_("removed-trails-from")} "${list.name}"`
                        : `${$_("removed-trail-from")} "${list.name}"`,
                });
            } else {
                show_toast({
                    type: "success",
                    icon: "check",
                    text: multiple
                        ? `${$_("added-trails-to")} "${list.name}"`
                        : `${$_("added-trail-to")} "${list.name}"`,
                });
            }
        } catch (e) {
            console.error(e);

            show_toast({
                type: "error",
                icon: "close",
                text: "Error adding trail to list.",
            });
        }
    }

    async function doHandleListSelection(
        list: List,
        lTrail: Trail,
    ): Promise<boolean> {
        if (list.trails?.includes(lTrail.id!)) {
            if (listContainsAllTrails(list)) {
                await lists_remove_trail(list, lTrail);
                return true;
            }
        } else {
            await lists_add_trail(list, lTrail);
        }

        return false;
    }
    function listContainsAllTrails(list: List): boolean {
        if (trails === undefined) {
            return false;
        } else if (list.trails !== undefined) {
            for (const lTrail of trails) {
                if (!list.trails!.includes(lTrail.id!)) return false;
            }

            return true;
        }

        return false;
    }
</script>

<Dropdown
    items={dropdownItems()}
    onchange={(item) => handleDropdownClick(item)}
>
    {#snippet children({ toggleMenu: openDropdown })}
        {#if toggle}{@render toggle({
                toggleMenu: openDropdown,
            })}{:else if mode == "multi-select"}
            <button
                aria-label="Open dropdown"
                class="btn-primary flex-shrink-0 !font-medium"
                onclick={openDropdown}
            >
                <span
                    >{trails?.size}
                    {$_("selected")}
                    <i class="fa fa-caret-down ml-1"></i></span
                >
            </button>
        {:else}
            <button
                aria-label="Open dropdown"
                class=" btn-primary !rounded-full h-12 w-12"
                onclick={openDropdown}
            >
                <i class="fa fa-ellipsis-vertical"></i>
            </button>
        {/if}
    {/snippet}
</Dropdown>

<ConfirmModal
    text={$_("delete-trail-confirm")}
    bind:this={confirmModal}
    onconfirm={deleteTrails}
></ConfirmModal>
<ListSelectModal
    {lists}
    trails={getTrails()}
    bind:this={listSelectModal}
    onchange={(list) => handleListSelection(list)}
></ListSelectModal>
<TrailExportModal
    bind:this={trailExportModal}
    onexport={(settings) => exportTrails(settings)}
></TrailExportModal>
<TrailShareModal
    trail={trail()}
    onsave={handleShareUpdate}
    bind:this={trailShareModal}
></TrailShareModal>
<TrailMergeModal
    bind:this={trailMergeModal}
    onmerge={(settings) => mergeTrails(settings)}
></TrailMergeModal>

<MergeDialog/>