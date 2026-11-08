import type { RequestEvent } from "@sveltejs/kit";
import { expect, it, vi } from "vitest";
import { DELETE } from "./+server.ts";

function deleteEvent() {
    const send = vi.fn().mockResolvedValue({ acknowledged: true, assetDeleted: false });
    const event = {
        params: { id: "photo0000000001" },
        url: new URL("https://a.example/api/v1/assets/photo0000000001?trail=trail0000000001&waypoint=point0000000001"),
        locals: { user: { actor: "actor0000000001" }, pb: { send } },
        fetch: vi.fn(),
    } as unknown as RequestEvent;
    return { event, send };
}

it("sends the concrete child target and preserves the unlink result", async () => {
    const { event, send } = deleteEvent();
    const response = await DELETE(event);
    expect(await response.json()).toEqual({ acknowledged: true, assetDeleted: false });
    expect(send).toHaveBeenCalledWith("/assets/photo0000000001?waypoint=point0000000001", { method: "DELETE", fetch: event.fetch });
});

it("rejects anonymous deletion and invalid target IDs before contacting the backend", async () => {
    const { event, send } = deleteEvent();
    event.locals.user = undefined;
    expect((await DELETE(event)).status).toBe(401);
    expect(send).not.toHaveBeenCalled();
    event.locals.user = { actor: "actor0000000001" } as typeof event.locals.user;
    event.url.searchParams.set("waypoint", "invalid");
    expect((await DELETE(event)).status).toBe(400);
    expect(send).not.toHaveBeenCalled();
});
