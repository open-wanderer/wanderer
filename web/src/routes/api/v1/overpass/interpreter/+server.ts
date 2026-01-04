import { json, type RequestEvent } from "@sveltejs/kit";
import { fetchOverpass } from "$lib/server/overpass";

export async function GET(event: RequestEvent) {
    const data = event.url.searchParams.get("data");
    if (!data) {
        return json({ message: "Missing query parameter: data" }, { status: 400 });
    }

    const params = new URLSearchParams({
        data,
    });

    try {
        const response = await fetchOverpass(event, params);
        const text = await response.text();
        const payload = text.length ? safeJson(text) : {};
        if (!response.ok) {
            return json(payload, { status: response.status });
        }
        return json(payload);
    } catch (error) {
        return json({ message: "Overpass request failed" }, { status: 502 });
    }
}

function safeJson(text: string): any {
    try {
        return JSON.parse(text);
    } catch {
        return { message: text };
    }
}
