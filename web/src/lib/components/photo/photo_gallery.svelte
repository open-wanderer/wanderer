<script lang="ts">
    import { isVideoURL } from "$lib/util/file_util";
    import PhotoSwipeVideoPlugin from "$lib/vendor/photo-swipe-video-plugin";
    import type { DataSource } from "photoswipe";
    import PhotoSwipeLightbox from "photoswipe/lightbox";
    import { onMount } from "svelte";

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

        lightbox.init();

        lightbox.on("beforeOpen", () => {
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
    });
</script>
