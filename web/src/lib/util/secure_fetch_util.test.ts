import { createServer, type Server } from "node:http";
import type { AddressInfo } from "node:net";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

const env = vi.hoisted(() => ({ dev: true }));
vi.mock("$app/env", () => ({ get dev() { return env.dev; } }));

import { DownloadError, fetchExternalFile, fetchExternalJson } from "./secure_fetch_util";

describe("fetchExternalJson", () => {
    let server: Server;
    let base: string;
    let received: { method?: string; body: string; type?: string } | undefined;

    beforeAll(async () => {
        server = createServer(async (req, res) => {
            let body = "";
            for await (const chunk of req) body += chunk;
            received = { method: req.method, body, type: req.headers["content-type"] };
            if (req.url === "/json") return res.end(`{"hits":[1]}`);
            if (req.url === "/error") { res.writeHead(404); return res.end(`{"message":"gone"}`); }
            if (req.url === "/text") return res.end("not json");
            if (req.url === "/big") return res.end(`"${"x".repeat(100_000)}"`);
            if (req.url === "/redirect-private") { res.writeHead(302, { location: "http://169.254.169.254/latest" }); return res.end(); }
            if (req.url === "/redirect") { res.writeHead(302, { location: "/json" }); return res.end(); }
            if (req.url === "/trickle") {
                const timer = setInterval(() => res.write(" "), 10);
                res.on("close", () => clearInterval(timer));
                return;
            }
            res.writeHead(500);
            res.end();
        });
        await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
        base = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
    });

    afterAll(() => new Promise((resolve) => { server.closeAllConnections(); server.close(resolve); }));

    beforeEach(() => {
        env.dev = true;
        received = undefined;
    });

    const options = { maxBytes: 1024 };

    it("posts JSON and reads JSON", async () => {
        const response = await fetchExternalJson(`${base}/json`, { ...options, method: "POST", body: { q: "a" } });

        expect(response).toEqual({ status: 200, ok: true, json: { hits: [1] } });
        expect(received).toEqual({ method: "POST", body: `{"q":"a"}`, type: "application/json" });
    });

    it("hands back an error response", async () => {
        expect(await fetchExternalJson(`${base}/error`, options)).toEqual({ status: 404, ok: false, json: { message: "gone" } });
    });

    it("refuses a body that is no JSON", async () => {
        await expect(fetchExternalJson(`${base}/text`, options)).rejects.toMatchObject({ message: "invalid_response", status: 502 });
    });

    it("stops at the size limit", async () => {
        await expect(fetchExternalJson(`${base}/big`, options)).rejects.toMatchObject({ message: "file_too_large" });
    });

    it("gives up on a response that keeps trickling in", async () => {
        await expect(fetchExternalJson(`${base}/trickle`, { ...options, deadlineMs: 100 })).rejects.toMatchObject({ message: "request_timeout" });
    });

    it("follows a redirect to an allowed address", async () => {
        expect((await fetchExternalJson(`${base}/redirect`, options)).json).toEqual({ hits: [1] });
    });

    it("refuses a redirect to a private address", async () => {
        await expect(fetchExternalJson(`${base}/redirect-private`, options)).rejects.toBeInstanceOf(DownloadError);
    });

    it("reaches loopback only in a dev build", async () => {
        env.dev = false;

        await expect(fetchExternalJson(`${base}/json`, options)).rejects.toMatchObject({ message: "invalid_url" });
        expect(received).toBeUndefined();
    });

    it.each([
        "http://10.0.0.1/",
        "http://192.168.1.1/",
        "http://169.254.169.254/latest/meta-data",
        "http://[fd00::1]/",
        "http://localhost/",
        "http://db:8090/",
        "file:///etc/passwd",
    ])("refuses %s even in a dev build", async (url) => {
        await expect(fetchExternalJson(url, options)).rejects.toMatchObject({ message: "invalid_url" });
    });

    it("leaves fetchExternalFile without the dev allowance", async () => {
        await expect(fetchExternalFile(`${base}/json`)).rejects.toMatchObject({ message: "invalid_url" });
    });
});
