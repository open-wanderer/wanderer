import type { Trail } from "#lib/models/trail.ts";

export function initialTrailFormValues(trail: Trail, defaults: { public: boolean; category: string }) {
    return {
        ...copyTrailEditorValues(trail),
        completed_at: completionDateValue(trail.completed_at),
        public: trail.id ? trail.public : defaults.public,
        category: trail.category || defaults.category,
        subcategory: trail.subcategory || "",
    };
}

export function completionDateValue(value?: string): string | undefined {
    return value?.substring(0, 10) || undefined;
}

// Felte can retain string arrays passed to setFields. Give the editor its own
// mutable values while preserving File/Blob identity and prototypes for
// expanded GPX models.
export function copyTrailEditorValues<T>(value: T): T {
    if (value instanceof Blob) return value;
    if (value instanceof Date) return new Date(value) as T;
    if (Array.isArray(value)) return value.map(copyTrailEditorValues) as T;
    if (value !== null && typeof value === "object") {
        return Object.assign(
            Object.create(Object.getPrototypeOf(value)),
            Object.fromEntries(Object.entries(value).map(([key, field]) => [key, copyTrailEditorValues(field)])),
        );
    }
    return value;
}
