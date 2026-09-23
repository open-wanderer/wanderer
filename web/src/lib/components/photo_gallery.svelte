<script lang="ts">
    import { isVideoURL } from "$lib/util/file_util";
    import PhotoSwipeVideoPlugin from "$lib/vendor/photo-swipe-video-plugin";
    import type { DataSource } from "photoswipe";
    import PhotoSwipeLightbox from "photoswipe/lightbox";
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
    let isHistoryPushed = false;
    let isClosingFromPopstate = false;

    function handlePopstate() {
        if (lightbox?.pswp?.isOpen) {
            isClosingFromPopstate = true;
            lightbox.pswp.close();
        }
    }

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

        window.addEventListener("popstate", handlePopstate);

        lightbox.on("beforeOpen", () => {
            isHistoryPushed = true;
            isClosingFromPopstate = false;
            history.pushState({ pswp: true }, "");

            const pswp = lightbox.pswp;
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

        lightbox.on("close", () => {
            if (isHistoryPushed && !isClosingFromPopstate) {
                isHistoryPushed = false;
                history.back();
            }
            isClosingFromPopstate = false;
        });

        lightbox.on("destroy", () => {
            isHistoryPushed = false;
            isClosingFromPopstate = false;
        });

        lightbox.init();
    });

    onDestroy(() => {
        if (typeof window !== "undefined") {
            window.removeEventListener("popstate", handlePopstate);
        }
        if (lightbox?.pswp?.isOpen) {
            lightbox.pswp.close();
        }
        lightbox?.destroy();
    });
</script>
