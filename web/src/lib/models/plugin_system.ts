export interface PluginSystemCapability {
    name: string;
    version: string;
    export: string;
    requiredHostFunctions?: string[];
    job?: string;
}

export interface PluginSystemManifest {
    manifestVersion: string;
    id: string;
    type: "trails";
    name: string;
    displayName?: string;
    description?: string;
    icon?: string;
    iconDark?: string;
    version: string;
    runtime: {
        type: string;
        entrypoint: string;
    };
    capabilities: PluginSystemCapability[];
    auth?: Record<string, unknown>;
    permissions?: Record<string, unknown>;
    configSchema?: unknown[];
    hostConfig?: Record<string, unknown>;
    metadata?: Record<string, unknown>;
}

export interface PluginSystemPlugin {
    id: string;
    type: "trails";
    name: string;
    displayName?: string;
    description?: string;
    icon?: string;
    iconDark?: string;
    version: string;
    runtime: string;
    capabilities: string[];
    status: "available" | "disabled" | "error";
    error?: string;
    manifest: PluginSystemManifest;
}
