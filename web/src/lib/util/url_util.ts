export function withShareToken(
    path: string,
    searchParams: Pick<URLSearchParams, "get">,
): string {
    const share = searchParams.get("share");
    return share ? `${path}?${new URLSearchParams({ share })}` : path;
}
