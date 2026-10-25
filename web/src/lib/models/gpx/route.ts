import Waypoint from './waypoint';
import Link from './link';

export default class Route {
  name: string;
  cmt: string;
  desc: string;
  src: string;
  number: number;
  type: string;
  extensions: string;
  link?: Link[];
  rtept?: Waypoint[];
  constructor(object: {
    name: string,
    cmt: string, desc: string,
    src: string,
    number: number,
    type: string,
    extensions: string,
    link?: Link | Link[],
    rtept?: Waypoint | Waypoint[]
  }) {
    this.name = object.name;
    this.cmt = object.cmt;
    this.desc = object.desc;
    this.src = object.src;
    this.number = object.number;
    this.type = object.type;
    this.extensions = object.extensions;
    if (object.link) {
      if (!Array.isArray(object.link)) {
        object.link = [object.link];
      }
      this.link = object.link.map(l => new Link(l));
    }

    if (object.rtept) {
      if (!Array.isArray(object.rtept)) {
        object.rtept = [object.rtept];
      }
      this.rtept = (object.rtept as Waypoint[]).map(rtept => new Waypoint(rtept))
    }
  }

  toGeoJSON(): GeoJSON.Feature {
    const coordinates = (this.rtept || []).map(pt =>
      [pt.$.lon ?? 0, pt.$.lat ?? 0, pt.ele ?? 0]
    );

    return {
      type: "Feature",
      geometry: {
        type: "LineString",
        coordinates
      },
      properties: {
        name: this.name,
        desc: this.desc,
        type: this.type,
        number: this.number
      }
    };
  }
}
