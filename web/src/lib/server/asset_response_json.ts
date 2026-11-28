// Keep canonical link expands on the wire; their derived asset arrays are
// useful in an editor but duplicate every asset in an API response.
export function assetResponseJson(data: unknown, init?: ResponseInit): Response {
    const expands = new WeakSet<object>();
    const headers = new Headers(init?.headers);
    if (!headers.has("Content-Type")) headers.set("Content-Type", "application/json");
    const body = JSON.stringify(data, function (key, value) {
        if (key === "expand" && typeof this.id === "string" && value && typeof value === "object") expands.add(value);
        if (expands.has(this) && ["assets_via_trail", "assets_via_waypoint", "assets_via_summit_log"].includes(key)) return undefined;
        return value;
    });
    return new Response(body, { ...init, headers });
}
