import { BODY_SIZE_LIMIT } from "$app/env/private";
import { error, isHttpError } from "@sveltejs/kit";
import type { Handle } from "@sveltejs/kit/hooks";
import type { IncomingMessage } from "node:http";
import { UPLOAD_BODY_LIMIT, uploadBodyLimit } from "./upload_body_limit.js";

export { UPLOAD_BODY_LIMIT };

export const REQUEST_BODY_LIMIT = 1 << 20;
export const INBOX_BODY_LIMIT = 2 << 20;
export const LOGIN_BODY_LIMIT = 64 << 10;

function checkDeclaredLength(request: Request, limit: number): void {
    const value = request.headers.get("content-length");
    if (value === null) return;
    if (!/^\d+$/.test(value)) throw error(400, "invalid_content_length");
    if (Number(value) > limit) throw error(413, "request_body_too_large");
}

function cancelReader(reader: ReadableStreamDefaultReader<Uint8Array>): void {
    void reader.cancel().catch(() => {}).finally(() => reader.releaseLock());
}

type CancelBodyReader = typeof cancelReader;

export async function readBoundedRequestBody(request: Request, limit: number, cancelBody: CancelBodyReader = cancelReader): Promise<Uint8Array<ArrayBuffer>> {
    checkDeclaredLength(request, limit);
    if (!request.body) return new Uint8Array();
    const reader = request.body.getReader();
    const chunks: Uint8Array[] = [];
    let total = 0;
    let failed = false;
    try {
        while (true) {
            const { value, done } = await reader.read();
            if (done) break;
            total += value.byteLength;
            if (total > limit) throw error(413, "request_body_too_large");
            chunks.push(value);
        }
        const bytes = new Uint8Array(total);
        let offset = 0;
        for (const chunk of chunks) {
            bytes.set(chunk, offset);
            offset += chunk.byteLength;
        }
        return bytes;
    } catch (cause) {
        failed = true;
        cancelBody(reader);
        throw cause;
    } finally {
        if (!failed) reader.releaseLock();
    }
}

export async function readBoundedRequestText(request: Request, limit: number): Promise<string> {
    return new TextDecoder().decode(await readBoundedRequestBody(request, limit));
}

function limitedUpload(request: Request, limit: number, cancelBody: CancelBodyReader): Request {
    const reader = request.body!.getReader();
    let total = 0;
    let stopped = false;
    const stopReading = () => {
        if (stopped) return;
        stopped = true;
        cancelBody(reader);
    };
    const body = new ReadableStream<Uint8Array>({
        async pull(controller) {
            try {
                const { value, done } = await reader.read();
                if (done) {
                    stopped = true;
                    controller.close();
                    reader.releaseLock();
                    return;
                }
                total += value.byteLength;
                if (total > limit) throw error(413, "request_body_too_large");
                controller.enqueue(value);
            } catch (cause) {
                stopReading();
                controller.error(cause);
            }
        },
        cancel: stopReading,
    }, { highWaterMark: 0 });
    const init: RequestInit & { duplex: "half" } = { body, duplex: "half" };
    return new Request(request, init);
}

function requestLimit(url: URL, request: Request): { limit: number; upload: boolean } {
    const path = url.pathname;
    if (/^\/api\/v1\/activitypub\/user\/[^/]+\/inbox\/?$/.test(path)) return { limit: INBOX_BODY_LIMIT, upload: false };
    if (path === "/api/v1/auth/login") return { limit: LOGIN_BODY_LIMIT, upload: false };
    const multipart = request.headers.get("content-type")?.split(";", 1)[0].trim().toLowerCase() === "multipart/form-data";
    if (multipart && (request.method === "POST" || request.method === "PUT") && (
        path === "/api/v1/trail/upload" ||
        /^\/api\/v1\/(trail|list|summit-log)\/form(?:\/[^/]+)?\/?$/.test(path) ||
        /^\/api\/v1\/(trail|list|summit-log|waypoint|user)\/[^/]+\/file\/?$/.test(path)
    )) return { limit: uploadBodyLimit(BODY_SIZE_LIMIT), upload: true };
    return { limit: REQUEST_BODY_LIMIT, upload: false };
}

// Runs before authentication or route parsing. Ordinary bodies are buffered
// only up to their small route limit. Upload bodies stay streamed until a
// route checks authentication and consumes them.
export const limitRequestBodies: Handle = async ({ event, resolve }) => {
    // Internal event.fetch requests carry server-generated bodies.
    if (!event.request.body || event.isSubRequest) return resolve(event);
    const { limit, upload } = requestLimit(event.url, event.request);
    const nodeRequest = (event.platform as { req?: IncomingMessage } | undefined)?.req;
    let cancellationDeferred = false;
    const cancelBody: CancelBodyReader = (reader) => {
        if (!nodeRequest || nodeRequest.socket.destroyed) return cancelReader(reader);
        // Kit cancels a Node request by destroying its socket. Stop reading
        // now, but let the rejection finish writing before cancelling it.
        cancellationDeferred = true;
        nodeRequest.pause();
        nodeRequest.socket.once("close", () => cancelReader(reader));
    };
    try {
        checkDeclaredLength(event.request, limit);
        const request = upload ? limitedUpload(event.request, limit, cancelBody) : new Request(event.request, {
            body: (await readBoundedRequestBody(event.request, limit, cancelBody)).buffer,
        });
        const response = await resolve({ ...event, request });
        if (cancellationDeferred) response.headers.set("Connection", "close");
        return response;
    } catch (cause) {
        if (isHttpError(cause, 413) || isHttpError(cause, 400)) return Response.json({ message: cause.body.message }, {
            status: cause.status,
            headers: nodeRequest ? { Connection: "close" } : undefined,
        });
        throw cause;
    }
};
