import type { Map } from "maplibre-gl";

// MapLibre 6.11.1: public resize() stops drag/wheel handlers and emits a
// movestart that unlocks GeolocateControl. Its redraw() renders at time 0,
// advancing ease callbacks with the wrong clock. Keep these private calls in
// one place; map_resize.test.ts guards the installed engine contract.
type ResizeEngine = Map & {
    _resizeInternal: () => void;
    _render: (time: number) => void;
    _lostContextStyle?: { style: unknown };
    _update: () => void;
    style?: unknown;
    _camera?: {
        _moving?: boolean;
        _onEaseFrame?: (progress: number) => void;
        transform: { width: number; height: number; resize: (width: number, height: number, constrain: boolean) => void };
        _requestedCameraState?: { resize: (width: number, height: number, constrain: boolean) => void };
    };
};

export function paintMapCanvas(map: Map) {
    if ((map as ResizeEngine).style && !(map as ResizeEngine)._lostContextStyle?.style) (map as ResizeEngine)._render(performance.now());
}

const resizingEngines = new WeakSet<Map>();
export function isMapCanvasResize(map: Map) { return resizingEngines.has(map); }

const adaptedFrames = new WeakSet<Function>();

export function resizeMapCanvas(map: Map) {
    const engine = map as ResizeEngine;
    if (engine._lostContextStyle?.style != null) return;
    const camera = engine._camera;
    const frame = camera?._onEaseFrame;
    if (camera && frame && !adaptedFrames.has(frame)) {
        // Mercator ease closures retain a pixel anchor from their starting
        // dimensions. Evaluate them in that frame, then restore the live canvas
        // dimensions. Otherwise a resize changes the animation's destination.
        const { width, height } = camera.transform;
        const adapted = (progress: number) => {
            const live = camera.transform;
            const nextWidth = live.width, nextHeight = live.height;
            live.resize(width, height, false);
            camera._requestedCameraState?.resize(width, height, false);
            frame(progress);
            live.resize(nextWidth, nextHeight, false);
            camera._requestedCameraState?.resize(nextWidth, nextHeight, false);
            // The easing callback emitted move in its original pixel frame.
            // Reproject DOM overlays in the frame that is about to be painted.
            engine.fire("move", { canvasResize: true });
        };
        adaptedFrames.add(adapted);
        camera._onEaseFrame = adapted;
    }
    const canvas = map.getCanvas();
    const width = camera?.transform.width, height = camera?.transform.height;
    const bufferWidth = canvas.width, bufferHeight = canvas.height;
    engine._resizeInternal();
    if (camera?.transform.width === width && camera?.transform.height === height
        && canvas.width === bufferWidth && canvas.height === bufferHeight) {
        paintMapCanvas(map);
        return;
    }
    engine._update();
    const moving = camera?._moving;
    if (!moving) engine.fire("move", { canvasResize: true });
    engine.fire("resize", { canvasResize: true });
    if (!moving) {
        resizingEngines.add(map);
        try { engine.fire("moveend", { canvasResize: true }); }
        finally { resizingEngines.delete(map); }
    }
    // ResizeObserver runs after rAF. Paint now so resizing the drawing buffer
    // cannot expose one blank frame; use the animation clock, never redraw(0).
    paintMapCanvas(map);
}
