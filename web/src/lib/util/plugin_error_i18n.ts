import { get } from "svelte/store";
import { _ } from "svelte-i18n";
import { APIError } from "$lib/util/api_util";

type PluginErrorLike = {
    code?: unknown;
    message?: unknown;
};

const credentialErrorCodes = new Set(["auth_failed"]);
const reconnectErrorCodes = new Set(["invalid_grant", "unauthorized"]);
const unavailableErrorCodes = new Set(["provider_unavailable", "temporary_unavailable"]);
const internalErrorCodes = new Set(["internal_error", "plugin_error"]);

const pluginSetupErrorKeys = new Map<string, string>([
    ["manifest_missing", "plugin-setup-error-manifest-missing"],
    ["manifest_unreadable", "plugin-setup-error-manifest-unreadable"],
    ["manifest_invalid", "plugin-setup-error-manifest-invalid"],
    ["runtime_entrypoint_invalid", "plugin-setup-error-runtime-entrypoint-invalid"],
    ["runtime_entrypoint_missing", "plugin-setup-error-runtime-entrypoint-missing"],
    ["runtime_entrypoint_unreadable", "plugin-setup-error-runtime-entrypoint-unreadable"],
    ["setup_failed", "plugin-setup-error-details"],
]);

const authErrorHints = [
    "auth_failed",
    "login failed",
    "email and password are required",
];

export function translatePluginError(code?: string, message?: string): string {
    const normalizedCode = code?.trim();

    if (normalizedCode && credentialErrorCodes.has(normalizedCode)) {
        return get(_)("wrong-username-or-password");
    }
    if (normalizedCode && reconnectErrorCodes.has(normalizedCode)) {
        return get(_)("plugin-error-reconnect-required");
    }
    if (normalizedCode === "rate_limited") {
        return get(_)("plugin-error-rate-limited");
    }
    if (normalizedCode && unavailableErrorCodes.has(normalizedCode)) {
        return get(_)("plugin-error-provider-unavailable");
    }
    if (normalizedCode === "invalid_request") {
        return get(_)("plugin-error-invalid-request");
    }
    if (normalizedCode && internalErrorCodes.has(normalizedCode)) {
        return get(_)("plugin-error-internal");
    }

    return message?.trim() || get(_)("plugin-setup-error");
}

// Setup error codes come from plugin discovery and are deliberately mapped
// through an allowlist so backend diagnostics never become user-facing text.
export function pluginSetupErrorKey(code: unknown): string {
    if (typeof code !== "string") {
        return "plugin-setup-error-details";
    }
    return pluginSetupErrorKeys.get(code) ?? "plugin-setup-error-details";
}

export function translatePluginAPIError(error: unknown, fallback: string): string {
    if (error instanceof APIError) {
        const pluginError = extractPluginError(error.detail);
        if (pluginError?.code) {
            return translatePluginError(String(pluginError.code), stringValue(pluginError.message));
        }

        const raw = `${error.message} ${stringValue(error.detail)}`.toLowerCase();
        if (authErrorHints.some((hint) => raw.includes(hint))) {
            return get(_)("wrong-username-or-password");
        }
    }

    if (error instanceof Error && error.message) {
        return error.message;
    }

    return fallback;
}

function extractPluginError(value: unknown): PluginErrorLike | undefined {
    if (!value) {
        return undefined;
    }

    if (typeof value === "string") {
        return extractPluginErrorFromString(value);
    }

    if (typeof value !== "object") {
        return undefined;
    }

    const record = value as Record<string, unknown>;
    if (typeof record.code === "string") {
        return record;
    }

    for (const nested of Object.values(record)) {
        const parsed = extractPluginError(nested);
        if (parsed) {
            return parsed;
        }
    }

    return undefined;
}

function extractPluginErrorFromString(value: string): PluginErrorLike | undefined {
    try {
        const parsed = JSON.parse(value);
        return extractPluginError(parsed);
    } catch {
        const match = value.match(/"code"\s*:\s*"([^"]+)"/);
        return match ? { code: match[1] } : undefined;
    }
}

function stringValue(value: unknown): string | undefined {
    if (typeof value === "string") {
        return value;
    }
    if (value == null) {
        return undefined;
    }
    return JSON.stringify(value);
}
