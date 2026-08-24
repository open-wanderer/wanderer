import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";
import { PATCH } from "./+server";

describe("PATCH /api/v1/routing/mappings/[id]", () => {
    it("encodes the mapping id before forwarding it", async () => {
        const send = vi.fn().mockResolvedValue({});
        const event = {
            request: new Request("http://localhost/api/v1/routing/mappings/mapping", {
                method: "PATCH",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ enabled: true }),
            }),
            params: { id: "mapping id/ä?" },
            locals: { pb: { send } },
        } as unknown as RequestEvent;

        const response = await PATCH(event);

        expect(response.status).toBe(200);
        expect(send).toHaveBeenCalledWith(
            "/plugins/routing/mappings/mapping%20id%2F%C3%A4%3F",
            {
                method: "PATCH",
                body: JSON.stringify({ enabled: true }),
                headers: { "Content-Type": "application/json" },
            },
        );
    });
});
