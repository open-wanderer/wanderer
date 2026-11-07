import { describe, expect, it } from "vitest";
import { withParseSlot } from "./parse_slot";

function deferred() {
    let resolve!: () => void;
    const promise = new Promise<void>((r) => (resolve = r));
    return { promise, resolve };
}

describe("withParseSlot", () => {
    it("runs one task at a time, in order", async () => {
        const first = deferred();
        const order: string[] = [];
        const a = withParseSlot(async () => {
            order.push("a start");
            await first.promise;
            order.push("a end");
        });
        const b = withParseSlot(async () => {
            order.push("b");
        });

        await Promise.resolve();
        expect(order).toEqual(["a start"]);
        first.resolve();
        await Promise.all([a, b]);
        expect(order).toEqual(["a start", "a end", "b"]);
    });

    it("frees the slot when a task fails", async () => {
        await expect(withParseSlot(async () => { throw new Error("bad file"); })).rejects.toThrow("bad file");

        await expect(withParseSlot(async () => "next")).resolves.toBe("next");
    });

    it("answers 503 once too many tasks are waiting", async () => {
        const first = deferred();
        const running = withParseSlot(() => first.promise);
        const queued = Array.from({ length: 8 }, () => withParseSlot(async () => "done"));

        await expect(withParseSlot(async () => "too many")).rejects.toMatchObject({ status: 503 });
        first.resolve();
        await running;
        await expect(Promise.all(queued)).resolves.toHaveLength(8);
    });
});
