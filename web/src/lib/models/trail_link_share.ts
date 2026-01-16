
export class TrailLinkShare {
    id?: string;
    trail: string;
    token?: string;
    permission: "view" | "edit"


    constructor(trail: string, permission: "view" | "edit") {
        this.trail = trail;
        this.permission = permission
    }
}