import { afterEach, describe, expect, it, vi } from "vitest";
import { federationSaveRenewalInterval, withFederationSaveBatch } from "./federation_save_util";

afterEach(() => vi.useRealTimers());

describe("federation save leases", () => {
    it("renews during a slow upload and completes only after the saved state is final", async () => {
        vi.useFakeTimers();
        const actions: string[] = [];
        const request = vi.fn(async (url: RequestInfo | URL, _options?: RequestInit) => {
            actions.push(String(url).split("/").at(-1)!);
            return Response.json({ status: "ok" });
        });
        let finish!: (result: string) => void;
        const upload = new Promise<string>((resolve) => { finish = resolve; });
        const operation = withFederationSaveBatch("trail", "trail", () => upload, request);
        await vi.advanceTimersByTimeAsync(4 * federationSaveRenewalInterval);
        expect(actions).toEqual(["begin", "begin", "begin", "begin", "begin"]);
        const tokens = request.mock.calls.map(([, options]) => JSON.parse(String(options?.body)).batchId);
        expect(new Set(tokens).size).toBe(1);
        finish("saved with photos");
        await expect(operation).resolves.toBe("saved with photos");
        expect(actions.at(-1)).toBe("complete");
        expect(vi.getTimerCount()).toBe(0);
    });

    it("completes a partial save and keeps its original failure", async () => {
        const failure = new Error("second photo upload failed");
        const request = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) => Response.json({ status: "ok" }));
        await expect(withFederationSaveBatch("summit-log", "log", async () => { throw failure; }, request)).rejects.toBe(failure);
        expect(request.mock.calls.map(([url]) => String(url).split("/").at(-1))).toEqual(["begin", "complete"]);
    });

    it("retains the successful save when completion must fall back to its server deadline", async () => {
        const request = vi.fn()
            .mockResolvedValueOnce(Response.json({ status: "ok" }))
            .mockRejectedValueOnce(new Error("connection lost"));
        await expect(withFederationSaveBatch("trail", "trail", async () => "saved", request)).resolves.toBe("saved");
    });

    it.each([403, 429])("saves content when begin returns %s", async (status) => {
        const warning = vi.spyOn(console, "warn").mockImplementation(() => {});
        const save = vi.fn(async () => "saved");
        const request = vi.fn(async () => Response.json({ message: "Lease unavailable" }, { status }));
        await expect(withFederationSaveBatch("trail", "trail", save, request)).resolves.toBe("saved");
        expect(save).toHaveBeenCalledOnce();
        expect(request).toHaveBeenCalledOnce();
        expect(warning).toHaveBeenCalledWith("Federation save lease unavailable", status);
        warning.mockRestore();
    });

    it("keeps save failures when begin cannot reach the server", async () => {
        const warning = vi.spyOn(console, "warn").mockImplementation(() => {});
        const failure = new Error("photo upload failed");
        const request = vi.fn().mockRejectedValue(new Error("connection lost"));
        await expect(withFederationSaveBatch("summit-log", "log", async () => { throw failure; }, request)).rejects.toBe(failure);
        expect(request).toHaveBeenCalledTimes(2);
        expect(request.mock.calls[1][0]).toMatch(/\/complete$/);
        warning.mockRestore();
    });

    it("does not renew or complete a not_federated no-op", async () => {
        vi.useFakeTimers();
        const request = vi.fn(async () => Response.json({ status: "not_federated" }));
        const operation = withFederationSaveBatch("trail", "historical", async () => {
            await new Promise((resolve) => setTimeout(resolve, 60_000));
            return "saved";
        }, request);
        await vi.runAllTimersAsync();
        await expect(operation).resolves.toBe("saved");
        expect(request).toHaveBeenCalledOnce();
    });

    it("retains a not_federated save failure without retrying the mutation", async () => {
        const failure = new Error("save failed");
        const save = vi.fn().mockRejectedValue(failure);
        const request = vi.fn(async () => Response.json({ status: "not_federated" }));
        await expect(withFederationSaveBatch("trail", "historical", save, request)).rejects.toBe(failure);
        expect(save).toHaveBeenCalledOnce();
        expect(request).toHaveBeenCalledOnce();
    });

    it("completes a potentially acquired lease after losing the begin response", async () => {
        const warning = vi.spyOn(console, "warn").mockImplementation(() => {});
        const request = vi.fn().mockRejectedValueOnce(new DOMException("deadline", "TimeoutError"))
            .mockResolvedValueOnce(Response.json({ status: "ok" }));
        await expect(withFederationSaveBatch("trail", "trail", async () => "saved", request, "same-batch")).resolves.toBe("saved");
        expect(request.mock.calls.map(([url]) => String(url).split("/").at(-1))).toEqual(["begin", "complete"]);
        expect(request.mock.calls.map(([, options]) => JSON.parse(String(options.body)).batchId)).toEqual(["same-batch", "same-batch"]);
        warning.mockRestore();
    });
});
