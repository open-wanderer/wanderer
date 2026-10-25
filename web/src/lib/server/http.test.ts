import { createServer, type Server } from "node:http";
import type { AddressInfo } from "node:net";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { forwardableHeaders } from "./http";

// Headers of an inbox POST behind nginx with `proxy_set_header Connection "upgrade"`.
function inboxHeaders() {
    return new Headers({
        host: "wanderer.example",
        date: "Wed, 30 Sep 2026 05:00:00 GMT",
        digest: "SHA-256=abc",
        "content-type": "application/activity+json",
        signature: 'keyId="https://remote.example/users/a#main-key",headers="(request-target) host date digest"',
        connection: "upgrade",
        upgrade: "websocket",
        "keep-alive": "timeout=5",
    });
}

describe("forwardableHeaders", () => {
    it("keeps end-to-end headers and drops hop-by-hop ones", () => {
        const forwarded = forwardableHeaders(inboxHeaders());

        expect(Object.keys(forwarded).sort()).toEqual(["content-type", "date", "digest", "host", "signature"]);
        expect(forwarded.signature).toContain("keyId=");
    });

    describe("with fetch", () => {
        let server: Server;
        let url: string;

        beforeAll(async () => {
            server = createServer((_, res) => res.end("ok"));
            await new Promise<void>(resolve => server.listen(0, "127.0.0.1", resolve));
            url = `http://127.0.0.1:${(server.address() as AddressInfo).port}/`;
        });
        afterAll(() => server.close());

        it("fails with the original headers", async () => {
            const original: Record<string, string> = {};
            inboxHeaders().forEach((value, key) => (original[key] = value));
            await expect(fetch(url, { method: "POST", headers: original, body: "{}" })).rejects.toThrow();
        });

        it("succeeds with the forwarded headers", async () => {
            const res = await fetch(url, { method: "POST", headers: forwardableHeaders(inboxHeaders()), body: "{}" });
            expect(await res.text()).toBe("ok");
        });
    });
});
