const APP_ANDROID_PACKAGE = "com.openwanderer.wanderer";

/**
 * Where a confirmed login goes. Any app can claim the wanderer:// scheme, so
 * Android gets an intent URL, which the browser delivers to this package only.
 * iOS has nothing like it; there the user's confirmation is the only safeguard.
 */
export function appCallbackUrl(params: URLSearchParams, userAgent: string) {
    return /Android/i.test(userAgent)
        ? `intent://oauth-callback?${params}#Intent;scheme=wanderer;package=${APP_ANDROID_PACKAGE};end`
        : `wanderer://oauth-callback?${params}`;
}

/** `value` if it is an absolute http(s) URL. Query parameters can hold `javascript:` URLs. */
export function httpUrlOrNull(value: string | null) {
    if (!value || !URL.canParse(value)) {
        return null;
    }
    const url = new URL(value);
    return url.protocol === "http:" || url.protocol === "https:" ? url.href : null;
}
