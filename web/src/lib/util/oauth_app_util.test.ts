import { describe, expect, it } from "vitest";
import { appCallbackUrl, httpUrlOrNull } from "./oauth_app_util";

const params = new URLSearchParams({ code: "abc", state: "s" });

describe("appCallbackUrl", () => {
    it("names the app's package on Android", () => {
        expect(appCallbackUrl(params, "Mozilla/5.0 (Linux; Android 14) Chrome/120"))
            .toBe("intent://oauth-callback?code=abc&state=s#Intent;scheme=wanderer;package=com.openwanderer.wanderer;end");
    });

    it("uses the scheme elsewhere", () => {
        expect(appCallbackUrl(params, "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0)")).toBe("wanderer://oauth-callback?code=abc&state=s");
    });
});

describe("httpUrlOrNull", () => {
    it.each(["https://idp.example/help", "http://idp.example/help?a=b"])("keeps %s", (url) => {
        expect(httpUrlOrNull(url)).toBe(url);
    });

    it.each([
        null,
        "",
        "javascript:alert(document.cookie)",
        "JaVaScRiPt:alert(1)",
        " javascript:alert(1)",
        "java\tscript:alert(1)",
        "data:text/html,<script>alert(1)</script>",
        "vbscript:x",
        "/relative",
        "//idp.example/help",
    ])("refuses %s", (url) => {
        expect(httpUrlOrNull(url)).toBeNull();
    });
});
