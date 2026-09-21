<script lang="ts">
    import { page } from "$app/state";
    import emptyStateUploadDark from "$lib/assets/svgs/empty_states/empty_state_upload_dark.svg";
    import emptyStateUploadLight from "$lib/assets/svgs/empty_states/empty_state_upload_light.svg";
    import Button from "$lib/components/base/button.svelte";
    import Toggle from "$lib/components/base/toggle.svelte";
    import TrailExportModal from "$lib/components/trail/trail_export_modal.svelte";
    import type { Settings } from "$lib/models/settings";
    import { settings_update } from "$lib/stores/settings_store";
    import { theme } from "$lib/stores/theme_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { fetchGPX, trails_index, trails_upload } from "$lib/stores/trail_store";
    import { processUploadQueue, uploadStore, type Upload } from "$lib/stores/upload_store.svelte";
    import { currentUser } from "$lib/stores/user_store";
    import { getFileURL, saveAs } from "$lib/util/file_util";
    import { trail2gpx } from "$lib/util/gpx_util";
    import { gpx } from "$lib/vendor/toGeoJSON/toGeoJSON";
    import JSZip from "jszip";
    import { untrack } from "svelte";
    import { _ } from "svelte-i18n";

    let exportModal: TrailExportModal;

    let offerUpload: boolean = $state(false);

    let settings = $derived(page.data.settings as Settings);
    let displayedSettingsId = $state<string | null | undefined>(page.data.settings?.id);
    const initialDuplicateCheck = page.data.settings?.uploadDuplicateCheck;
    let includePublic = $state(initialDuplicateCheck?.includePublic === true);
    let includeShared = $state(initialDuplicateCheck?.includeShared === true);
    let savingDuplicateCheck = $state(false);
    let duplicateCheckSaveRevision = 0;

    $effect(() => {
        const settingsId = settings?.id;
        const incoming = {
            includePublic: settings?.uploadDuplicateCheck?.includePublic === true,
            includeShared: settings?.uploadDuplicateCheck?.includeShared === true,
        };

        untrack(() => {
            if (settingsId !== displayedSettingsId) {
                displayedSettingsId = settingsId;
                duplicateCheckSaveRevision += 1;
                savingDuplicateCheck = false;
            } else if (savingDuplicateCheck) {
                return;
            }

            includePublic = incoming.includePublic;
            includeShared = incoming.includeShared;
        });
    });

    async function updateDuplicateCheck() {
        const settingsId = settings?.id;
        if (savingDuplicateCheck || !settingsId || settingsId !== displayedSettingsId) {
            return;
        }

        const uploadDuplicateCheck = { includePublic, includeShared };
        const saveRevision = ++duplicateCheckSaveRevision;
        savingDuplicateCheck = true;
        try {
            await settings_update({
                id: settingsId,
                uploadDuplicateCheck,
            });
        } catch (e) {
            console.error(e);
            if (saveRevision !== duplicateCheckSaveRevision || settings?.id !== settingsId) {
                return;
            }
            show_toast({
                type: "error",
                icon: "close",
                text: $_("error-saving-settings"),
            });
        } finally {
            if (saveRevision === duplicateCheckSaveRevision && settings?.id === settingsId) {
                includePublic = settings?.uploadDuplicateCheck?.includePublic === true;
                includeShared = settings?.uploadDuplicateCheck?.includeShared === true;
                savingDuplicateCheck = false;
            }
        }
    }

    function openFileBrowser() {
        document.getElementById("file-input")!.click();
    }

    function handleDragOver(e: DragEvent) {
        e.preventDefault();
        offerUpload = true;
    }

    function handleDragLeave(e: DragEvent) {
        offerUpload = false;
    }

    function handleDrop(e: DragEvent) {
        e.preventDefault();
        offerUpload = false;
        handleFileSelection(e.dataTransfer?.files);
    }

    async function handleFileSelection(files?: FileList | null) {
        if (!files) {
            files = (document.getElementById("file-input") as HTMLInputElement)
                .files;
        }

        if (!files) {
            return;
        }

        for (let i = 0; i < files.length; i++) {
            const u: Upload = {
                file: files[i],
                progress: 0,
                status: "enqueued",
                function: trails_upload
            };
            uploadStore.enqueuedUploads.push(u);
        }

        await processUploadQueue();
    }

    async function exportTrails(exportSettings: {
        fileFormat: "gpx" | "json";
        photos: boolean;
        summitLog: boolean;
    }) {
        try {
            const trails = await trails_index(-1);

            const zip = new JSZip();

            for (const trail of trails) {
                const gpxData: string = await fetchGPX(trail);
                if (!trail.expand) {
                    (trail as any).expand = {};
                }
                trail.expand!.gpx_data = gpxData;
                const trailFolder = zip.folder(`${trail.name}`);
                let fileData: string = await trail2gpx(trail, $currentUser);
                if (exportSettings.fileFormat == "json") {
                    fileData = JSON.stringify(
                        gpx(
                            new DOMParser().parseFromString(
                                fileData,
                                "text/xml",
                            ),
                        ),
                    );
                }
                trailFolder?.file(
                    `${trail.name}.${exportSettings.fileFormat}`,
                    fileData,
                );
                if (exportSettings.photos) {
                    const photoFolder = trailFolder?.folder($_("photos"));
                    for (const photo of trail.photos ?? []) {
                        const photoURL = getFileURL(trail, photo);
                        const photoBlob = await fetch(photoURL).then(
                            (response) => response.blob(),
                        );
                        const photoData = new File([photoBlob], photo);
                        photoFolder?.file(photo, photoData, { base64: true });
                    }
                }
                if (exportSettings.summitLog) {
                    let summitLogString = "";
                    for (const summitLog of trail.expand?.summit_logs_via_trail ?? []) {
                        summitLogString += `${summitLog.date},${summitLog.text}\n`;
                    }
                    trailFolder?.file(
                        `${trail.name} - ${$_("summit-book")}.csv`,
                        summitLogString,
                    );
                }
            }

            const blob = await zip.generateAsync({ type: "blob" });
            saveAs(blob, `wanderer-export-${Date.now()}.zip`);
        } catch (e) {
            console.error(e);
            show_toast({
                type: "error",
                icon: "close",
                text: $_("error-exporting-trail"),
            });
        }
    }
</script>

<svelte:head>
    <title>{$_("settings")} | wanderer</title>
</svelte:head>

<div class="space-y-6">
    <h3 class="text-2xl font-semibold">{$_("import")}</h3>
    <hr class="mt-4 mb-6 border-input-border" />
    <section aria-labelledby="upload-duplicate-check-heading" class="space-y-3">
        <h4 id="upload-duplicate-check-heading" class="text-lg font-medium">
            {$_("upload-duplicate-check-title")}
        </h4>
        <p class="text-sm text-gray-500">
            {$_("upload-duplicate-check-hint")}
        </p>
        <div>
            <Toggle
                name="uploadDuplicateCheck.includePublic"
                label={$_("upload-duplicate-check-public")}
                bind:value={includePublic}
                disabled={savingDuplicateCheck || !settings?.id}
                onchange={updateDuplicateCheck}
            />
            <Toggle
                name="uploadDuplicateCheck.includeShared"
                label={$_("upload-duplicate-check-shared")}
                bind:value={includeShared}
                disabled={savingDuplicateCheck || !settings?.id}
                onchange={updateDuplicateCheck}
            />
        </div>
    </section>
    <button
        class="drop-area relative h-64 w-full p-4 border border-content border-dashed rounded-xl flex items-center justify-center text-gray-500 bg-background cursor-pointer hover:bg-menu-item-background-hover focus:bg-menu-item-background-focus transition-colors"
        class:border-2={offerUpload}
        onclick={openFileBrowser}
        ondragover={handleDragOver}
        ondragleave={handleDragLeave}
        ondrop={handleDrop}
    >
        <div class="">
            <img
                class="rounded-full aspect-square mx-auto"
                src={$theme === "light"
                    ? emptyStateUploadLight
                    : emptyStateUploadDark}
                alt="Empty State showing a wanderer going into the distance"
            />
            {$_("import-hint")}
        </div>
    </button>

    <input
        type="file"
        id="file-input"
        accept=".gpx,.GPX,.tcx,.TCX,.kml,.KML,.fit,.FIT"
        multiple={true}
        style="display: none;"
        onchange={() => handleFileSelection()}
    />
    <h3 class="text-2xl font-semibold">{$_("export")}</h3>
    <hr class="mt-4 mb-6 border-input-border" />
    <Button secondary={true} onclick={() => exportModal.openModal()}
        >{$_("export-all-trails")}</Button
    >
</div>

<TrailExportModal bind:this={exportModal} onexport={exportTrails}
></TrailExportModal>
