import { describe, expect, it, vi } from "vitest";
import { copyTimeoutMs, sharedAssetCopy } from "./asset_copy_singleflight.ts";
import type { Asset } from "#lib/models/asset.ts";

describe("shared asset copying", () => {
    it("shares a copy while canceling only the first caller's wait", async () => {
        let finish!: (asset: Asset) => void;
        let sharedSignal!: AbortSignal;
        const create = vi.fn((signal: AbortSignal) => {
            sharedSignal = signal;
            return new Promise<Asset>((resolve) => { finish = resolve; });
        });
        const first = new AbortController();
        const firstCopy = sharedAssetCopy("same-user:source", first.signal, create).catch((error) => error);
        const secondCopy = sharedAssetCopy("same-user:source", new AbortController().signal, create);
        first.abort();
        expect(await firstCopy).toMatchObject({ status: 499 });
        expect(sharedSignal.aborted).toBe(false);
        finish({ id: "copied" } as Asset);
        expect(await secondCopy).toEqual({ asset: { id: "copied" }, created: false });
        expect(create).toHaveBeenCalledOnce();
    });

    it("reclaims an endlessly pending operation at its deadline", async () => {
        vi.useFakeTimers();
        try {
            const pending = sharedAssetCopy("timeout-user:source", new AbortController().signal,
                () => new Promise<Asset>(() => {})).catch((error) => error);
            await vi.advanceTimersByTimeAsync(copyTimeoutMs);
            const error = await pending;
            expect(error).toMatchObject({ status: 504, response: { code: "asset_copy_timeout" } });
            await expect(sharedAssetCopy("timeout-user:source", new AbortController().signal,
                async () => ({ id: "retry" }) as Asset)).resolves.toMatchObject({ asset: { id: "retry" }, created: true });
        } finally {
            vi.useRealTimers();
        }
    });
});
