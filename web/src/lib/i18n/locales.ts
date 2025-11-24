export const SUPPORTED_LOCALES = [
  "en",
  "de",
  "es",
  "eu",
  "fr",
  "hu",
  "it",
  "nl",
  "pl",
  "pt",
  "ru",
  "zh",
] as const;

export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number];

export const defaultLocale: SupportedLocale = "en";

export function normalizeLocale(raw: string | null | undefined): SupportedLocale {
  if (!raw) return defaultLocale;

  // remove spaces and any ";q=0.7" part
  let tag = raw.trim().split(";")[0];

  // Turn "de_CH" into "de-CH"
  tag = tag.replace("_", "-");

  // Lowercase language, uppercase region if present
  const parts = tag.split("-");
  if (parts.length === 2) {
    tag = `${parts[0].toLowerCase()}-${parts[1].toUpperCase()}`;
  } else {
    tag = tag.toLowerCase();
  }

  // Map "de-CH" -> "de", "en-US" -> "en", etc.
  const base = tag.split("-")[0] as SupportedLocale;

  if (SUPPORTED_LOCALES.includes(base)) {
    return base;
  }

  return defaultLocale;
}
