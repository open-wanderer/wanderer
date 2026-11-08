export async function forEachConcurrent<T>(values: T[], limit: number, work: (value: T) => Promise<void>): Promise<void> {
    let next = 0;
    await Promise.all(Array.from({ length: Math.min(values.length, Math.max(1, limit)) }, async () => {
        while (next < values.length) {
            const index = next++;
            await work(values[index]);
        }
    }));
}
