import { describe, expect, it, vi } from "vitest";
import { APIError } from "#lib/util/api_util.ts";
import { plugin_sync } from "./plugin_instance_store";

describe("plugin_sync", () => {
    it("posts the instance id", async () => {
        const fetch = vi.fn(async () => Response.json({ started: true }, { status: 202 }));

        await expect(plugin_sync("instance0000001", fetch)).resolves.toEqual({ started: true });
        expect(fetch).toHaveBeenCalledWith("/api/v1/plugin-system/sync", {
            method: "POST",
            body: JSON.stringify({ instanceId: "instance0000001" }),
        });
    });

    it("throws the status and the wait of a refused sync", async () => {
        const fetch = vi.fn(async () =>
            Response.json(
                { status: 429, message: "Plugin sync was started too recently.", data: { retryAfter: 112 } },
                { status: 429 },
            ),
        );

        const error = await plugin_sync("instance0000001", fetch).catch((e) => e);

        expect(error).toBeInstanceOf(APIError);
        expect(error.status).toBe(429);
        expect(error.detail).toEqual({ retryAfter: 112 });
    });
});
