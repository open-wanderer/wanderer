import { getRequest, setResponse } from "@sveltejs/kit/node";
import { isHttpError, type RequestEvent } from "@sveltejs/kit";
import { createServer, request as httpRequest } from "node:http";
import { once } from "node:events";
import type { AddressInfo } from "node:net";
import { expect, it, vi } from "vitest";
import { INBOX_BODY_LIMIT, LOGIN_BODY_LIMIT, REQUEST_BODY_LIMIT, limitRequestBodies } from "./request_body";
import { POST as uploadTrailFile } from "../../routes/api/v1/trail/[id]/file/+server";
import { POST as uploadListFile } from "../../routes/api/v1/list/[id]/file/+server";
import { POST as uploadLogFile } from "../../routes/api/v1/summit-log/[id]/file/+server";
import { POST as updateTrail } from "../../routes/api/v1/trail/form/[id]/+server";
import { POST as updateList } from "../../routes/api/v1/list/form/[id]/+server";
import { POST as updateLog } from "../../routes/api/v1/summit-log/form/[id]/+server";
import { PUT as uploadTrail } from "../../routes/api/v1/trail/upload/+server";

it.each([
    ["/api/v1/auth/login", 70 * 1024, LOGIN_BODY_LIMIT],
    ["/api/v1/activitypub/user/alice/inbox", 3 * 1024 * 1024, INBOX_BODY_LIMIT],
    ["/api/v1/search/trails", 2 * 1024 * 1024, REQUEST_BODY_LIMIT],
])("writes JSON 413 before closing an oversized native chunked request at %s", async (path, size, limit) => {
    const resolve = vi.fn(async () => new Response("resolved"));
    let consumedBeforeResponse = 0;
    let socketClosed: Promise<unknown> | undefined;
    const server = createServer(async (req, res) => {
        socketClosed = once(req.socket, "close");
        req.on("data", (chunk: Buffer) => { if (!res.writableFinished) consumedBeforeResponse += chunk.length; });
        try {
            const request = await getRequest({ request: req, base: "http://127.0.0.1", bodySizeLimit: 64 * 1024 * 1024 });
            const event = { request, url: new URL(request.url), platform: { req } } as unknown as RequestEvent;
            await setResponse(res, await limitRequestBodies({ event, resolve }));
        } catch (cause) {
            res.writeHead(500).end(String(cause));
        }
    });
    server.listen(0, "127.0.0.1");
    await once(server, "listening");
    try {
        const result = await new Promise<{ status: number | undefined; text: string; connection: string | undefined }>((accept, reject) => {
            const req = httpRequest({ hostname: "127.0.0.1", port: (server.address() as AddressInfo).port, method: "POST", path,
                headers: { "Content-Type": "application/json" } }, res => {
                const chunks: Buffer[] = [];
                res.on("data", chunk => chunks.push(chunk));
                res.on("end", () => accept({ status: res.statusCode, text: Buffer.concat(chunks).toString(), connection: res.headers.connection }));
                res.on("error", reject);
            });
            req.on("error", reject);
            req.setTimeout(5000, () => req.destroy(new Error("response timed out")));
            for (let offset = 0; offset < size; offset += 16 * 1024) req.write(Buffer.alloc(Math.min(16 * 1024, size - offset), 120));
            req.end();
        });
        expect(result.status).toBe(413);
        expect(JSON.parse(result.text)).toEqual({ message: "request_body_too_large" });
        expect(result.connection).toBe("close");
        expect(resolve).not.toHaveBeenCalled();
        // The native stream pauses when its bounded reader stops consuming.
        expect(consumedBeforeResponse).toBeGreaterThan(limit);
        expect(consumedBeforeResponse).toBeLessThanOrEqual(limit + 128 * 1024);
        await socketClosed;
    } finally {
        server.closeAllConnections();
        await new Promise<void>(accept => server.close(() => accept()));
    }
});

it.each([
    ["/api/v1/trail/abcdefghijklmn0/file", "POST", uploadTrailFile],
    ["/api/v1/list/abcdefghijklmn0/file", "POST", uploadListFile],
    ["/api/v1/summit-log/abcdefghijklmn0/file", "POST", uploadLogFile],
    ["/api/v1/trail/form/abcdefghijklmn0", "POST", updateTrail],
    ["/api/v1/list/form/abcdefghijklmn0", "POST", updateList],
    ["/api/v1/summit-log/form/abcdefghijklmn0", "POST", updateLog],
    ["/api/v1/trail/upload", "PUT", uploadTrail],
] as const)("rejects a native anonymous large upload without buffering it at %s", async (path, method, route) => {
    const declaredBytes = 1024 ** 3;
    let receivedBeforeResponse = 0;
    const server = createServer(async (req, res) => {
        const receivedAtStart = req.socket.bytesRead;
        try {
            const request = await getRequest({ request: req, base: "http://127.0.0.1", bodySizeLimit: Infinity });
            const event = { request, url: new URL(request.url), platform: { req }, params: { id: "abcdefghijklmn0" }, locals: {} } as unknown as RequestEvent;
            const response = await limitRequestBodies({ event, resolve: (input) => route(input) });
            receivedBeforeResponse = req.socket.bytesRead - receivedAtStart;
            response.headers.set("Connection", "close");
            await setResponse(res, response);
        } catch (cause) {
            if (isHttpError(cause)) res.writeHead(cause.status, { "Content-Type": "application/json" }).end(JSON.stringify(cause.body));
            else res.writeHead(500).end(String(cause));
        }
    });
    server.listen(0, "127.0.0.1");
    await once(server, "listening");
    const before = process.memoryUsage().rss;
    try {
        const response = await new Promise<{ status: number | undefined; text: string }>((accept, reject) => {
            const req = httpRequest({ hostname: "127.0.0.1", port: (server.address() as AddressInfo).port, method,
                path,
                headers: { "Content-Type": "multipart/form-data; boundary=test", "Content-Length": declaredBytes } }, res => {
                const chunks: Buffer[] = [];
                res.on("data", chunk => chunks.push(chunk));
                res.on("end", () => { req.destroy(); accept({ status: res.statusCode, text: Buffer.concat(chunks).toString() }); });
                res.on("error", reject);
            });
            req.on("error", reject);
            req.setTimeout(2000, () => req.destroy(new Error("upload was read before authentication")));
            // Keep a declared 1 GiB request open after its first small chunk.
            // Authentication must respond before the remainder is available.
            req.write(Buffer.alloc(8 * 1024, 120));
        });
        expect(response.status).toBe(401);
        expect(JSON.parse(response.text)).toEqual({ message: "Unauthorized" });
        expect(receivedBeforeResponse).toBeLessThan(64 * 1024);
        expect(process.memoryUsage().rss - before).toBeLessThan(32 * 1024 * 1024);
    } finally {
        server.closeAllConnections();
        await new Promise<void>(accept => server.close(() => accept()));
    }
});
