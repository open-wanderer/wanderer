import type { User } from "../user";

export interface Actor {
    id?: string;
    username: string;
    preferred_username: string;
    domain?: string;
    summary?: string;
    published?: string;
    follower_count?: number,
    following_count?: number,
    iri: string;
    inbox: string;
    outbox?: string;
    icon?: string;
    followers?: string;
    following?: string;
    is_local: boolean;
    public_key: string;
    last_fetched: string;
    user?: string
    expand?: {
        user?: User
    }
}

export interface ActorSearchResult {
    id: string;
    username: string;
    preferred_username: string;
    domain: string;
    is_local: boolean;
    iri: string;
    icon?: string;
}