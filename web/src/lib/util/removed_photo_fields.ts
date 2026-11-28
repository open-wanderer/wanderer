export const isPhotoField = (key: string): boolean => key.replace(/^[+-]+|[+-]+$/g, "") === "photos";

export function hasRemovedPhotoKeys(data: unknown): boolean {
    return data !== null && typeof data === "object" && !Array.isArray(data)
        && Object.keys(data).some(isPhotoField);
}
