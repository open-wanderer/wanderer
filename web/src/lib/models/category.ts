interface Category {
    id: string;
    name: string;
    img: string;
    thresholds?: Threshold[] | null;
}

class Threshold {
    speed: "relaxed" | "moderate" | "medium" | "fast" | "expert";
    type: "distance" | "duration" | "elevation";
    difficulty: "easy" | "moderate" | "difficult";
    limit: number;

    constructor(
        speed: "relaxed" | "moderate" | "medium" | "fast" | "expert",
        type: "distance" | "duration" | "elevation",
        difficulty: "easy" | "moderate" | "difficult",
        limit: number,
    ) {
        this.speed = speed;
        this.type = type;
        this.difficulty = difficulty;
        this.limit = limit;
    }
}

export type {Category}
export { Threshold };