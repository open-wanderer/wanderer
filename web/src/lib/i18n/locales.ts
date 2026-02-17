type LocaleMessages = Record<string, unknown>;
type LocaleModule = { default: LocaleMessages };
type LocaleLoader = () => Promise<LocaleMessages>;

const localeModules = import.meta.glob<LocaleModule>("./locales/*.json");

const localeEntries = Object.entries(localeModules)
  .map(([path, loadModule]) => {
    const match = path.match(/\/([a-z]{2,5}(?:-[a-z]{2,4})?)\.json$/i);
    if (!match) return null;

    const locale = match[1].toLowerCase();
    const loader: LocaleLoader = async () => (await loadModule()).default;
    return [locale, loader] as const;
  })
  .filter((entry): entry is readonly [string, LocaleLoader] => entry !== null);

export const LOCALE_LOADERS: Record<string, LocaleLoader> = Object.fromEntries(
  localeEntries,
);

export const SUPPORTED_LOCALES = Object.keys(LOCALE_LOADERS).sort();
export type SupportedLocale = string;

export const defaultLocale: SupportedLocale = SUPPORTED_LOCALES.includes("en")
  ? "en"
  : (SUPPORTED_LOCALES[0] ?? "en");

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
  const base = tag.split("-")[0];

  if (SUPPORTED_LOCALES.includes(base)) {
    return base;
  }

  return defaultLocale;
}
