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
