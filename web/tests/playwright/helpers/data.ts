function uid(): string {
    return `${Date.now()}-${Math.floor(Math.random() * 1_000_000)}`;
}

export function trailName(): string {
    return `Test Trail ${uid()}`;
}

export function listName(): string {
    return `Test List ${uid()}`;
}

export function commentText(): string {
    return `Test Comment ${uid()}`;
}
