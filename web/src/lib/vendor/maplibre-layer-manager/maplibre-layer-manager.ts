import * as M from "maplibre-gl";
import { DebugLayer } from "./debug-layer";
import { baseMapStyles, defaultMapState, type BaseLayer, type MapState } from "./layers";
import { OverlayLayer } from "./overlay-layer";
import { OverpassLayer, type OverpassPopupActionFactory, } from "./overpass-layer";



const cloneState = (state: MapState): MapState =>
    JSON.parse(JSON.stringify(state)) as MapState;

export class LayerManager {
    private map: M.Map;
    state!: MapState;
    layers: Record<string, BaseLayer> = {};
    private addedListeners: Set<string> = new Set();
    private options?: LayerManagerOptions;
    private waitingForBaseStyle = false;
    private pendingLayerOperations: Array<() => void> = [];
    private isFlushScheduled = false;

  constructor(map: M.Map, options?: LayerManagerOptions) {
        this.map = map;
        this.options = options;

        const storedMapState = localStorage.getItem("map-state")
        if (storedMapState) {
            this.state = JSON.parse(storedMapState)
        } else {
            this.state = cloneState(defaultMapState)
        }
    }

    init() {
        this.map.on('moveend', () => { });

        this.map.on('styledata', () => {
            this.restoreLayers();
        })
        try {
            this.update(this.state, true);

      const overpassLayer = new OverpassLayer(
        this.map,
        this.options?.overpassActionFactory,
      );
            const debugLayer = new DebugLayer()

            this.addLayer("overpass", overpassLayer)
            this.addLayer("debug", debugLayer)

            this.map.on('moveend', this.updateOverpassLayerAfterMapMoveBinded);
        } catch (e) {
            console.error(e)
            // map is probably not initialized yet
        }
    }

    update(newState: MapState, initialize: boolean = false) {
        const oldState = cloneState(this.state);
        this.state = cloneState(newState);

        if (oldState.base != this.state.base || initialize) {
            this.updateBaseLayer(baseMapStyles[this.state.base])
        }

        for (const [name, active] of Object.entries(this.state.overlays)) {
            const oldOverlayActive = oldState.overlays[name]

            if (active && (!oldOverlayActive || initialize)) {
                this.addLayer(name, new OverlayLayer(name))
            } else if (oldOverlayActive && !active) {
                this.removeLayer(name)
            }
        }

        this.updateOverpassLayer(this.state);

        localStorage.setItem("map-state", JSON.stringify(this.state));
    }

    updateOverpassLayerAfterMapMoveBinded = this.updateOverpassLayerAfterMapMove.bind(this);
    updateOverpassLayerAfterMapMove() {
        this.updateOverpassLayer(this.state)
    }

    private async updateOverpassLayer(newState: MapState) {
        const overpassLayer = this.layers.overpass;
        const initialSource = this.map.getSource('overpass') as M.GeoJSONSource | undefined;

        if (!overpassLayer || !initialSource) {
            return;
        }

        const castedOverpassLayer = overpassLayer as OverpassLayer;
        overpassLayer.filter = await castedOverpassLayer.updateLayerIfNeeded(newState, this.map.getBounds());

        // The style can reload while awaiting, so re-grab the source before calling setData.
        const overpassSource = this.map.getSource('overpass') as M.GeoJSONSource | undefined;
        overpassSource?.setData(castedOverpassLayer.data);
    }

    private updateBaseLayer(layer: string | M.StyleSpecification) {
        this.waitingForBaseStyle = true;
        this.map.once("idle", () => {
            this.waitingForBaseStyle = false;
            this.flushPendingLayerOperations();
        });
        this.map.setStyle(layer);
    }


    addLayer(id: string, layer: BaseLayer) {
        const existingLayer = this.layers[id];
        this.layers[id] = layer;
        this.enqueueLayerOperation(() => this.performAddLayer(id, layer, existingLayer));
    }

    private performAddLayer(id: string, layer: BaseLayer, existingLayer?: BaseLayer) {
        if (this.isOverlayId(id) && !this.state.overlays[id]) {
            return;
        }

        if (!layer.spec) {
            return;
        }

        if (existingLayer && this.map.getLayer(id)) {
            for (const [sourceId, s] of Object.entries(layer.spec.sources)) {
                if (s.type !== "geojson" || !this.map.getSource(sourceId)) {
                    continue;
                }

                const source = this.map.getSource(sourceId) as M.GeoJSONSource;
                source.setData(s.data);
            }
            return;
        }

        for (const [sourceId, s] of Object.entries(layer.spec.sources)) {
            if (!this.map.getSource(sourceId)) {
                this.map.addSource(sourceId, s);
            }
        }

        for (const l of layer.spec.layers) {
            if (!this.map.getLayer(l.id)) {
                this.map.addLayer(l);
            }
        }

        if (layer.listeners) {
            for (const [layerId, listener] of Object.entries(layer.listeners)) {
                if (listener.onEnter && !this.addedListeners.has("mouseenter-" + layerId)) {
                    this.addedListeners.add("mouseenter-" + layerId);
                    this.map.on("mouseenter", layerId, listener.onEnter);
                }

                if (listener.onLeave && !this.addedListeners.has("onleave-" + layerId)) {
                    this.addedListeners.add("mouseleave-" + layerId);
                    this.map.on("mouseleave", layerId, listener.onLeave);
                }

                if (listener.onMouseDown && !this.addedListeners.has("mousedown-" + layerId)) {
                    this.addedListeners.add("mousedown-" + layerId);
                    this.map.on("mousedown", layerId, listener.onMouseDown);
                }

                if (listener.onMouseUp && !this.addedListeners.has("mouseup-" + layerId)) {
                    this.addedListeners.add("mouseup-" + layerId);
                    this.map.on("mouseup", layerId, listener.onMouseUp);
                }

                if (listener.onMouseMove && !this.addedListeners.has("mousemove-" + layerId)) {
                    this.addedListeners.add("mousemove-" + layerId);
                    this.map.on("mousemove", layerId, listener.onMouseMove);
                }
            }
        }

        if (layer.filter && !this.map.getFilter(id)) {
            this.map.setFilter(id, layer.filter);
        }
    }

    removeLayer(id: string) {
        const layer = this.layers[id];
        if (!layer) {
            return;
        }
        delete this.layers[id];
        this.enqueueLayerOperation(() => this.performRemoveLayer(id, layer));
    }

    private performRemoveLayer(id: string, layer: BaseLayer) {
        if (this.isOverlayId(id) && this.state.overlays[id]) {
            return;
        }

        if (!layer?.spec) {
            return;
        }

        for (const mapLayer of layer.spec.layers) {
            if (this.map.getLayer(mapLayer.id)) {
                this.map.removeLayer(mapLayer.id);
            }
        }

        for (const sourceId of Object.keys(layer.spec.sources)) {
            if (this.map.getSource(sourceId)) {
                this.map.removeSource(sourceId);
            }
        }

        if (layer.listeners) {
            for (const [layerId, listener] of Object.entries(layer.listeners)) {
                if (listener.onEnter) {
                    this.addedListeners.delete("mouseenter-" + layerId);
                    this.map.off("mouseenter", layerId, listener.onEnter);
                }

                if (listener.onLeave) {
                    this.addedListeners.delete("mouseleave-" + layerId);
                    this.map.off("mouseleave", layerId, listener.onLeave);
                }
                if (listener.onMouseDown) {
                    this.addedListeners.delete("click-" + layerId);
                    this.map.off("click", layerId, listener.onMouseDown);
                }
                if (listener.onMouseMove) {
                    this.addedListeners.delete("mousemove-" + layerId);
                    this.map.off("mousemove", layerId, listener.onMouseMove);
                }
            }
        }
    }

    private restoreLayers() {
        if (!this.map.isStyleLoaded()) {
            return;
        }
        for (const [id, layer] of Object.entries(this.layers)) {
            this.addLayer(id, layer)
        }
    }

    private enqueueLayerOperation(callback: () => void) {
        if (!this.waitingForBaseStyle && this.map.isStyleLoaded()) {
            callback();
            return;
        }

        this.pendingLayerOperations.push(callback);
        this.scheduleLayerOperationFlush();
    }

    private flushPendingLayerOperations() {
        if (!this.map.isStyleLoaded() || this.waitingForBaseStyle) {
            this.isFlushScheduled = false;
            return;
        }

        this.isFlushScheduled = false;

        while (this.pendingLayerOperations.length) {
            const operation = this.pendingLayerOperations.shift();
            if (!operation) {
                continue;
            }
            operation();

            if (this.waitingForBaseStyle) {
                break;
            }
        }

        if (this.pendingLayerOperations.length) {
            this.scheduleLayerOperationFlush();
        }
    }

    private scheduleLayerOperationFlush() {
        if (this.waitingForBaseStyle) {
            return;
        }

        if (!this.pendingLayerOperations.length || this.isFlushScheduled) {
            return;
        }

        if (!this.map.isStyleLoaded()) {
            this.isFlushScheduled = true;
            this.map.once("styledata", () => {
                this.isFlushScheduled = false;
                this.scheduleLayerOperationFlush();
            });
            return;
        }

        const run = () => {
            this.isFlushScheduled = false;
            this.flushPendingLayerOperations();
        };

        if (this.map.areTilesLoaded() && !this.map.isMoving()) {
            run();
            return;
        }

        this.isFlushScheduled = true;
        this.map.once("idle", run);
        this.map.triggerRepaint();
    }

    private isOverlayId(id: string): id is keyof MapState["overlays"] {
        return Object.prototype.hasOwnProperty.call(defaultMapState.overlays, id);
    }
}

type LayerManagerOptions = {
  overpassActionFactory?: OverpassPopupActionFactory;
};
