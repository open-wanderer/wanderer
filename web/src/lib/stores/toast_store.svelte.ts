import cryptoRandomString from "crypto-random-string";

export type Toast = {
    id?: string;
    type: "info" | "success" | "warning" | "error"
    icon: string;
    text: string;
}

class ToastStore {
    toasts: Toast[] = $state([]);
}

export const toastStore = new ToastStore()

type ToastTimer = {
    handle?: ReturnType<typeof setTimeout>;
    remaining: number;
    started: number;
};

const timers = new Map<string, ToastTimer>();

export function show_toast(newToast: Toast, duration: number = 3000) {
    newToast.id = cryptoRandomString({ length: 15 })
    toastStore.toasts.push(newToast);
    timers.set(newToast.id, { remaining: duration, started: 0 });
    resumeToast(newToast);
}

export function closeToast(toast: Toast) {
    const timer = timers.get(toast.id ?? "");
    clearTimeout(timer?.handle);
    timers.delete(toast.id ?? "");
    const toastIndex = toastStore.toasts.findIndex(t => t.id == toast.id)
    // The timer can fire after the toast was closed by hand; splice(-1) would
    // then remove the newest toast instead.
    if (toastIndex < 0) {
        return;
    }
    toastStore.toasts.splice(toastIndex, 1)
}

// pauseToast keeps a toast open while the pointer or keyboard focus is on it,
// so there is time to read it or reach its close button.
export function pauseToast(toast: Toast) {
    const timer = timers.get(toast.id ?? "");
    if (!timer?.handle) {
        return;
    }
    clearTimeout(timer.handle);
    timer.handle = undefined;
    timer.remaining -= Date.now() - timer.started;
}

export function resumeToast(toast: Toast) {
    const timer = timers.get(toast.id ?? "");
    if (!timer || timer.handle) {
        return;
    }
    timer.started = Date.now();
    timer.handle = setTimeout(() => closeToast(toast), Math.max(timer.remaining, 0));
}
