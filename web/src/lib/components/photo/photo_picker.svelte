<script lang="ts">
    import type { Asset } from "#lib/models/asset.ts";
    import { assetFileCoordinatesFor, markAssetFileMetadata } from "#lib/stores/asset_store.ts";
    import { show_toast } from "#lib/stores/toast_store.svelte.ts";
    import { assetIdFromPhotoURL, expandedPhotoAssets, isGeneratedRoutePreviewAsset } from "#lib/util/asset_link_util.ts";
    import { ASSET_UPLOAD_ACCEPT, convertAssetUploadFile, isAssetUploadImage, isSupportedAssetUpload } from "#lib/util/asset_upload_util.ts";
    import { extractPhotoExifMetadata, type GPSCoordinates, type PhotoExifMetadata } from "#lib/util/exif_util.ts";
    import { getFileURL } from "#lib/util/file_util.ts";
    import { photoFilePreview } from "#lib/util/photo_preview_util.ts";
    import { _ } from "svelte-i18n";
    import PhotoCard from "./photo_card.svelte";

    interface Props {
        id: string;
        photos?: string[];
        photoFiles: File[] | undefined;
        parent: { [key: string]: any };
        thumbnail?: number;
        showThumbnailControls?: boolean;
        showExifControls?: boolean;
        maxSizeBytes?: number;
        onexif?: (src: string, coordinates?: GPSCoordinates) => void | Promise<void>;
    }

    let {
        id,
        photos = $bindable(undefined),
        photoFiles = $bindable(),
        parent,
        thumbnail = $bindable(0),
        showThumbnailControls = true,
        showExifControls = false,
        maxSizeBytes = 20971520,
        onexif,
    }: Props = $props();

    let photoPreviews: string[] = $state([]);

    // A generated route preview is only the thumbnail of a trail without other
    // photos (thumbnailPhotoIndex in asset_link_util.ts). While the editor
    // holds a real photo, saved or pending, the preview is not offered as one.
    let routePreviews = $derived(routePreviewPhotos(photos ?? [], expandedPhotoAssets(parent, "trail")));
    let hasRealPhoto = $derived(
        (photos ?? []).some((photo) => !routePreviews.has(photo)) ||
            Boolean(photoFiles?.length),
    );

    function routePreviewPhotos(shown: string[], assets: Asset[]): Set<string> {
        const previewIds = new Set(assets.filter((asset) => isGeneratedRoutePreviewAsset(asset)).map((asset) => asset.id));
        return new Set(shown.filter((photo) => previewIds.has(assetIdFromPhotoURL(photo) ?? "")));
    }

    $effect(() => {
        let canceled = false;
        Promise.all((photoFiles ?? []).map(photoFilePreview)).then((previews) => {
            if (!canceled) photoPreviews = previews;
        }).catch(() => {});
        return () => { canceled = true; };
    });

    let offerUpload: boolean = $state(false);

    function handlePhotoDragOver(e: DragEvent) {
        e.preventDefault();
        offerUpload = true;
    }

    function handlePhotoDragLeave() {
        offerUpload = false;
    }

    function handlePhotoDrop(e: DragEvent) {
        e.preventDefault();
        offerUpload = false;
        handlePhotoSelection(e.dataTransfer?.files);
    }

    function openPhotoBrowser() {
        document.getElementById(`${id}-photo-input`)!.click();
    }

    async function handlePhotoSelection(files?: FileList | null) {
        if (!files) {
            files = (
                document.getElementById(`${id}-photo-input`) as HTMLInputElement
            ).files;
        }

        if (!files) {
            return;
        }

        for (const file of files) {
            if (file.size > maxSizeBytes) {
                show_toast({
                    type: "error",
                    text: $_("file-too-big", {
                        values: { file: file.name, size: "20 MB" },
                    }),
                    icon: "close",
                });
                continue;
            }
            if (!isSupportedAssetUpload(file)) {
                show_toast({ type: "error", text: $_("photo-file-type-not-supported", { values: { file: file.name } }), icon: "close" });
                continue;
            }

            try {
                let metadata: PhotoExifMetadata | undefined;
                if (isAssetUploadImage(file)) {
                    metadata = await extractPhotoExifMetadata(file);
                }

                const photoFile = await convertAssetUploadFile(file);
                markAssetFileMetadata(photoFile, metadata);
                if (!photoFiles) {
                    photoFiles = [];
                }
                photoFiles = [...photoFiles, photoFile];
            } catch {
                show_toast({ type: "error", text: $_("photo-conversion-failed", { values: { file: file.name } }), icon: "close" });
            }
        }
    }

    function makePhotoThumbnail(index: number) {
        thumbnail = index;
    }

    function handlePhotoDelete(index: number) {
        if (thumbnail == index) {
            thumbnail = 0;
        } else if (thumbnail > index) {
            thumbnail -= 1;
        }

        if (index >= (photos?.length ?? 0)) {
            if (!photoFiles) {
                photoFiles = [];
            }
            const adjustedIndex = index - (photos?.length ?? 0);
            photoFiles = photoFiles.filter((_, i) => i !== adjustedIndex);
            photoPreviews = photoPreviews.filter((_, i) => i !== adjustedIndex);
        } else {
            photos = photos?.filter((_, i) => i !== index);
        }
    }

    function handlePhotoExif(index: number, src: string) {
        const fileIndex = index - (photos?.length ?? 0);
        const coordinates = fileIndex >= 0 ? assetFileCoordinatesFor(photoFiles?.[fileIndex]) : undefined;
        // Thumbnails carry no EXIF data, so saved photos are read from the original file.
        onexif?.(fileIndex >= 0 ? src : getFileURL(parent, (photos ?? [])[index]), coordinates);
    }

</script>

<div
    class="flex gap-x-4 max-w-full shrink-0 rounded-xl {offerUpload
        ? 'outline-dashed outline-input-border'
        : ''}"
    role="dialog"
    tabindex="0"
    ondragover={handlePhotoDragOver}
    ondragleave={handlePhotoDragLeave}
    ondrop={handlePhotoDrop}
>
    <button aria-label="Open photo browser" class="btn-secondary h-32 w-32 shrink-0 grow-0 basis-auto" type="button" onclick={openPhotoBrowser}><i class="fa fa-plus"></i></button>
    <input
        type="file"
        id="{id}-photo-input"
        accept={ASSET_UPLOAD_ACCEPT}
        multiple={true}
        style="display: none;"
        onchange={() => handlePhotoSelection()}
    />
    <div class="flex overflow-x-auto gap-x-3 w-full">
        {#each (photos ?? []).concat(photoPreviews) as photo, i}
            <div class="shrink-0 grow-0 basis-auto">
                <PhotoCard
                    src={i >= (photos?.length ?? 0) ? photo : getFileURL(parent, photo, "600x0")}
                    ondelete={() => handlePhotoDelete(i)}
                    isThumbnail={thumbnail === i}
                    onthumbnail={() => makePhotoThumbnail(i)}
                    onexif={(src) => handlePhotoExif(i, src)}
                    showThumbnailControls={showThumbnailControls && !(hasRealPhoto && routePreviews.has(photo))}
                    {showExifControls}
                ></PhotoCard>
            </div>
        {/each}
    </div>
</div>
