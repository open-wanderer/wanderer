import * as M from "maplibre-gl";
import { DebugLayer } from "./debug-layer";
import { baseMapStyles, defaultMapState, type BaseLayer, type MapState } from "./layers";
import { OverlayLayer } from "./overlay-layer";
import { OverpassLayer } from "./overpass-layer";



export class LayerManager {
    private map: M.Map;
    state!: MapState;
    layers: Record<string, BaseLayer> = {};
    private addedListeners: Set<string> = new Set();

    constructor(map: M.Map) {
        this.map = map;

        const storedMapState = localStorage.getItem("map-state")
        if (storedMapState) {
            this.state = JSON.parse(storedMapState)

            // Fix for spelling correction introduced in v0.18.4
            if (this.state && 'ammenity' in this.state.pois) {
                this.state.pois.amenity = this.state.pois.ammenity;
                delete this.state.pois.ammenity;

                localStorage.setItem('map-state', JSON.stringify(this.state));
            }
        } else {
            this.state = defaultMapState
        }
    }

    init() {
        this.map.on('moveend', () => { });

        this.map.on('styledata', () => {
            this.restoreLayers();
        })
        try {
            this.update(this.state, true);

            const overpassLayer = new OverpassLayer(this.map)
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
        const oldState = this.state;

        if (oldState.base != newState.base || initialize) {
            this.updateBaseLayer(baseMapStyles[newState.base])
        }

        for (const [name, active] of Object.entries(newState.overlays)) {
            const oldOverlayActive = oldState.overlays[name]

            if (active && (!oldOverlayActive || initialize)) {
                this.addLayer(name, new OverlayLayer(name))
            } else if (oldOverlayActive && !active) {
                this.removeLayer(name)
            }
        }

        this.updateOverpassLayer(newState);

        this.state = newState
        localStorage.setItem("map-state", JSON.stringify(this.state));
    }

    updateOverpassLayerAfterMapMoveBinded = this.updateOverpassLayerAfterMapMove.bind(this);
    updateOverpassLayerAfterMapMove() {
        this.updateOverpassLayer(this.state)
    }

    private async updateOverpassLayer(newState: MapState) {
        const overpassLayer = this.layers.overpass;
        if (overpassLayer && this.map.getSource('overpass')) {
            const castedOverpassLayer = overpassLayer as OverpassLayer;
            overpassLayer.filter = await castedOverpassLayer.updateLayerIfNeeded(newState, this.map.getBounds());
            (this.map.getSource('overpass') as M.GeoJSONSource).setData(castedOverpassLayer.data);
        }
    }

    private updateBaseLayer(layer: string | M.StyleSpecification) {
        this.map.setStyle(layer);
    }


    addLayer(id: string, layer: BaseLayer) {
        if (!layer.spec) {
            return;
        }
        if (this.layers[id] && this.map.getLayer(id)) {
            // update sources and return
            for (const [sourceId, s] of Object.entries(layer.spec.sources)) {
                if (s.type != "geojson" || !this.map.getSource(sourceId)) {
                    continue;
                }

                const source = this.map.getSource(sourceId) as M.GeoJSONSource
                source.setData(s.data)
                this.layers[id] = layer
            }
            return;
        }
        for (const [id, s] of Object.entries(layer.spec.sources)) {
            if (!this.map.getSource(id)) {
                this.map.addSource(id, s)
            }
        }

        for (const l of layer.spec.layers) {
            if (!this.map.getLayer(l.id)) {
                this.map.addLayer(l)
            }
        }

        if (layer.listeners) {
            for (const [id, listener] of Object.entries(layer.listeners)) {
                if (listener.onEnter && !this.addedListeners.has("mouseenter-" + id)) {
                    this.addedListeners.add("mouseenter-" + id)
                    this.map.on('mouseenter', id, listener.onEnter);
                }

                if (listener.onLeave && !this.addedListeners.has("onleave-" + id)) {
                    this.addedListeners.add("mouseleave-" + id)
                    this.map.on('mouseleave', id, listener.onLeave);
                }

                if (listener.onMouseDown && !this.addedListeners.has("mousedown-" + id)) {
                    this.addedListeners.add("mousedown-" + id)
                    this.map.on('mousedown', id, listener.onMouseDown);
                }

                if (listener.onMouseUp && !this.addedListeners.has("mouseup-" + id)) {
                    this.addedListeners.add("mouseup-" + id)
                    this.map.on('mouseup', id, listener.onMouseUp);
                }

                if (listener.onMouseMove && !this.addedListeners.has("mousemove-" + id)) {
                    this.addedListeners.add("mousemove-" + id)
                    this.map.on('mousemove', id, listener.onMouseMove);
                }
            }
        }

        if (layer.filter && !this.map.getFilter(id)) {
            this.map.setFilter(id, layer.filter)
        }

        this.layers[id] = layer
    }

    removeLayer(id: string) {
        const layer = this.layers[id];
        if (!layer) {
            return;
        }
        for (const l of layer.spec.layers) {
            if (this.map.getLayer(l.id)) {
                this.map.removeLayer(l.id)
            }
        }

        for (const [id, _] of Object.entries(layer.spec.sources)) {
            if (this.map.getSource(id)) {
                this.map.removeSource(id)
            }
        }

        if (layer.listeners) {
            for (const [id, listener] of Object.entries(layer.listeners)) {
                if (listener.onEnter) {
                    this.addedListeners.delete("mouseenter-" + id)
                    this.map.off('mouseenter', id, listener.onEnter);
                }

                if (listener.onLeave) {
                    this.addedListeners.delete("mouseleave-" + id)
                    this.map.off('mouseleave', id, listener.onLeave);
                }
                if (listener.onMouseDown) {
                    this.addedListeners.delete("click-" + id)
                    this.map.off('click', id, listener.onMouseDown);
                }
                if (listener.onMouseMove) {
                    this.addedListeners.delete("mousemove-" + id)
                    this.map.off('mousemove', id, listener.onMouseMove);
                }
            }
        }

        delete this.layers[id]

    }

    private restoreLayers() {
        for (const [id, layer] of Object.entries(this.layers)) {
            this.addLayer(id, layer)
        }
    }
}