import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";
import { DELETE } from "./+server";

describe("DELETE /api/v1/routing/profiles/[id]", () => {
    it("encodes the profile id before forwarding it", async () => {
        const send = vi.fn().mockResolvedValue({});
        const event = {
            request: new Request("http://localhost/api/v1/routing/profiles/profile", {
                method: "DELETE",
            }),
            params: { id: "profile id/ä?" },
            locals: { pb: { send } },
        } as unknown as RequestEvent;

        const response = await DELETE(event);

        expect(response.status).toBe(200);
        expect(send).toHaveBeenCalledWith(
            "/plugins/routing/profiles/profile%20id%2F%C3%A4%3F",
            { method: "DELETE" },
        );
    });
});
