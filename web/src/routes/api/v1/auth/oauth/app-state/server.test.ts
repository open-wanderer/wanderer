import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";

vi.mock("$app/env/private", () => ({ POCKETBASE_PROXY_SECRET: "test-secret" }));

import { verifyAppState } from "#lib/server/oauth_app_state.ts";
import { POST } from "./+server";

function event(body: string) {
    const listAuthMethods = vi.fn().mockResolvedValue({ oauth2: { providers: [{ name: "google", displayName: "Google" }] } });
    return {
        request: new Request("http://localhost/api/v1/auth/oauth/app-state", { method: "POST", body }),
        locals: { pb: { collection: () => ({ listAuthMethods }) } },
    } as unknown as RequestEvent;
}

describe("POST /api/v1/auth/oauth/app-state", () => {
    it("issues a state naming the provider", async () => {
        const response = await POST(event(`{"provider":"google"}`));

        expect(response.status).toBe(200);
        const { state, expiresIn } = await response.json();
        expect(verifyAppState(state)).toEqual({ provider: "Google" });
        expect(expiresIn).toBe(600);
    });

    it("refuses a provider this instance does not offer", async () => {
        const response = await POST(event(`{"provider":"Your Bank"}`));

        expect(response.status).toBe(400);
        expect(await response.json()).toEqual({ message: "invalid_provider" });
    });

    it.each([["a missing provider", "{}"], ["invalid JSON", "nope"]])("answers %s with 400", async (_name, body) => {
        expect((await POST(event(body))).status).toBe(400);
    });
});
