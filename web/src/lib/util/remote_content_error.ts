import { APIError } from "#lib/util/api_util.ts";

export function remoteContentError(error: APIError): Omit<App.Error, "status"> {
    const result: Omit<App.Error, "status"> = { message: error.status === 404 ? "Not found" : error.message };
    if (error.status !== 422) return result;
    const detail = error.detail?.code === "remote_document_limit" ? error.detail : error.detail?.data;
    if (detail?.code !== "remote_document_limit") return result;
    try {
        const origin = new URL(detail.origin_url);
        if ((origin.protocol === "https:" || origin.protocol === "http:") && !origin.username && !origin.password
            && (/^\/trail\/view\/[^/]+\/[^/]+$/.test(origin.pathname) || /^\/lists\/[^/]+\/[^/]+$/.test(origin.pathname))) {
            result.originURL = origin.href;
        }
    } catch {
        // Other errors keep the ordinary error page.
    }
    return result;
}
