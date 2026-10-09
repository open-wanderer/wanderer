function removeEmpty(obj: Record<string, any>) {
  Object.entries(obj).forEach(([key, val]) => {
    if (val && val instanceof Object) {
      removeEmpty(val);
    } else if (val == null) {
      delete obj[key];
    }
  });
}

function allDatesToISOString(obj: Record<string, any>) {
  Object.entries(obj).forEach(([key, val]) => {
    if (val) {
      if (val instanceof Date) {
        obj[key] = val.toISOString().split('.')[0] + 'Z';
      } else if (val instanceof Object) {
        allDatesToISOString(val);
      }
    }
  });
}

function haversineDistance(lat1: number, lon1: number, lat2: number, lon2: number): number {
  const R = 6371; // Radius of the Earth in km
  const dLat = (lat2 - lat1) * (Math.PI / 180); // Convert degrees to radians
  const dLon = (lon2 - lon1) * (Math.PI / 180);
  const a =
    Math.sin(dLat / 2) * Math.sin(dLat / 2) +
    Math.cos((lat1 * (Math.PI / 180))) * Math.cos((lat2 * (Math.PI / 180))) *
    Math.sin(dLon / 2) * Math.sin(dLon / 2);
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
  const distance = R * c * 1000; // Distance in km
  return distance;
}

function convertDMSToDD(dms: Number[], direction: "N" | "O" | "S" | "W") {
  var dd = dms[0].valueOf() + dms[1].valueOf() / 60 + dms[2].valueOf() / (60 * 60);

  if (direction == "S" || direction == "W") {
    dd = dd * -1;
  }
  return dd;
}

export { removeEmpty, allDatesToISOString, haversineDistance, convertDMSToDD };
