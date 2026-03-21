
export class APIToken {
    id?: string;
    name: string;
    expiration?: string;
    last_used?: string;
    user?: string;

    constructor(name: string, expiration?: string) {
        this.name = name;
        this.expiration = expiration;
    }
}