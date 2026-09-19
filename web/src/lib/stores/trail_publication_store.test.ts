import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { get } from "svelte/store";
import { trailPublications, trails_publish, trails_resume_publication, type TrailPublication } from "./trail_publication_store";

vi.mock("$app/environment", () => ({ browser: true }));

const job: TrailPublication = { id: "job", trailId: "trail", status: "running", total: 3, processed: 0, failed: 0 };

beforeEach(() => {
    vi.useFakeTimers();
    trailPublications.set({});
});
afterEach(() => vi.useRealTimers());

describe("trail publication monitoring", () => {
    it("reports progress, polls with increasing intervals, and stops on completion", async () => {
        const request = vi.fn()
            .mockResolvedValueOnce(Response.json(job))
            .mockResolvedValueOnce(Response.json({ ...job, processed: 2 }))
            .mockResolvedValueOnce(Response.json({ ...job, status: "completed", processed: 3 }));
        const result = trails_publish("trail", request);
        await vi.advanceTimersByTimeAsync(0);
        expect(get(trailPublications).trail.processed).toBe(0);
        await vi.advanceTimersByTimeAsync(999);
        expect(request).toHaveBeenCalledTimes(1);
        await vi.advanceTimersByTimeAsync(1);
        expect(get(trailPublications).trail.processed).toBe(2);
        await vi.advanceTimersByTimeAsync(1499);
        expect(request).toHaveBeenCalledTimes(2);
        await vi.advanceTimersByTimeAsync(1);
        await expect(result).resolves.toMatchObject({ status: "completed" });
        await vi.advanceTimersByTimeAsync(60000);
        expect(request).toHaveBeenCalledTimes(3);
        expect(request.mock.calls.map(([, options]) => options.method)).toEqual(["POST", "GET", "GET"]);
    });

    it("resumes a running job after reload without another publication POST", async () => {
        const request = vi.fn()
            .mockResolvedValueOnce(Response.json(job))
            .mockResolvedValueOnce(Response.json({ ...job, status: "completed", processed: 3 }));
        const result = trails_resume_publication("trail", request);
        await vi.advanceTimersByTimeAsync(1000);
        await expect(result).resolves.toMatchObject({ status: "completed" });
        expect(request.mock.calls.every(([, options]) => options.method === "GET")).toBe(true);
    });

    it("starts publication even while the initial status probe is pending", async () => {
        let resolveProbe!: (response: Response) => void;
        const request = vi.fn(async (_url: RequestInfo | URL, options?: RequestInit) => {
            if (options?.method === "GET") return new Promise<Response>((resolve) => { resolveProbe = resolve; });
            return Response.json({ ...job, status: "completed" });
        });
        const probe = trails_resume_publication("trail", request);
        const publication = trails_publish("trail", request);
        await expect(publication).resolves.toMatchObject({ status: "completed" });
        expect(request).toHaveBeenCalledWith("/api/v1/trail/trail/publication", { method: "POST" });
        resolveProbe(new Response(null, { status: 404 }));
        await expect(probe).resolves.toBeNull();
    });

    it("shares a running publication monitor between save and progress panel", async () => {
        const request = vi.fn()
            .mockResolvedValueOnce(Response.json(job))
            .mockResolvedValueOnce(Response.json({ ...job, status: "completed" }));
        const first = trails_publish("trail", request);
        const second = trails_resume_publication("trail", request);
        await vi.advanceTimersByTimeAsync(1000);
        await Promise.all([first, second]);
        expect(request).toHaveBeenCalledTimes(2);
    });

    it("preserves a failed job and its actionable error for retry", async () => {
        const request = vi.fn().mockResolvedValue(Response.json({ ...job, status: "failed", failed: 1, error: "asset_publish_photo_too_large" }));
        await expect(trails_publish("trail", request)).rejects.toMatchObject({ message: "asset_publish_photo_too_large" });
        expect(get(trailPublications).trail).toMatchObject({ status: "failed", failed: 1 });
    });

    it("stops polling after three connection failures without claiming the job failed", async () => {
        const request = vi.fn().mockResolvedValueOnce(Response.json(job)).mockRejectedValue(new Error("offline"));
        const result = trails_publish("trail", request).catch((error) => error);
        await vi.runAllTimersAsync();
        expect(await result).toMatchObject({ message: "asset_publish_status_failed" });
        expect(request).toHaveBeenCalledTimes(4);
        expect(get(trailPublications).trail).toMatchObject({ status: "running", monitoringError: true });
        expect(vi.getTimerCount()).toBe(0);
    });

    it("clears a connection warning when checking a job that completed in the meantime", async () => {
        trailPublications.set({ trail: { ...job, name: "Mountain trail", monitoringError: true } });
        const request = vi.fn().mockResolvedValue(Response.json({ ...job, status: "completed", processed: 3 }));
        await trails_resume_publication("trail", request);
        expect(get(trailPublications).trail).toMatchObject({ status: "completed", name: "Mountain trail" });
        expect(get(trailPublications).trail.monitoringError).toBeUndefined();
    });

    it("does not show historic completed jobs as a new background task", async () => {
        const request = vi.fn().mockResolvedValue(Response.json({ ...job, status: "completed" }));
        await trails_resume_publication("trail", request);
        expect(get(trailPublications)).toEqual({});
    });

    it.each([true, false])("checks persisted visibility when a job disappears after restart (public=%s)", async (isPublic) => {
        const request = vi.fn()
            .mockResolvedValueOnce(Response.json(job))
            .mockResolvedValueOnce(new Response(null, { status: 404 }))
            .mockResolvedValueOnce(Response.json({ public: isPublic }));
        const result = trails_publish("trail", request).catch((error) => error);
        await vi.runAllTimersAsync();
        expect(await result).toMatchObject(isPublic ? { status: "completed" } : { message: "asset_publish_interrupted" });
        expect(request).toHaveBeenLastCalledWith("/api/v1/trail/trail", { method: "GET" });
        expect(get(trailPublications).trail.status).toBe(isPublic ? "completed" : "failed");
    });
});
