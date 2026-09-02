import type { Trail } from "./trail";
import type { ListShare } from "./list_share";
import type { UserAnonymous } from "./user";
import type { Actor } from "./activitypub/actor";

export class List {
    id?: string;
    name: string;
    public: boolean;
    description?: string;
    elevation_gain?: number;
    elevation_loss?: number;
    distance?: number;
    duration?: number;
    avatar?: string;
    trails?: string[];
    iri?: string;
    expand?: {
        trails?: Trail[]
        list_share_via_list?: ListShare[]
        author?: Actor;

    }
    created?: string;
    updated?: string;
    author: string;

    constructor(name: string, trails: Trail[], params?: { description?: string, public?: boolean, avatar?: string, author?: string }) {
        this.name = name;
        this.public = params?.public ?? false
        this.expand = { trails: trails };
        this.trails = trails.map(t => t.id!);
        this.description = params?.description ?? "";
        this.avatar = params?.avatar;
        this.author = params?.author ?? "000000000000000";
    }
}

export interface ListFilter {
    q: string,
    sort?: "name" | "size" | "created";
    author?: string;
    public?: boolean;
    shared?: boolean;
    sortOrder?: "+" | "-"
}