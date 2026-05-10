
export interface BaseIntegration {
    active: boolean
}

export interface IntegrationMergeSettings {
    enabled: boolean;
}

export interface StravaIntegration extends BaseIntegration {
    clientId: string | number;
    clientSecret?: string;
    routes: boolean;
    activities: boolean;
    accessToken?: string;
    refreshToken?: string;
    expiresAt?: number;
    after?: string
    privacy: "original" | "settings"
    merge: IntegrationMergeSettings;
}

export interface KomootIntegration extends BaseIntegration {
    email: string,
    password: string,
    completed: boolean,
    planned: boolean
    privacy: "original" | "settings"
    merge: IntegrationMergeSettings;
}

export interface HammerheadIntegration extends BaseIntegration {
    email: string,
    password: string,
    completed: boolean,
    planned: boolean,
    after?: string
    merge: IntegrationMergeSettings;
}


export class Integration {
    id?: string;
    user: string;
    strava?: StravaIntegration | null;
    komoot?: KomootIntegration | null;
    hammerhead?: HammerheadIntegration | null;

    constructor(user: string, strava?: StravaIntegration, komoot?: KomootIntegration, hammerhead?: HammerheadIntegration) {
        this.user = user;
        this.strava = strava;
        this.komoot = komoot;
        this.hammerhead = hammerhead;
    }
}
