import { describe, expect, it, vi } from "vitest";

vi.mock("$app/env/private", () => ({ POCKETBASE_PROXY_SECRET: "test-secret" }));

import { APP_STATE_TTL_SECONDS, isAppState, issueAppState, verifyAppState } from "./oauth_app_state";

describe("OAuth app state", () => {
    it("verifies a state it issued", () => {
        expect(verifyAppState(issueAppState("Google"))).toEqual({ provider: "Google" });
    });

    it("issues a different state every time", () => {
        expect(issueAppState("Google")).not.toBe(issueAppState("Google"));
    });

    it("rejects an expired state", () => {
        const issued = Date.now();
        const state = issueAppState("Google", issued);

        expect(verifyAppState(state, issued + (APP_STATE_TTL_SECONDS - 1) * 1000)).not.toBeNull();
        expect(verifyAppState(state, issued + APP_STATE_TTL_SECONDS * 1000)).toBeNull();
    });

    it("rejects a changed payload", () => {
        const [prefix, payload, signature] = issueAppState("Google").split(".");
        const forged = Buffer.from(JSON.stringify({ ...JSON.parse(Buffer.from(payload, "base64url").toString()), p: "Bank" })).toString("base64url");

        expect(verifyAppState(`${prefix}.${forged}.${signature}`)).toBeNull();
    });

    it("rejects a state signed with another key", async () => {
        vi.resetModules();
        vi.doMock("$app/env/private", () => ({ POCKETBASE_PROXY_SECRET: "another-secret" }));
        const other = await import("./oauth_app_state");

        expect(verifyAppState(other.issueAppState("Google"))).toBeNull();
    });

    it.each([
        ["the old marker", "abc123.wanderer-app"],
        ["only the prefix", "wanderer-app."],
        ["no signature", "wanderer-app.e30"],
        ["an empty signature", "wanderer-app.e30."],
        ["an extra part", `${issueAppState("Google")}.x`],
    ])("rejects %s", (_name, state) => {
        expect(verifyAppState(state)).toBeNull();
    });

    it("does not take the old marker or a web state for an app state", () => {
        expect(isAppState("abc123.wanderer-app")).toBe(false);
        expect(isAppState("abc123")).toBe(false);
        expect(isAppState(null)).toBe(false);
        expect(isAppState(issueAppState("Google"))).toBe(true);
    });
});
