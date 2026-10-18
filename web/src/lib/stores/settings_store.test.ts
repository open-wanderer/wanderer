import { afterEach, describe, expect, it, vi } from "vitest";
import { settings_for_user } from "./settings_store";

function respondWith(expand: unknown) {
    vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({ id: "user1", expand }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
    })));
}

describe("settings_for_user", () => {
    afterEach(() => vi.unstubAllGlobals());

    it("reads the single record the unique settings.user index expands to", async () => {
        respondWith({ settings_via_user: { id: "settings1", user: "user1", language: "de" } });
        await expect(settings_for_user("user1")).resolves.toMatchObject({ id: "settings1", language: "de" });
    });

    it("reads the one-element list a backend without the index returns", async () => {
        respondWith({ settings_via_user: [{ id: "settings1", user: "user1", language: "de" }] });
        await expect(settings_for_user("user1")).resolves.toMatchObject({ id: "settings1", language: "de" });
    });

    it("rejects missing, ambiguous or foreign settings", async () => {
        for (const expand of [
            undefined,
            { settings_via_user: [] },
            { settings_via_user: [{ id: "a", user: "user1" }, { id: "b", user: "user1" }] },
            { settings_via_user: { id: "settings2", user: "someone-else" } },
        ]) {
            respondWith(expand);
            await expect(settings_for_user("user1")).rejects.toMatchObject({ status: 404 });
        }
    });
});
