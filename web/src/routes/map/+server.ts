import { redirect, type RequestHandler } from "@sveltejs/kit";

// Keep existing map bookmarks and coordinate links as shortcuts to map view.
export const GET: RequestHandler = ({ url }) => {
    const params = new URLSearchParams(url.searchParams);
    params.set("view", "map");
    redirect(307, `/trails?${params}`);
};
