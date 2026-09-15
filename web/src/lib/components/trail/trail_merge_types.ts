export interface MergeSettings {
    summitLog: boolean;
    photos: boolean;
    comments: boolean;
    delete: boolean;
    tags: boolean;
    likes: boolean;
}

export interface MergeSelection {
    targetTrail: import("$lib/models/trail").Trail;
    sourceTrails: import("$lib/models/trail").Trail[];
}
