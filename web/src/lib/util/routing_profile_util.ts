export function encodeRoutingProfileText(content: string): string {
    return bytesToBase64(new TextEncoder().encode(content));
}

export function decodeRoutingProfileText(contentBase64: string): string {
    const binary = atob(contentBase64);
    const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0));
    return new TextDecoder().decode(bytes);
}

export function routingProfileDownloadFilename(
    originalFilename: unknown,
    profileName: string,
    extension = ".txt",
): string {
    const original = typeof originalFilename === "string"
        ? originalFilename.split(/[\\/]/).at(-1)?.trim()
        : "";
    if (original) return safeFilename(original);

    const base = safeFilename(profileName.trim() || "routing-profile");
    const suffix = normalizedExtension(extension);
    return suffix && !base.toLowerCase().endsWith(suffix.toLowerCase())
        ? `${base}${suffix}`
        : base;
}

function bytesToBase64(bytes: Uint8Array): string {
    let binary = "";
    for (let offset = 0; offset < bytes.length; offset += 8192) {
        binary += String.fromCharCode(...bytes.subarray(offset, offset + 8192));
    }
    return btoa(binary);
}

function safeFilename(value: string): string {
    return value.replace(/[<>:"|?*\u0000-\u001f]/g, "-") || "routing-profile";
}

function normalizedExtension(value: string): string {
    const extension = value.trim().replace(/^\.+/, "").replace(/[^a-zA-Z0-9_-]/g, "");
    return extension ? `.${extension}` : "";
}
