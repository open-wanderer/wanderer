<script lang="ts">
    import { show_toast } from "#lib/stores/toast_store.svelte.ts";
    import { getFileURL } from "#lib/util/file_util.ts";
    import { photoFilePreview } from "#lib/util/photo_preview_util.ts";
    import { _ } from "svelte-i18n";
    import PhotoCard from "./photo_card.svelte";

    interface Props {
        id: string;
        photos: string[];
        photoFiles: File[] | undefined;
        parent: { [key: string]: any };
        thumbnail?: number;
        showThumbnailControls?: boolean;
        showExifControls?: boolean;
        maxSizeBytes?: number;
        onexif?: (src: string) => void;
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
    }: Props = $props();

    let photoPreviews: string[] = $state([]);

    $effect(() => {
        let cancelled = false;
        Promise.all((photoFiles ?? []).map(photoFilePreview)).then((v) => {
            if (!cancelled) photoPreviews = v;
        }).catch(() => {});
        return () => { cancelled = true; };
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
            let photoFile = file;
            if (
                !file.type.startsWith("image") &&
                !["video/mp4", "video/ogg", "video/webm"].includes(file.type)
            ) {
                continue;
            } else if (file.type === "image/heic") {
                const heic2any = (await import("heic2any")).default;
                photoFile = new File(
                    [
                        (await heic2any({
                            blob: file,
                            toType: "image/jpeg",
                        })) as Blob,
                    ],
                    file.name,
                );
            }
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
        }

        if (index >= photos.length) {
            if (!photoFiles) {
                photoFiles = [];
            }
            const adjustedIndex = index - photos.length;
            photoFiles = photoFiles.filter((_, i) => i !== adjustedIndex);
            photoPreviews = photoPreviews.filter((_, i) => i !== adjustedIndex);
        } else {
            photos = photos.filter((_, i) => i !== index);
        }
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
    <button
        aria-label="Open photo browser"
        class="btn-secondary h-32 w-32 shrink-0 grow-0 basis-auto"
        type="button"
        onclick={openPhotoBrowser}><i class="fa fa-plus"></i></button
    >
    <input
        type="file"
        id="{id}-photo-input"
        accept="image/*,video/mp4"
        multiple={true}
        style="display: none;"
        onchange={() => handlePhotoSelection()}
    />
    <div class="flex overflow-x-auto gap-x-3 w-full">
        {#each (photos ?? []).concat(photoPreviews) as photo, i}
            <div class="shrink-0 grow-0 basis-auto">
                <PhotoCard
                    src={i >= photos.length ? photo : getFileURL(parent, photo, "600x0")}
                    ondelete={() => handlePhotoDelete(i)}
                    isThumbnail={thumbnail === i}
                    onthumbnail={() => makePhotoThumbnail(i)}
                    {onexif}
                    {showThumbnailControls}
                    {showExifControls}
                ></PhotoCard>
            </div>
        {/each}
    </div>
</div>
