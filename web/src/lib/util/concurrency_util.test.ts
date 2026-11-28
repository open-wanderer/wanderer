import { expect, it } from "vitest";
import { forEachConcurrent } from "./concurrency_util";

it("bounds work and exposes completed results before the batch finishes", async () => {
    let active = 0, peak = 0;
    const release: (() => void)[] = [];
    const completed: number[] = [];
    const task = forEachConcurrent([0, 1, 2, 3], 2, async (value) => {
        active++;
        peak = Math.max(peak, active);
        await new Promise<void>((resolve) => release.push(resolve));
        active--;
        completed.push(value);
    });
    expect(release).toHaveLength(2);
    release[0]();
    await Promise.resolve();
    await Promise.resolve();
    expect(completed).toEqual([0]);
    expect(release).toHaveLength(3);
    release[1]();
    await Promise.resolve();
    await Promise.resolve();
    release[2]();
    release[3]();
    await task;
    expect(peak).toBe(2);
    expect(completed).toEqual([0, 1, 2, 3]);
});
