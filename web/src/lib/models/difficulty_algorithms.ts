class DifficultyAlgorithm {
  id: string;
  name: string;
  thresholds?: Threshold[] | null;

  constructor(
    id: string,
    name: string,
    params?: {
      thresholds?: Threshold[]
    }
  ) {
    this.id = id;
    this.name = name;
    this.thresholds = params?.thresholds;
  }
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


export { DifficultyAlgorithm };
export { Threshold };
