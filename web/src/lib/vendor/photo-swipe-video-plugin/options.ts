export type PhotoSwipeVideoPluginOptions = {
    videoAttributes: {
        controls: string,
        playsinline: string,
        preload: string,
    },
    autoplay: boolean,
    preventDragOffset: number
}

export const defaultOptions: PhotoSwipeVideoPluginOptions = {
    videoAttributes: { controls: '', playsinline: '', preload: 'auto' },
    autoplay: false,
  
    // prevent drag/swipe gesture over the bottom part of video
    // set to 0 to disable
    preventDragOffset: 40
  };
