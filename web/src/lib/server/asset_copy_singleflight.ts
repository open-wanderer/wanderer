import type { Asset } from "#lib/models/asset.ts";
import { ClientResponseError } from "pocketbase";

// Bound the shared download and local persistence operation.
export const copyTimeoutMs = 180_000;
const copies = new Map<string, { promise: Promise<Asset>; joined: boolean }>();

export class AssetCopyTimeoutError extends ClientResponseError {
    constructor() {
        super({ status: 504, response: { code: "asset_copy_timeout", message: "Photo copy timed out" } });
    }
}

// A caller leaving the page stops only its wait. The shared copy has its own
// deadline and is reclaimed even if a provider ignores cancellation.
export async function sharedAssetCopy(
    key: string,
    signal: AbortSignal,
    create: (signal: AbortSignal) => Promise<Asset>,
): Promise<{ asset: Asset; created: boolean }> {
    let entry = copies.get(key);
    const owner = !entry;
    if (!entry) {
        const controller = new AbortController();
        let timeout!: ReturnType<typeof setTimeout>;
        const deadline = new Promise<never>((_, reject) => {
            timeout = setTimeout(() => {
                controller.abort();
                reject(new AssetCopyTimeoutError());
            }, copyTimeoutMs);
        });
        entry = { joined: false, promise: Promise.race([Promise.resolve().then(() => create(controller.signal)), deadline]).finally(() => {
            clearTimeout(timeout);
            copies.delete(key);
        }) };
        copies.set(key, entry);
    } else {
        entry.joined = true;
    }
    const asset = await waitForCopy(entry.promise, signal);
    // Several targets may already be attaching this copy. A failing target
    // must not remove a copy another request is using.
    return { asset, created: owner && !entry.joined };
}

function waitForCopy(promise: Promise<Asset>, signal: AbortSignal): Promise<Asset> {
    return new Promise((resolve, reject) => {
        const canceled = () => reject(new ClientResponseError({ status: 499, response: { message: "Request canceled" } }));
        if (signal.aborted) {
            // Keep the detached operation's rejection observed.
            void promise.catch(() => {});
            canceled();
            return;
        }
        signal.addEventListener("abort", canceled, { once: true });
        promise.then(resolve, reject).finally(() => signal.removeEventListener("abort", canceled));
    });
}
