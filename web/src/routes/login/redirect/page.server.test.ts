import { describe, expect, it, vi } from "vitest";

vi.mock("$app/env/private", () => ({ POCKETBASE_PROXY_SECRET: "test-secret" }));

import { issueAppState } from "#lib/server/oauth_app_state.ts";
import { load } from "./+page.server";

const run = (state: string) => load({ url: new URL(`http://localhost/login/redirect?code=c&state=${encodeURIComponent(state)}`) } as never);

describe("/login/redirect load", () => {
    it("recognises a state this server issued", () => {
        expect(run(issueAppState("Google"))).toEqual({ appLogin: { provider: "Google" }, appLoginRefused: false });
    });

    it("refuses a forged state", () => {
        expect(run("wanderer-app.e30.AAAA")).toEqual({ appLogin: null, appLoginRefused: true });
    });

    it("refuses an expired state", () => {
        expect(run(issueAppState("Google", Date.now() - 601_000))).toEqual({ appLogin: null, appLoginRefused: true });
    });

    it("gives the old marker no special treatment", () => {
        expect(run("abc.wanderer-app")).toEqual({ appLogin: null, appLoginRefused: false });
    });

    it("leaves a web login alone", () => {
        expect(run("abc")).toEqual({ appLogin: null, appLoginRefused: false });
    });
});
