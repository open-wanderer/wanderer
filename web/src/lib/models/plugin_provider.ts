export type LocalizedTextMap = Record<string, string>;

export interface ConfigFieldOption {
    value: string;
    label?: string;
    labels?: LocalizedTextMap;
}

export interface ConfigField {
    key: string;
    type: "boolean" | "date" | "select" | "text" | "url";
    label?: string;
    labels?: LocalizedTextMap;
    description?: string;
    descriptions?: LocalizedTextMap;
    options?: ConfigFieldOption[];
    default?: unknown;
    required?: boolean;
    hidden?: boolean;
}

export interface PluginProvider {
    id: string;
    type: "trails";
    name: string;
    displayName?: string;
    displayNames?: LocalizedTextMap;
    description?: string;
    descriptions?: LocalizedTextMap;
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
    hostConfig?: Record<string, unknown>;
    metadata?: Record<string, unknown>;
    capabilities?: string[];
    limits?: {
        recommendedBatchSize?: number;
    };
    status: "available" | "disabled" | "error";
    error?: string;
}
