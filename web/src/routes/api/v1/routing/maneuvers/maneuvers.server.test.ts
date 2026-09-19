import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";
import { POST } from "./+server";

describe("POST /api/v1/routing/maneuvers", () => {
    it("forwards the normalized request and strips provider-only fields", async () => {
        const send = vi.fn().mockResolvedValue({
            geometry: {
                format: "encoded_polyline",
                precision: 6,
                coordinates: "_c`|@_oyo@_pR_pR",
            },
            maneuvers: [
                {
                    type: "start",
                    providerInstruction: "Provider start text",
                    distanceMeters: 10,
                    beginShapeIndex: 0,
                    endShapeIndex: 1,
                    warnings: [],
                },
                {
                    type: "destination",
                    providerInstruction: "Provider destination text",
                    distanceMeters: 0,
                    beginShapeIndex: 1,
                    endShapeIndex: 1,
                    warnings: [],
                },
            ],
        });
        const request = new Request("http://localhost/api/v1/routing/maneuvers", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "Accept-Language": "de",
            },
            body: JSON.stringify({ trailId: "trail1234567890", share: "token" }),
        });
        const event = {
            request,
            locals: {
                pb: { send },
                settings: { language: "de" },
            },
        } as unknown as RequestEvent;

        const response = await POST(event);
        expect(response.status).toBe(200);
        expect(send).toHaveBeenCalledOnce();
        const [path, options] = send.mock.calls[0];
        expect(path).toBe("/plugins/routing/maneuvers");
        expect(JSON.parse(options.body)).toEqual({
            trailId: "trail1234567890",
            language: "de",
            share: "token",
        });

        const raw = await response.text();
        expect(raw).not.toContain("providerInstruction");
        const body = JSON.parse(raw);
        expect(body.language).toBe("de");
        expect(body.maneuvers[0].instruction).toBe("Auf der Route starten.");
        expect(body.maneuvers[0]).not.toHaveProperty("warnings");
    });
});
