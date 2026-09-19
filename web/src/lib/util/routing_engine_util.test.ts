import type { RoutingEngine, RoutingOptions } from "$lib/models/routing";
import { describe, expect, it } from "vitest";
import {
    applyRouteEditorRoutingEngineSelection,
    applyRoutingEffectiveControlDefaults,
    selectEnabledRoutingPlugin,
} from "./routing_engine_util";

function engine(
    pluginId: string,
    { enabled = true, route = true, via = false } = {},
): RoutingEngine {
    return {
        pluginId,
        instanceId: `${pluginId}-instance`,
        name: pluginId,
        enabled,
        roles: route ? ["route"] : ["elevation"],
        metadata: { routing: { supportsViaRouting: via } },
    };
}

function options(): RoutingOptions {
    return {
        autoRouting: true,
        modeOfTransport: "pedestrian",
        routingPluginId: "primary",
        routingInstanceId: "primary-instance",
        routingMode: "via",
        nativeProfileKey: "custom-profile",
        profileRevisions: { primary: "revision" },
        preferences: { speedPreference: 5 },
        nativeConfig: { pedestrian: { shortest: true } },
    };
}

describe("route editor engine selection", () => {
    it("switches only the editor context and clears engine-specific profile state", () => {
        const current = options();

        const selected = applyRouteEditorRoutingEngineSelection(
            current,
            [engine("primary", { via: true }), engine("alternative", { via: true })],
            "alternative",
        );

        expect(selected?.pluginId).toBe("alternative");
        expect(current.routingPluginId).toBe("alternative");
        expect(current.routingInstanceId).toBe("alternative-instance");
        expect(current.nativeProfileKey).toBeUndefined();
        expect(current.profileRevisions).toEqual({});
        expect(current.preferences).toEqual({});
        expect(current.nativeConfig).toEqual({});
        expect(current.routingMode).toBe("via");
    });

    it("falls back to segment mode when the selected engine cannot route via anchors", () => {
        const current = options();

        applyRouteEditorRoutingEngineSelection(current, [engine("alternative")], "alternative");

        expect(current.routingMode).toBe("segment");
        expect(current.routingModeExplicit).toBe(true);
    });

    it("ignores disabled and non-route engines", () => {
        for (const candidate of [
            engine("alternative", { enabled: false }),
            engine("alternative", { route: false }),
        ]) {
            const current = options();
            expect(
                applyRouteEditorRoutingEngineSelection(current, [candidate], "alternative"),
            ).toBeUndefined();
            expect(current.routingPluginId).toBe("primary");
            expect(current.nativeProfileKey).toBe("custom-profile");
        }
    });

    it("selects only an enabled engine with the requested executable role", () => {
        expect(
            selectEnabledRoutingPlugin(
                [engine("disabled", { enabled: false }), engine("elevation", { route: false })],
                "disabled",
                "elevation",
            ),
        ).toBe("elevation");
    });

    it("keeps effective control defaults as display-only fallbacks", () => {
        const current = options();
        current.preferences = {};
        applyRoutingEffectiveControlDefaults(current, {
            category: "Hiking",
            controls: [
                {
                    key: "futurePreference",
                    type: "number",
                    target: "preference",
                    default: 0.75,
                },
            ],
        });
        expect(current.preferences).toEqual({});
    });
});
