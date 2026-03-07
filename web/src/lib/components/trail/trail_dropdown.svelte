<script lang="ts">
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import type { List } from "$lib/models/list";
    import type { Trail } from "$lib/models/trail";
    import { categories } from "$lib/stores/category_store.js";
    import {
        lists_add_trail,
        lists_index,
        lists_remove_trail,
    } from "$lib/stores/list_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { trails_delete, trails_update } from "$lib/stores/trail_store";
    import { currentUser } from "$lib/stores/user_store";
    import { handleFromRecordWithIRI } from "$lib/util/activitypub_util";
    import { getFileURL, saveAs } from "$lib/util/file_util";
    import { trail2gpx } from "$lib/util/gpx_util";
    import { gpx } from "$lib/vendor/toGeoJSON/toGeoJSON";
    import JSZip from "jszip";
    import type { Snippet } from "svelte";
    import { _ } from "svelte-i18n";
    import Dropdown, { type DropdownItem } from "../base/dropdown.svelte";
    import ConfirmModal from "../confirm_modal.svelte";
    import ListSearchModal from "../list/list_search_modal.svelte";
    import TrailExportModal from "./trail_export_modal.svelte";
    import TrailShareModal from "./trail_share_modal.svelte";

    interface Props {
        trails?: Set<Trail> | undefined;
        mode: "overview" | "map" | "list" | "multi-select";
        toggle?: Snippet<[any]>;
        onDelete?: () => void;
        onShare?: () => void;
        onUpdate?: () => void;
    }

    let { trails, mode, toggle, onDelete, onShare, onUpdate }: Props = $props();

    let confirmModal: ConfirmModal;
    let listSelectModal: ListSearchModal;
    let trailExportModal: TrailExportModal;
    let trailShareModal: TrailShareModal;

    let lists: List[] = $state([]);

    let loading: boolean = $state(false);

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

    function majorityOfSelectedTrailsArePublic(): boolean {
        if (trails === undefined || trails.size === 0) return false;

        if (!Boolean($currentUser)) return false;

        let publicCount = 0;

        for (const cTrail of trails) {
            if (cTrail.expand?.author === undefined) return false;
            if (
                cTrail.expand!.author!.id !== $currentUser?.actor &&
                !cTrail.expand?.trail_share_via_trail?.some(
                    (s) => s.permission == "edit",
                )
            ) {
                return false;
            }

            if (cTrail.public) {
                publicCount += 1;
            }
        }
        return publicCount >= trails.size / 2;
    }

    function allowCopy(): boolean {
        if ((trails?.size ?? 0) > 1) return false;

        return !isMultiselectMode();
    }

    function allowPublish(): boolean {
        if (mode !== "multi-select") return false;

        if (trails === undefined || trails.size === 0) return false;

        if (!Boolean($currentUser)) {
            return false;
        }

        for (const cTrail of trails) {
            if (cTrail.expand?.author === undefined) return false;
            if (
                cTrail.expand!.author!.id !== $currentUser?.actor &&
                !cTrail.expand?.trail_share_via_trail?.some(
                    (s) => s.permission == "edit",
                )
            ) {
                return false;
            }
        }

        return true;
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
                ? [
                      {
                          text: $_("directions"),
                          value: "direction",
                          icon: "car",
                      },
                  ]
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
                ? [
                      {
                          text: $_("print"),
                          value: "print",
                          icon: "print",
                      },
                  ]
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
                : [
                      {
                          text: $_("share"),
                          value: "share",
                          icon: "share",
                      },
                  ]),
            ...(allowCopy()
                ? [
                      {
                          text: $_("duplicate"),
                          value: "copy",
                          icon: "copy",
                      },
                  ]
                : []),
            ...(allowPublish()
                ? [
                      {
                          text: `${majorityOfSelectedTrailsArePublic() ? $_("set-private") : $_("set-public")}`,
                          value: "publish",
                          icon: majorityOfSelectedTrailsArePublic()
                              ? "lock"
                              : "globe",
                      },
                  ]
                : []),
            ...(allowEdit()
                ? [
                      {
                          text: $_("edit"),
                          value: "edit",
                          icon: "pen",
                      },
                  ]
                : []),
            ...(allowDelete()
                ? [
                      {
                          text: $_("delete"),
                          value: "delete",
                          icon: "trash",
                      },
                  ]
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

        const handle = page.params.handle ?? handleFromRecordWithIRI(trail());

        const ddVal = item.value as string;

        if (ddVal == "show") {
            if (hasTrail()) {
                const url =
                    mode == "overview" || mode == "multi-select"
                        ? `/map/trail/${handle}/${trailId()}`
                        : `/trail/view/${handle}/${trailId()}`;

                goto(url + "?" + page.url.searchParams);
            }
        } else if (ddVal == "list") {
            lists = (
                await lists_index(
                    { q: "", author: $currentUser?.actor ?? "" },
                    1,
                    -1,
                )
            ).items;
            listSelectModal.openModal();
        } else if (ddVal == "direction") {
            if (hasTrail()) {
                window
                    .open(
                        `https://www.openstreetmap.org/directions?to=${trail()!.lat},${trail()!.lon}`,
                        "_blank",
                    )
                    ?.focus();
            }
        } else if (ddVal == "print") {
            if (hasTrail()) {
                goto(
                    `/map/trail/${handle}/${trailId()}/print?${page.url.searchParams}`,
                );
            }
        } else if (ddVal == "share") {
            trailShareModal.openModal();
        } else if (ddVal == "download") {
            trailExportModal.openModal();
        } else if (ddVal == "edit") {
            if (hasTrail()) {
                goto(`/trail/edit/${trailId()}`);
            }
        } else if (ddVal == "copy") {
            if (hasTrail()) {
                goto("/trail/edit/new?orig=" + trail()?.id);
            }
        } else if (ddVal == "publish") {
            updateTrailsVisibility();
        } else if (ddVal == "delete") {
            confirmModal.openModal();
        }
    }

    async function updateTrailsVisibility() {
        const newVisibility = !majorityOfSelectedTrailsArePublic();

        loading = true;
        for (const cTrail of trails ?? []) {
            if (!cTrail) continue;

            if (!cTrail.expand?.author?.id) continue;

            const origTrail: Trail = {
                ...cTrail,
                author: cTrail.expand!.author!.id,
            };
            const updatedTrail: Trail = {
                ...origTrail,
                public: newVisibility,
            };

            try {
                await trails_update(
                    origTrail,
                    updatedTrail,
                    undefined,
                    undefined,
                    ["tags", "category"],
                );
            } catch (e) {
                console.error(e);

                show_toast({
                    type: "error",
                    icon: "close",
                    text: `${$_("error-saving-trail")}: ${cTrail.name}`,
                });
            }
        }

        loading = false;
        onUpdate?.();
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
            loading = true;

            for (const dTrail of trails!) {
                await doDeleteTrail(dTrail);
            }
            loading = false;

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
            })}
        {:else if loading}
            <div class:w-16={isMultiselectMode()}></div>
            <div class="spinner light:spinner-dark"></div>
        {:else if mode == "multi-select"}
            <button
                aria-label="Open dropdown"
                class="btn-primary shrink-0 font-medium!"
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
                class="btn-primary rounded-full! h-12 w-12"
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
<ListSearchModal
    {lists}
    trails={getTrails()}
    bind:this={listSelectModal}
    onchange={(list) => handleListSelection(list)}
></ListSearchModal>
<TrailExportModal
    bind:this={trailExportModal}
    onexport={(settings) => exportTrails(settings)}
></TrailExportModal>
<TrailShareModal
    trail={trail()}
    onsave={handleShareUpdate}
    bind:this={trailShareModal}
></TrailShareModal>
