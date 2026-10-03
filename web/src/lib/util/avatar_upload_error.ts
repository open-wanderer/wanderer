import { APIError } from "$lib/util/api_util";

type AvatarUploadErrorMessage = {
    key: string;
    values?: {
        maxSize: number;
        unit: "MiB" | "KiB" | "bytes";
    };
};

const fallback = { key: "error-updating-avatar" };

// Map backend error codes to translated messages.
export function avatarUploadErrorMessage(error: unknown): AvatarUploadErrorMessage {
    if (error instanceof APIError) {
        switch (error.status) {
            case 401:
                return { key: "error-updating-avatar-sign-in" };
            case 403:
                return { key: "error-updating-avatar-forbidden" };
            case 408:
            case 504:
                return { key: "error-updating-avatar-timeout" };
            case 413:
                return { key: "error-updating-avatar-too-large" };
            case 429:
                return { key: "error-updating-avatar-rate-limited" };
        }

        if (error.status >= 500 && error.status <= 599) {
            return { key: "error-updating-avatar-server" };
        }

        if (error.status === 400) {
            const data = record(record(error.detail)?.data);
            const avatar = record(data?.avatar);

            if (avatar?.code === "validation_invalid_mime_type") {
                return { key: "error-updating-avatar-unsupported-type" };
            }

            if (avatar?.code === "validation_file_size_limit") {
                const maxBytes = record(avatar.params)?.maxSize;
                if (
                    typeof maxBytes === "number" &&
                    Number.isSafeInteger(maxBytes) &&
                    maxBytes > 0
                ) {
                    // Exact multiples use readable units without rounding the limit up.
                    const divisor =
                        maxBytes % (1024 * 1024) === 0
                            ? 1024 * 1024
                            : maxBytes % 1024 === 0 ? 1024 : 1;
                    return {
                        key: "error-updating-avatar-too-large-with-limit",
                        values: {
                            maxSize: maxBytes / divisor,
                            unit: divisor === 1024 * 1024
                                ? "MiB"
                                : divisor === 1024 ? "KiB" : "bytes",
                        },
                    };
                }
                return { key: "error-updating-avatar-too-large" };
            }
        }
    } else if (error instanceof TypeError) {
        // Browser fetch rejects with TypeError when the request cannot complete.
        return { key: "error-updating-avatar-network" };
    }

    return fallback;
}

function record(value: unknown): Record<string, unknown> | undefined {
    return value !== null && typeof value === "object" && !Array.isArray(value)
        ? (value as Record<string, unknown>)
        : undefined;
}
