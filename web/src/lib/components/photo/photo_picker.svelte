<script lang="ts">
    import { assetFileCoordinatesFor, markAssetFileMetadata } from "$lib/stores/asset_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { extractPhotoExifMetadata, type GPSCoordinates, type PhotoExifMetadata } from "$lib/util/exif_util";
    import { getFileURL, readAsDataURLAsync } from "$lib/util/file_util";
    import { _ } from "svelte-i18n";
    import PhotoCard from "./photo_card.svelte";

    interface AssetPluginPreview {
        pluginId?: string;
        assetId: string;
        filename: string;
        takenAt?: string;
        thumbnailUrl?: string;
    }

    interface Props {
        id: string;
        photos: string[];
        photoFiles: File[] | undefined;
        parent: { [key: string]: any };
        thumbnail?: number;
        showThumbnailControls?: boolean;
        showExifControls?: boolean;
        maxSizeBytes?: number;
        onexif?: (src: string, coordinates?: GPSCoordinates) => void | Promise<void>;
        onassetplugin?: () => void;
        assetPluginPreviews?: AssetPluginPreview[];
        onassetplugindelete?: (assetId: string) => void;
    }

    let {
        id,
        photos = $bindable(),
        photoFiles = $bindable(),
        parent,
        thumbnail = $bindable(0),
        showThumbnailControls = true,
        showExifControls = false,
        maxSizeBytes = 20971520,
        onexif,
        onassetplugin,
        assetPluginPreviews = [],
        onassetplugindelete,
    }: Props = $props();

    let photoPreviews: string[] = $state([]);
    let showSourceMenu = $state(false);

    $effect(() => fetchPhotos(photoFiles ?? []));

    function fetchPhotos(photos: File[]) {
        Promise.all(
            photos.map(async (f) => {
                return await readAsDataURLAsync(f);
            }),
        ).then((v) => {
            photoPreviews = v;
        });
    }

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

    function handlePlusClick() {
        if (onassetplugin) {
            showSourceMenu = !showSourceMenu;
        } else {
            openPhotoBrowser();
        }
    }

    function selectLocal() {
        showSourceMenu = false;
        openPhotoBrowser();
    }

    function selectAssetPlugin() {
        showSourceMenu = false;
        onassetplugin?.();
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
            let photoFile = file;
            if (!isSupportedUpload(file)) {
                continue;
            }

            let metadata: PhotoExifMetadata | undefined;
            if (isImageFile(file)) {
                metadata = await extractPhotoExifMetadata(file);
            }

            if (isHEICFile(file)) {
                const heic2any = (await import("heic2any")).default;
                const converted = (await heic2any({
                    blob: file,
                    toType: "image/jpeg",
                })) as Blob | Blob[];
                photoFile = new File(
                    [Array.isArray(converted) ? converted[0] : converted],
                    jpegFileName(file.name),
                    { type: "image/jpeg" },
                );
            }
            markAssetFileMetadata(photoFile, metadata);
            if (!photoFiles) {
                photoFiles = [];
            }
            photoFiles = [...photoFiles, photoFile];
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

        if (index >= photos.length) {
            if (!photoFiles) {
                photoFiles = [];
            }
            const adjustedIndex = index - photos.length;
            photoFiles.splice(adjustedIndex, 1);
            photoPreviews.splice(adjustedIndex, 1);

            photoPreviews = [...photoPreviews];
        } else {
            photos.splice(index, 1);
            photos = [...photos];
        }
    }

    function handlePhotoExif(index: number, src: string) {
        const fileIndex = index - photos.length;
        const coordinates = fileIndex >= 0 ? assetFileCoordinatesFor(photoFiles?.[fileIndex]) : undefined;
        onexif?.(src, coordinates);
    }

    function assetThumbnailURL(preview: AssetPluginPreview): string {
        if (preview.thumbnailUrl) {
            return preview.thumbnailUrl;
        }
        if (!preview.pluginId) {
            return "";
        }
        const plugin = encodeURIComponent(preview.pluginId);
        return `/api/v1/plugins/assets/${plugin}/thumbnail/${encodeURIComponent(preview.assetId)}`;
    }

    function formatTakenAt(value?: string): string {
        if (!value) return "";
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) return "";
        return date.toLocaleString(undefined, {
            dateStyle: "medium",
            timeStyle: "short",
        });
    }

    function isSupportedUpload(file: File): boolean {
        return isImageFile(file) ||
            isHEICFile(file) ||
            ["video/mp4", "video/ogg", "video/webm"].includes(file.type);
    }

    function isImageFile(file: File): boolean {
        return file.type.startsWith("image") || /\.(jpe?g|png|webp|gif|avif|heic|heif)$/i.test(file.name);
    }

    function isHEICFile(file: File): boolean {
        return ["image/heic", "image/heif", "image/heic-sequence", "image/heif-sequence"].includes(file.type) ||
            /\.(heic|heif)$/i.test(file.name);
    }

    function jpegFileName(fileName: string): string {
        const jpegName = fileName.replace(/\.(heic|heif)$/i, ".jpg");
        return jpegName === fileName ? `${fileName}.jpg` : jpegName;
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
    <div class="relative shrink-0 grow-0 basis-auto">
        <button
            aria-label="Open photo browser"
            class="btn-secondary relative h-32 w-32"
            type="button"
            onclick={handlePlusClick}
        >
            <i class="fa fa-plus"></i>
            {#if onassetplugin}
                <span class="absolute bottom-2 right-2 flex h-5 w-5 items-center justify-center rounded-full bg-background/90 text-[10px] text-content/70 shadow-sm">
                    <i class="fa fa-chevron-down"></i>
                </span>
            {/if}
        </button>
        {#if showSourceMenu}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div
                class="fixed inset-0 z-[9]"
                onclick={() => (showSourceMenu = false)}
            ></div>
            <div
                class="absolute top-full left-0 mt-1 bg-menu-background border border-input-border rounded-lg shadow-lg z-[10] min-w-[130px] overflow-hidden"
            >
                <button
                    type="button"
                    class="w-full text-left px-3 py-2 text-sm hover:bg-menu-item-background-hover flex items-center gap-2"
                    onclick={selectLocal}
                >
                    <i class="fa fa-image w-4 text-center"></i>
                    {$_("local")}
                </button>
                <button
                    type="button"
                    class="w-full text-left px-3 py-2 text-sm hover:bg-menu-item-background-hover flex items-center gap-2 border-t border-input-border"
                    onclick={selectAssetPlugin}
                >
                    <i class="fa fa-images w-4 text-center"></i>
                    {$_("photo-library")}
                </button>
            </div>
        {/if}
    </div>
    <input
        type="file"
        id="{id}-photo-input"
        accept="image/jpeg,image/png,image/heic,image/heif,image/heic-sequence,image/heif-sequence,image/webp,image/gif,image/avif,.jpg,.jpeg,.png,.heic,.heif,.webp,.gif,.avif,video/mp4"
        multiple={true}
        style="display: none;"
        onchange={() => handlePhotoSelection()}
    />
    <div class="flex overflow-x-auto gap-x-3 w-full">
        {#each (photos ?? []).concat(photoPreviews) as photo, i}
            <div class="shrink-0 grow-0 basis-auto">
                <PhotoCard
                    src={i >= photos.length ? photo : getFileURL(parent, photo)}
                    ondelete={() => handlePhotoDelete(i)}
                    isThumbnail={thumbnail === i}
                    onthumbnail={() => makePhotoThumbnail(i)}
                    onexif={(src) => handlePhotoExif(i, src)}
                    {showThumbnailControls}
                    {showExifControls}
                ></PhotoCard>
            </div>
        {/each}
        {#each assetPluginPreviews as preview (preview.assetId)}
            <div class="relative shrink-0 grow-0 basis-auto">
                <PhotoCard
                    src={assetThumbnailURL(preview)}
                    ondelete={() => onassetplugindelete?.(preview.assetId)}
                    showThumbnailControls={false}
                    showExifControls={false}
                ></PhotoCard>
                <div
                    class="absolute top-1 left-1 bg-black/60 rounded px-1 py-0.5 flex items-center pointer-events-none"
                >
                    <i class="fa fa-images text-white text-xs"></i>
                </div>
                <div
                    class="absolute bottom-0 left-0 right-0 rounded-b-xl bg-black/65 px-2 py-1 text-white pointer-events-none"
                    title={[preview.filename, formatTakenAt(preview.takenAt)].filter(Boolean).join(" - ")}
                >
                    <div class="truncate text-[11px] font-medium leading-tight">{preview.filename}</div>
                    {#if formatTakenAt(preview.takenAt)}
                        <div class="truncate text-[10px] leading-tight opacity-85">
                            {formatTakenAt(preview.takenAt)}
                        </div>
                    {/if}
                </div>
            </div>
        {/each}
    </div>
</div>
