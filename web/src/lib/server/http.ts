import { json } from "@sveltejs/kit";

function safeJson(text: string): any {
    try {
        return JSON.parse(text);
    } catch {
        return { message: text };
    }
}

export async function proxyJsonResponse(response: Response) {
    const text = await response.text();
    const payload = text.length ? safeJson(text) : {};
    if (!response.ok) {
        return json(payload, { status: response.status });
    }
    return json(payload);
}
