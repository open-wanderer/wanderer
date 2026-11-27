import type { PluginInstance } from "#lib/models/plugin_instance.ts";

// An incomplete import does not invalidate OAuth credentials. Keep the existing
// status schema while allowing the user to toggle and save the connected instance.
export function isPluginInstanceConnected(
    instance?: Pick<PluginInstance, "status" | "last_error">,
): boolean {
    return instance?.status === "configured"
        || instance?.status === "syncing"
        || (instance?.status === "error" && instance.last_error?.code === "sync_incomplete");
}
