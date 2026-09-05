export function withShareToken(
    path: string,
    searchParams: URLSearchParams,
): string {
    const share = searchParams.get("share");
    return share ? `${path}?${new URLSearchParams({ share })}` : path;
}
