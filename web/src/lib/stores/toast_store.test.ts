import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { closeToast, pauseToast, resumeToast, show_toast, toastStore } from "./toast_store.svelte";

describe("toast_store", () => {
    beforeEach(() => vi.useFakeTimers());
    afterEach(() => {
        for (const toast of [...toastStore.toasts]) {
            closeToast(toast);
        }
        vi.useRealTimers();
    });

    const toast = (text: string) => ({ type: "info" as const, icon: "check", text });

    it("closes a toast after its duration", () => {
        show_toast(toast("saved"));
        vi.advanceTimersByTime(2999);
        expect(toastStore.toasts.map((t) => t.text)).toEqual(["saved"]);
        vi.advanceTimersByTime(1);
        expect(toastStore.toasts).toEqual([]);
    });

    it("leaves other toasts alone when a closed toast's timer would fire", () => {
        const first = toast("first");
        show_toast(first);
        closeToast(first);
        vi.advanceTimersByTime(1000);
        show_toast(toast("second"));
        vi.advanceTimersByTime(2000);
        expect(toastStore.toasts.map((t) => t.text)).toEqual(["second"]);
    });

    it("keeps a toast open while paused and finishes its time after", () => {
        const paused = toast("paused");
        show_toast(paused);
        vi.advanceTimersByTime(1000);
        pauseToast(paused);
        pauseToast(paused);
        vi.advanceTimersByTime(10000);
        expect(toastStore.toasts.map((t) => t.text)).toEqual(["paused"]);
        resumeToast(paused);
        resumeToast(paused);
        vi.advanceTimersByTime(1999);
        expect(toastStore.toasts).toHaveLength(1);
        vi.advanceTimersByTime(1);
        expect(toastStore.toasts).toEqual([]);
    });
});
