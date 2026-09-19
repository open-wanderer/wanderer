export const TRAIL_COLORS = [
    "#3549bb", // blue
    "#ff7f0e", // orange
    "#2ca02c", // green
    "#d62728", // red
    "#9467bd", // purple
    "#8c564b", // brown
    "#e377c2", // pink
    "#373642", // gray
    "#fae455", // yellow
    "#17becf", // teal
] as const;

export function trailColor(index: number) {
    return TRAIL_COLORS[index % TRAIL_COLORS.length];
}
