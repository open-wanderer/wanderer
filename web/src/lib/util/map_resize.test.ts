import { readFileSync } from "node:fs";
import { describe, expect, it, vi } from "vitest";
import type { Map } from "maplibre-gl";
import { resizeMapCanvas } from "./map_resize";

describe("MapLibre canvas resize contract", () => {
    it("uses the installed 6.11.1 resize and animation-clock rendering internals", () => {
        const pkg = JSON.parse(readFileSync(new URL("../../../node_modules/maplibre-gl/package.json", import.meta.url), "utf8"));
        expect(pkg.version).toBe("6.11.1");
        const source = readFileSync(new URL("../../../node_modules/maplibre-gl/dist/maplibre-gl-dev.mjs", import.meta.url), "utf8");
        expect(source).toContain("_resizeInternal(constrainTransform = true)");
        expect(source).toContain("_render(paintStartTimeStamp)");
        expect(source).toContain("if (this._lostContextStyle.style !== null) return this;");
        expect(source).toContain("_update(updateStyle)");
        expect(source).toContain("this._sourcesDirty = true;");
        expect(source).toContain("this._camera._moving");
        expect(source).toContain("this._onEaseFrame");
        expect(source).toContain("this._camera._requestedCameraState");
        const canvas = { width: 800, height: 600 };
        const engine = { style: {}, getCanvas: () => canvas, _resizeInternal: vi.fn(() => { canvas.width = 1000; }), _render: vi.fn(), _update: vi.fn(), fire: vi.fn(), resize: vi.fn(), redraw: vi.fn() };
        const before = performance.now();
        resizeMapCanvas(engine as unknown as Map);
        expect(engine._resizeInternal).toHaveBeenCalledOnce();
        expect(engine._render).toHaveBeenCalledOnce();
        expect(engine._render.mock.calls[0][0]).toBeGreaterThanOrEqual(before);
        expect(engine.resize).not.toHaveBeenCalled();
        expect(engine.redraw).not.toHaveBeenCalled();
    });
    it("only paints unchanged dimensions without scheduling sources or emitting movement", () => {
        const canvas = { width: 800, height: 600 };
        const engine = { style: {}, getCanvas: () => canvas, _resizeInternal: vi.fn(), _render: vi.fn(), _update: vi.fn(), fire: vi.fn() };
        resizeMapCanvas(engine as unknown as Map);
        expect(engine._render).toHaveBeenCalledOnce();
        expect(engine._update).not.toHaveBeenCalled();
        expect(engine.fire).not.toHaveBeenCalled();
    });
    it("invalidates sources and emits resize when only the DPR buffer changes", () => {
        const canvas = { width: 800, height: 600 };
        const engine = { style: {}, getCanvas: () => canvas, _resizeInternal: vi.fn(() => { canvas.width = 1600; canvas.height = 1200; }), _render: vi.fn(), _update: vi.fn(), fire: vi.fn() };
        resizeMapCanvas(engine as unknown as Map);
        expect(engine._update).toHaveBeenCalledOnce();
        expect(engine.fire.mock.calls.map(call => call[0])).toEqual(["move", "resize", "moveend"]);
    });
    it("leaves a lost WebGL context untouched", () => {
        const engine = { _lostContextStyle: { style: {} }, _resizeInternal: vi.fn(), _update: vi.fn(), _render: vi.fn() };
        resizeMapCanvas(engine as unknown as Map);
        expect(engine._resizeInternal).not.toHaveBeenCalled();
        expect(engine._update).not.toHaveBeenCalled();
        expect(engine._render).not.toHaveBeenCalled();
    });
});
