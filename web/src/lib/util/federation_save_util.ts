import { APIError } from "#lib/util/api_util.ts";

type FederationSaveKind = "trail" | "summit-log";
type SaveFetch = (url: RequestInfo | URL, config?: RequestInit) => Promise<Response>;

export const federationSaveRenewalInterval = 15_000;

export function federationSaveBatchID(): string {
    return Array.from(crypto.getRandomValues(new Uint8Array(16)), (byte) => byte.toString(16).padStart(2, "0")).join("");
}

// A save spans metadata, uploads, links and removals. Keep its announcement
// leased while those requests run; complete it even after a partial failure.
export async function withFederationSaveBatch<T>(
    kind: FederationSaveKind,
    id: string | undefined,
    save: () => Promise<T>,
    f: SaveFetch = fetch,
    batchId = federationSaveBatchID(),
    alreadyBegun = false,
    deferComplete?: (complete: () => Promise<void>) => void,
): Promise<T> {
    if (!id) throw new Error("Missing federation save target");
    const request = async (action: "begin" | "complete") => {
        const response = await f(`/api/v1/federation/${kind}/${id}/save/${action}`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ batchId }),
            signal: AbortSignal.timeout(10_000),
        });
        if (!response.ok) {
            const error = await response.json();
            throw new APIError(response.status, error.message, error.detail);
        }
        return (await response.json().catch(() => ({}))).status as string | undefined;
    };
    if (!alreadyBegun) {
        let status: string | undefined;
        try {
            status = await request("begin");
        } catch (error) {
            // An HTTP rejection did not acquire a lease. With a transport
            // failure, begin may have reached the server: still complete that
            // batch after saving, rather than waiting for its one-minute TTL.
            console.warn("Federation save lease unavailable", error instanceof APIError ? error.status : "transport error");
            if (error instanceof APIError) return await save();
        }
        if (status === "not_federated") return await save();
    }
    let renewal: Promise<unknown> | undefined;
    const timer = setInterval(() => {
        if (!renewal) {
            renewal = request("begin").catch(() => {}).finally(() => { renewal = undefined; });
        }
    }, federationSaveRenewalInterval);
    try {
        return await save();
    } finally {
        const complete = async () => {
            clearInterval(timer);
            await renewal;
            // The server's deadline remains the fallback if completion cannot
            // reach it. A saved record does not become an unsaved editor draft.
            await request("complete").catch(() => {});
        };
        if (deferComplete) deferComplete(complete);
        else await complete();
    }
}
