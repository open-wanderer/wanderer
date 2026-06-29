export interface PluginInstance {
    id?: string;
    user: string;
    plugin_id: string;
    enabled: boolean;
    auth?: Record<string, string>;
    config?: Record<string, unknown>;
    state?: Record<string, unknown>;
    status:
        | "configured"
        | "needs_auth"
        | "needs_reauth"
        | "syncing"
        | "rate_limited"
        | "unavailable"
        | "unsupported_protocol"
        | "error"
        | "disabled";
    last_error?: {
        code?: string;
        message?: string;
    };
    last_sync_at?: string;
    retry_not_before?: string;
}
