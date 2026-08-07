import type { Actor } from "./activitypub/actor";
import type { Trail } from "./trail";

class SummitLog {
  id?: string;
  date: string;
  text?: string;
  gpx?: string;
  _gpx?: File | Blob | null;
  photos: string[];
  _photos?: File[];
  distance?: number
  elevation_gain?: number
  elevation_loss?: number
  duration?: number
  author: string;
  trail?: string;
  iri?: string;
  created?: string;

  expand?: {
    gpx_data?: string;
    trail?: Trail;
    author?: Actor
  }

  constructor(date: string, params?: { id?: string, text?: string, distance?: number, elevation_loss?: number, elevation_gain?: number, duration?: number, photos?: string[] }) {
    this.date = date;
    this.id = params?.id;
    this.text = params?.text ?? "";
    this.expand = {}
    this.distance = params?.distance
    this.elevation_gain = params?.elevation_gain
    this.elevation_loss = params?.elevation_loss
    this.duration = params?.duration
    this._gpx = null
    this.photos = params?.photos ?? []
    this._photos = [];
    this.author = "000000000000000"
  }
}

interface SummitLogFilter {
  category: string[],
  startDate?: string;
  endDate?: string;
  trail?: string;
}

export { SummitLog, type SummitLogFilter };
