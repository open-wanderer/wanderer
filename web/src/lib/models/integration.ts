
export interface BaseIntegration {
    active: boolean
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
}

export interface KomootIntegration extends BaseIntegration {
    email: string,
    password: string,
    completed: boolean,
    planned: boolean
    privacy: "original" | "settings"
}


export class Integration {
    id?: string;
    user: string;
    strava?: StravaIntegration | null;
    komoot?: KomootIntegration | null

    constructor(user: string, strava?: StravaIntegration, komoot?: KomootIntegration) {
        this.user = user;
        this.strava = strava;
        this.komoot = komoot;
    }
}