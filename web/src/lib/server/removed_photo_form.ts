import { hasRemovedPhotoKeys, isPhotoField } from "#lib/util/removed_photo_fields.ts";

export function hasRemovedPhotoFields(data: FormData): boolean {
    if ([...data.keys()].some(isPhotoField)) return true;
    return data.getAll("@jsonPayload").some(value => {
        if (typeof value !== "string") return false;
        try {
            const payload = JSON.parse(value);
            return hasRemovedPhotoKeys(payload);
        } catch {
            return false;
        }
    });
}
