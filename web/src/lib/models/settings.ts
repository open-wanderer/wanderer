import type { NotificationType } from "./notification";

export enum Language {
  cs = "cs",
  en = "en",
  de = "de",
  es = "es",
  eu = "eu",
  fr = "fr",
  hu = "hu",
  it = "it",
  nl = "nl",
  no = "no",
  pl = "pl",
  pt = "pt",
  ru = "ru",
  zh = "zh"
}

class Settings {
  id?: string | null;
  unit?: "metric" | "imperial";
  language?: Language;
  bio?: string | null;
  mapFocus?: "trails" | "location";
  location?: { name: string, lat: number, lon: number } | null;
  category?: string;
  tilesets?: ({ name: string, url: string }[]) | null
  terrain?: { terrain?: string, hillshading?: string } | null;
  user?: string;
  privacy?: { account: "public" | "private", trails: "public" | "private", lists: "public" | "private" } | null
  notifications?: Record<NotificationType, { web: boolean, email: boolean }> | null

  constructor(
    unit: "metric" | "imperial",
    language: Language,
    mapFocus: "trails" | "location",
    user: string,
    params?: {
      location?: { name: string, lat: number, lon: number }
      category?: string
      tilesets?: { name: string, url: string }[]
      terrain?: { terrain: string, hillshading: string };
    }
  ) {
    this.unit = unit;
    this.language = language;
    this.mapFocus = mapFocus;
    this.user = user;
    this.location = params?.location;
    this.category = params?.category;
    this.tilesets = params?.tilesets ?? [];
    this.terrain = params?.terrain;
  }
}


export { Settings };
