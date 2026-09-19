import { describe, expect, it } from "vitest";
import {
    decodeRoutingProfileText,
    encodeRoutingProfileText,
    routingProfileDownloadFilename,
} from "./routing_profile_util";

describe("routing profile text encoding", () => {
    it("round-trips UTF-8 profile content", () => {
        const content = "# Persönliches Profil\nparameter = größe\n";
        expect(decodeRoutingProfileText(encodeRoutingProfileText(content))).toBe(content);
    });

    it("uses the original upload filename for downloads", () => {
        expect(routingProfileDownloadFilename("folder/original.brf", "Renamed", ".brf"))
            .toBe("original.brf");
    });

    it("builds a safe filename from the editable profile name", () => {
        expect(routingProfileDownloadFilename(undefined, "My: Profile", ".brf"))
            .toBe("My- Profile.brf");
        expect(routingProfileDownloadFilename(undefined, "already.brf", ".brf"))
            .toBe("already.brf");
    });
});
