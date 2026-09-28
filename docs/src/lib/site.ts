/**
 * Site-wide switches for the marketing pages and the docs chrome.
 *
 * APP_BETA_ANNOUNCEMENT gates every "the app is in beta" placement on the
 * homepage at once: the bar above the hero, the hero pill, and the
 * feature-list link. Docs pages carry no banner. Flip it to false when the
 * app leaves beta; the /app page and the "Using the app" sidebar section
 * stay either way.
 */
export const APP_BETA_ANNOUNCEMENT = true;

/** Where every announcement placement sends people. */
export const APP_BETA_URL = "/app";

/** Bump this to re-show the homepage bar to people who dismissed an older one. */
export const APP_BETA_ANNOUNCEMENT_ID = "app-beta-2026-09";
