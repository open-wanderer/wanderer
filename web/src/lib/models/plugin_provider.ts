export interface ConfigFieldOption {
    value: string;
}

export interface ConfigField {
    key: string;
    type: "boolean" | "date" | "select" | "text" | "url";
    options?: ConfigFieldOption[];
    default?: unknown;
}

export interface PluginProvider {
    id: string;
    name: string;
    description?: string;
    icon?: string;
    iconDark?: string;
    version?: string;
    protocolVersion?: string;
    risk?: string;
    auth: {
        type: string;
        fields?: string[];
        secretFields?: string[];
        authorizationUrl?: string;
        tokenUrl?: string;
        tokenRequestFormat?: "json" | "form";
        scopes?: string[];
        scopeSeparator?: string;
        authorizationParams?: Record<string, string>;
        pkce?: boolean;
        tokenAuth?: string;
    };
    configSchema?: ConfigField[];
    capabilities?: string[];
    limits?: {
        recommendedBatchSize?: number;
    };
    status: "available" | "disabled" | "error";
    error?: string;
}
