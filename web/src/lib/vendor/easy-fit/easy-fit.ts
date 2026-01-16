import { calculateCRC, getArrayBuffer, readRecord } from "./binary";

type Options = {
  force: boolean,
  speedUnit: 'm/s' | 'km/h' | 'mph',
  lengthUnit: 'm' | 'km' | 'mi',
  temperatureUnit: 'celsius' | 'fahrenheit',
  elapsedRecordField: boolean,
  mode: 'cascade' | 'list' | 'both'
}

class EasyFit {
  options: Options;
  constructor(
    options?: Options
  ) {
    this.options = options || {
      force: true,
      speedUnit: 'm/s',
      lengthUnit: 'm',
      temperatureUnit: 'celsius',
      elapsedRecordField: false,
      mode: 'list'
    };
  }

  parse(content: ArrayBuffer, callback: { (error: string, data: any): void; (arg0: string | null, arg1: {}): void; }) {
    var blob = new Uint8Array(getArrayBuffer(content));

    if (blob.length < 12) {
      callback('File to small to be a FIT file', {});
      if (!this.options.force) {
        return;
      }
    }

    var headerLength = blob[0];
    if (headerLength !== 14 && headerLength !== 12) {
      callback('Incorrect header size', {});
      if (!this.options.force) {
        return;
      }
    }

    var fileTypeString = '';
    for (var i = 8; i < 12; i++) {
      fileTypeString += String.fromCharCode(blob[i]);
    }
    if (fileTypeString !== '.FIT') {
      callback('Missing \'.FIT\' in header', {});
      if (!this.options.force) {
        return;
      }
    }

    if (headerLength === 14) {
      var crcHeader = blob[12] + (blob[13] << 8);
      var crcHeaderCalc = calculateCRC(blob, 0, 12);
      if (crcHeader !== crcHeaderCalc) {
        // callback('Header CRC mismatch', {});
        // TODO: fix Header CRC check
        if (!this.options.force) {
          return;
        }
      }
    }
    var dataLength = blob[4] + (blob[5] << 8) + (blob[6] << 16) + (blob[7] << 24);
    var crcStart = dataLength + headerLength;
    var crcFile = blob[crcStart] + (blob[crcStart + 1] << 8);
    var crcFileCalc = calculateCRC(blob, headerLength === 12 ? 0 : headerLength, crcStart);

    if (crcFile !== crcFileCalc) {
      // callback('File CRC mismatch', {});
      // TODO: fix File CRC check
      if (!this.options.force) {
        return;
      }
    }

    var fitObj: Record<string, any> = {};
    var sessions = [];
    var laps = [];
    var records = [];
    var events = [];

    var tempLaps = [];
    var tempRecords = [];

    var loopIndex = headerLength;
    var messageTypes: Record<number, any> = [];

    var isModeCascade = this.options.mode === 'cascade';
    var isCascadeNeeded = isModeCascade || this.options.mode === 'both';

    var startDate = void 0;

    while (loopIndex < crcStart) {
      var _readRecord = readRecord(blob, messageTypes, loopIndex, this.options, startDate),
        nextIndex = _readRecord.nextIndex,
        messageType = _readRecord.messageType,
        message = "message" in _readRecord ? _readRecord.message : undefined;

      loopIndex = nextIndex;
      switch (messageType) {
        case 'lap':
          if (isCascadeNeeded) {
            message.records = tempRecords;
            tempRecords = [];
            tempLaps.push(message);
          }
          laps.push(message);
          break;
        case 'session':
          if (isCascadeNeeded) {
            message.laps = tempLaps;
            tempLaps = [];
          }
          sessions.push(message);
          break;
        case 'event':
          events.push(message);
          break;
        case 'record':
          if (!startDate) {
            startDate = message.timestamp;
            message.elapsed_time = 0;
          }
          records.push(message);
          if (isCascadeNeeded) {
            tempRecords.push(message);
          }
          break;
        default:
          if (messageType !== '') {
            fitObj[messageType] = message;
          }
          break;
      }
    }

    if (isCascadeNeeded) {
      fitObj.activity.sessions = sessions;
      fitObj.activity.events = events;
    }
    if (!isModeCascade) {
      fitObj.sessions = sessions;
      fitObj.laps = laps;
      fitObj.records = records;
      fitObj.events = events;
    }

    callback(null, fitObj);
  }
}

export default EasyFit