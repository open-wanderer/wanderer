const MAX_TRACKED_KEYS = 10_000;

type Window = { count: number; resetAt: number };

/**
 * A fixed-window limiter, per process. Returns a function that counts a request
 * for `key`, and answers undefined when it may proceed, or the seconds to wait.
 */
export function createRateLimiter(limit: number, windowMs: number) {
    const windows = new Map<string, Window>();

    return (key: string): number | undefined => {
        const now = Date.now();
        let window = windows.get(key);
        if (!window || window.resetAt <= now) {
            if (windows.size >= MAX_TRACKED_KEYS) {
                for (const [tracked, { resetAt }] of windows) {
                    if (resetAt <= now) windows.delete(tracked);
                }
                // Every entry is still live, so many addresses are in use at once.
                if (windows.size >= MAX_TRACKED_KEYS) windows.clear();
            }
            window = { count: 0, resetAt: now + windowMs };
            windows.set(key, window);
        }
        window.count++;
        return window.count > limit ? Math.ceil((window.resetAt - now) / 1000) : undefined;
    };
}
