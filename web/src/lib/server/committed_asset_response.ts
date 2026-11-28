import { handleError } from "#lib/util/api_util.ts";

// A response refresh can fail after the write committed. Keep the error and
// disclose the saved identity, without presenting truncated expands as complete.
export async function committedAssetResponseError(error: unknown, record?: { id?: string; photos?: string[]; expand?: unknown }): Promise<Response> {
    const response = handleError(error);
    if (!record?.id) return response;
    const { photos: _photos, expand: _expand, ...savedRecord } = record;
    const body = await response.json();
    const headers = new Headers(response.headers);
    headers.delete("content-length");
    return Response.json({ ...body, detail: { ...body.detail, recordCommitted: true, savedRecord } }, { status: response.status, headers });
}
