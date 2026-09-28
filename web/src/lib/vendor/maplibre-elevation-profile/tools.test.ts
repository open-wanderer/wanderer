import { describe, expect, it } from 'vitest';
import { haversineCumulatedDistanceWgs84, smoothElevations } from './tools';
import type { Position } from 'geojson';

describe('tools', () => {
    describe('haversineCumulatedDistanceWgs84', () => {
        it('returns empty array when path has less than 2 coordinates', () => {
            expect(haversineCumulatedDistanceWgs84([])).toEqual([]);
            expect(haversineCumulatedDistanceWgs84([[0, 0]])).toEqual([]);
        });

        it('calculates cumulated distance correctly', () => {
            const path: Position[] = [
                [2.3522, 48.8566],
                [2.3522, 48.8666],
            ];
            const distances = haversineCumulatedDistanceWgs84(path);
            expect(distances).toHaveLength(2);
            expect(distances[0]).toBe(0);
            // 0.01 degrees of latitude is ~1111.95 m on the WGS84 mean radius.
            expect(distances[1]).toBeCloseTo(1111.95, 1);
        });
    });

    describe('smoothElevations', () => {
        it('returns the input array untouched when windowSize < 1', () => {
            const positions: Position[] = [[0, 0, 100], [0, 1, 200]];
            // The same array comes back, so compare against a separate literal
            // as well: toEqual on the identical reference can never fail.
            expect(smoothElevations(positions, 0)).toBe(positions);
            expect(smoothElevations(positions, -1)).toBe(positions);
            expect(positions).toEqual([[0, 0, 100], [0, 1, 200]]);
        });

        it('handles empty array', () => {
            expect(smoothElevations([], 5)).toEqual([]);
        });

        it('handles single position', () => {
            const single: Position[] = [[2.35, 48.85, 150]];
            const res = smoothElevations(single, 5);
            expect(res).toHaveLength(1);
            expect(res[0][0]).toBe(2.35);
            expect(res[0][1]).toBe(48.85);
            expect(res[0][2]).toBe(150);
        });

        it('keeps constant elevations identical', () => {
            const constant: Position[] = [
                [0, 0, 100],
                [1, 1, 100],
                [2, 2, 100],
                [3, 3, 100],
                [4, 4, 100],
            ];
            const smoothed = smoothElevations(constant, 3);
            for (let i = 0; i < constant.length; i++) {
                expect(smoothed[i][2]).toBeCloseTo(100, 5);
            }
        });

        it('correctly calculates weighted moving average', () => {
            // Test 3 points with windowSize = 3 (half = 1)
            // Points: [0, 0, 10], [0, 1, 20], [0, 2, 30]
            // For i = 1: start = 0, end = 3
            // weights: 1, 2, 3 -> sum of weights = 6
            // weighted sum: 10 * 1 + 20 * 2 + 30 * 3 = 10 + 40 + 90 = 140
            // smoothed elevation = 140 / 6 = 23.3333...
            const points: Position[] = [
                [0, 0, 10],
                [0, 1, 20],
                [0, 2, 30],
            ];
            const smoothed = smoothElevations(points, 3);
            expect(smoothed[1][2]).toBeCloseTo(140 / 6, 5);

            // For i = 0: start = 0, end = 2
            // weights: 1, 2 -> sum = 3
            // weighted sum: 10 * 1 + 20 * 2 = 50 -> 50 / 3 = 16.666...
            expect(smoothed[0][2]).toBeCloseTo(50 / 3, 5);

            // For i = 2: start = 1, end = 3
            // weights: 1, 2 -> sum = 3
            // weighted sum: 20 * 1 + 30 * 2 = 80 -> 80 / 3 = 26.666...
            expect(smoothed[2][2]).toBeCloseTo(80 / 3, 5);
        });

        it('centres an even window on the following point', () => {
            // windowSize 4 gives half = 2, so i = 2 spans [0, 5) and the
            // increasing weights lean the average towards the later points.
            const points: Position[] = [
                [0, 0, 10],
                [0, 1, 20],
                [0, 2, 30],
                [0, 3, 40],
                [0, 4, 50],
            ];
            const smoothed = smoothElevations(points, 4);
            const weighted = 10 * 1 + 20 * 2 + 30 * 3 + 40 * 4 + 50 * 5;
            expect(smoothed[2][2]).toBeCloseTo(weighted / 15, 5);
        });

        it('treats a position without elevation as 0', () => {
            // Nothing in the app reaches this: GPX.toGeoJSON already emits
            // `pt.ele ?? 0`. Kept as a guard so a 2D LineString cannot poison
            // an entire window with NaN.
            const flat: Position[] = [[0, 0], [1, 1], [2, 2]];
            for (const position of smoothElevations(flat, 3)) {
                expect(position[2]).toBe(0);
            }
        });

        // The optimisation's whole premise is that it changes performance and
        // nothing else, so pin it against the implementation it replaced.
        it('is bit-identical to the previous implementation', () => {
            let seed = 42;
            const random = () => (seed = (seed * 1103515245 + 12345) & 0x7fffffff) / 0x7fffffff;

            for (const count of [1, 2, 3, 5, 10, 33, 100, 501]) {
                const positions: Position[] = Array.from({ length: count }, () => [
                    random() * 360 - 180,
                    random() * 170 - 85,
                    random() * 4000 - 400,
                ]);
                for (const windowSize of [1, 2, 3, 4, 7, 15, 99, 1000, Math.ceil(count / 100)]) {
                    const expected = referenceSmoothElevations(positions, windowSize);
                    const actual = smoothElevations(positions, windowSize);
                    expect(actual).toHaveLength(expected.length);
                    for (let i = 0; i < expected.length; i++) {
                        // Object.is so NaN compares equal and -0 does not hide a sign flip.
                        expect(
                            Object.is(actual[i][2], expected[i][2]),
                            `count=${count} windowSize=${windowSize} i=${i}: ${actual[i][2]} != ${expected[i][2]}`,
                        ).toBe(true);
                        expect(actual[i][0]).toBe(expected[i][0]);
                        expect(actual[i][1]).toBe(expected[i][1]);
                    }
                }
            }
        });

        it('matches the previous implementation on degenerate elevations', () => {
            const cases: Record<string, Position[]> = {
                extremes: [[0, 0, 0], [1, 1, -0], [2, 2, 1e308], [3, 3, -1e308], [4, 4, 0.1]],
                nan: [[0, 0, NaN], [1, 1, 1], [2, 2, 2]],
                infinity: [[0, 0, Infinity], [1, 1, 1], [2, 2, 2]],
            };
            for (const [name, positions] of Object.entries(cases)) {
                const expected = referenceSmoothElevations(positions, 3);
                const actual = smoothElevations(positions, 3);
                for (let i = 0; i < expected.length; i++) {
                    expect(
                        Object.is(actual[i][2], expected[i][2]),
                        `${name} i=${i}: ${actual[i][2]} != ${expected[i][2]}`,
                    ).toBe(true);
                }
            }
        });
    });
});

/**
 * The allocating implementation that `smoothElevations` replaced, kept verbatim
 * as the reference for the equivalence tests above.
 */
function referenceSmoothElevations(positions: Position[], windowSize: number): Position[] {
    if (windowSize < 1) {
        return positions;
    }

    return positions.map((pos, i, arr) => {
        const start = Math.max(0, i - Math.floor(windowSize / 2));
        const end = Math.min(arr.length, i + Math.floor(windowSize / 2) + 1);
        const segment = arr.slice(start, end);

        const weights = segment.map((_, idx) => idx + 1);
        const elevations = segment.map(p => p[2]);
        const weightedSum = elevations.reduce((sum, elevation, idx) => sum + elevation * weights[idx], 0);
        const weightTotal = weights.reduce((sum, weight) => sum + weight, 0);

        return [pos[0], pos[1], weightedSum / weightTotal] as Position;
    });
}
