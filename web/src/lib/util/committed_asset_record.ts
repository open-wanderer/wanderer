import type { APIError } from "./api_util";

export function committedAssetRecord<T extends { id?: string }>(error: APIError): (Partial<T> & { id: string }) | undefined {
    const record = error.detail?.savedRecord;
    return error.detail?.recordCommitted === true && typeof record?.id === "string" ? record : undefined;
}
