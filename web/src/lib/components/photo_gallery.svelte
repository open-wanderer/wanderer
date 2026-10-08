<script lang="ts">
    import { isVideoURL } from "$lib/util/file_util";
    import PhotoSwipeVideoPlugin from "$lib/vendor/photo-swipe-video-plugin";
    import type { DataSource } from "photoswipe";
    import PhotoSwipeLightbox from "photoswipe/lightbox";
    import { pushState } from "$app/navigation";
    import { page } from "$app/state";
    import { onMount, onDestroy } from "svelte";

    interface Props {
        photos: string[];
        open?: (idx: number) => void;
    }

    let { photos }: Props = $props();

    export function openGallery(idx: number = 0) {
        lightbox.loadAndOpen(idx, lightboxDataSource);
    }
    let lightbox: PhotoSwipeLightbox;
    let lightboxDataSource: DataSource;

    // An open lightbox is a shallow history entry, so Back closes it and
    // Forward reopens it. The id keeps several galleries on a page apart.
    const galleryId = $props.id();
    let ownsHistoryEntry = false;

    $effect(() => {
        const entry = page.state.lightbox;
        if (!lightbox) {
            return;
        }
        if (entry?.id === galleryId) {
            if (!lightbox.pswp) {
                openGallery(entry.index);
            }
        } else if (lightbox.pswp) {
            ownsHistoryEntry = false;
            lightbox.pswp.close();
        }
    });

    onMount(() => {
        lightboxDataSource = photos.map((p) => {
            if (isVideoURL(p)) {
                return {
                    type: "video",
                    videoSrc: p,
                };
            }
            return {
                src: p,
            };
        });
        lightbox = new PhotoSwipeLightbox({
            dataSource: lightboxDataSource,
            pswpModule: async () => await import("photoswipe"),
        });
        const videoPlugin = new PhotoSwipeVideoPlugin(lightbox);

        lightbox.on("beforeOpen", () => {
            const pswp = lightbox.pswp;

            if (page.state.lightbox?.id !== galleryId) {
                pushState("", {
                    lightbox: { id: galleryId, index: pswp?.options.index ?? 0 },
                });
            }
            ownsHistoryEntry = true;

            const ds = pswp?.options?.dataSource;

            if (Array.isArray(ds)) {
                for (let idx = 0, len = ds.length; idx < len; idx++) {
                    const item = ds[idx];                    
                    if (item.type === "video") {
                        const v = document.createElement("video");
                        v.addEventListener(
                            "loadedmetadata",
                            function () {
                                item.width = this.videoWidth;
                                item.height = this.videoHeight;
                                pswp?.refreshSlideContent(idx);                                
                            },
                            false,
                        );
                        v.src = item.videoSrc as string
                    } else {
                        const img = new Image();
                        img.onload = () => {
                            item.width = img.naturalWidth;
                            item.height = img.naturalHeight;
                            pswp?.refreshSlideContent(idx);
                        };
                        img.src = item.src as string;
                    }
                }
            }
        });

        // Closed from the UI: drop the entry it added. Closing via Back has
        // already removed it.
        lightbox.on("close", () => {
            if (ownsHistoryEntry && page.state.lightbox?.id === galleryId) {
                history.back();
            }
            ownsHistoryEntry = false;
        });

        lightbox.init();
    });

    onDestroy(() => {
        // Unmounting must not navigate, so give up the entry before closing.
        ownsHistoryEntry = false;
        lightbox?.destroy();
    });
</script>
