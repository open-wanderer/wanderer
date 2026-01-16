import { FIT } from "./fit";
import { getFitMessage, getFitMessageBaseType } from "./messages";

export { addEndian, readRecord, getArrayBuffer, calculateCRC };


function addEndian(littleEndian: boolean, bytes: any[]) {
    var result = 0;
    if (!littleEndian) bytes.reverse();
    for (var i = 0; i < bytes.length; i++) {
        result += bytes[i] << (i << 3) >>> 0;
    }

    return result;
}

function readData(blob: Uint8Array, fDef: { endianAbility: boolean; size: number; littleEndian: boolean; type: string; }, startIndex: number) {
    if (fDef.endianAbility === true) {
        var temp = [];
        for (var i = 0; i < fDef.size; i++) {
            temp.push(blob[startIndex + i]);
        }
        var uint32Rep = addEndian(fDef.littleEndian, temp);

        if (fDef.type === 'sint32') {
            return uint32Rep >> 0;
        }

        return uint32Rep;
    }
    return blob[startIndex];
}

function formatByType(data: number, type: string | number, scale: number, offset: number) {
    switch (type) {
        case 'date_time':
            return new Date(data * 1000 + 631065600000);
        case 'sint32':
        case 'sint16':
            return data * FIT.scConst;
        case 'uint32':
        case 'uint16':
            return scale ? data / scale + offset : data;
        default: {
            const types = FIT.types as Record<string, Record<string, any>>;
            if (types[type]) {
                return types[type][String(data)];
            }
            return data;
        }
    }
}

function isInvalidValue(data: number, type: string) {
    switch (type) {
        case 'enum':
            return data === 0xFF;
        case 'sint8':
            return data === 0x7F;
        case 'uint8':
            return data === 0xFF;
        case 'sint16':
            return data === 0x7FFF;
        case 'unit16':
            return data === 0xFFFF;
        case 'sint32':
            return data === 0x7FFFFFFF;
        case 'uint32':
            return data === 0xFFFFFFFF;
        case 'string':
            return data === 0x00;
        case 'float32':
            return data === 0xFFFFFFFF;
        case 'float64':
            return data === 0xFFFFFFFFFFFFFFFF;
        case 'uint8z':
            return data === 0x00;
        case 'uint16z':
            return data === 0x0000;
        case 'uint32z':
            return data === 0x000000;
        case 'byte':
            return data === 0xFF;
        case 'sint64':
            return data === 0x7FFFFFFFFFFFFFFF;
        case 'uint64':
            return data === 0xFFFFFFFFFFFFFFFF;
        case 'uint64z':
            return data === 0x0000000000000000;
        default:
            return false;
    }
}

function convertTo(data: number, unitsList: string, speedUnit: string | number) {
    const options = FIT.options as Record<string, Record<string, { multiplier: number; offset: number }>>;
    var unitObj = options[unitsList]?.[String(speedUnit)];
    return unitObj ? data * unitObj.multiplier + unitObj.offset : data;
}

function applyOptions(data: number, field: any, options: { speedUnit?: string | number; lengthUnit?: string | number; temperatureUnit?: string | number; }) {
    switch (field) {
        case 'speed':
        case 'enhanced_speed':
        case 'vertical_speed':
        case 'avg_speed':
        case 'max_speed':
        case 'speed_1s':
        case 'ball_speed':
        case 'enhanced_avg_speed':
        case 'enhanced_max_speed':
        case 'avg_pos_vertical_speed':
        case 'max_pos_vertical_speed':
        case 'avg_neg_vertical_speed':
        case 'max_neg_vertical_speed':
            return convertTo(data, 'speedUnits', options.speedUnit ?? 0);
        case 'distance':
        case 'total_distance':
        case 'enhanced_avg_altitude':
        case 'enhanced_min_altitude':
        case 'enhanced_max_altitude':
        case 'enhanced_altitude':
        case 'height':
        case 'odometer':
        case 'avg_stroke_distance':
        case 'min_altitude':
        case 'avg_altitude':
        case 'max_altitude':
        case 'total_ascent':
        case 'total_descent':
        case 'altitude':
        case 'cycle_length':
        case 'auto_wheelsize':
        case 'custom_wheelsize':
        case 'gps_accuracy':
            return convertTo(data, 'lengthUnits', options.lengthUnit ?? 0);
        case 'temperature':
        case 'avg_temperature':
        case 'max_temperature':
            return convertTo(data, 'temperatureUnits', options.temperatureUnit ?? 0);
        default:
            return data;
    }
}

function readRecord(blob: Uint8Array, messageTypes: Record<number, any>, startIndex: number, options: { elapsedRecordField?: any; speedUnit?: string | number; lengthUnit?: string | number; temperatureUnit?: string | number; }, startDate: number | undefined) {
    var recordHeader = blob[startIndex];
    var localMessageType = recordHeader & 15;

    if ((recordHeader & 64) === 64) {
        // is definition message
        // startIndex + 1 is reserved

        var lEnd = blob[startIndex + 2] === 0;
        var mTypeDef: { littleEndian: boolean, globalMessageNumber: number, numberOfFields: number, fieldDefs: any[] } = {
            littleEndian: lEnd,
            globalMessageNumber: addEndian(lEnd, [blob[startIndex + 3], blob[startIndex + 4]]),
            numberOfFields: blob[startIndex + 5],
            fieldDefs: []
        };

        var _message = getFitMessage(mTypeDef.globalMessageNumber);

        for (var i = 0; i < mTypeDef.numberOfFields; i++) {
            var fDefIndex = startIndex + 6 + i * 3;
            var baseType = blob[fDefIndex + 2];

            var _message$getAttribute = _message.getAttributes(blob[fDefIndex]),
                field = _message$getAttribute.field,
                type = _message$getAttribute.type;

            var fDef = {
                type: type,
                fDefNo: blob[fDefIndex],
                size: blob[fDefIndex + 1],
                endianAbility: (baseType & 128) === 128,
                littleEndian: lEnd,
                baseTypeNo: baseType & 15,
                name: field,
                dataType: getFitMessageBaseType(baseType & 15)
            };

            mTypeDef.fieldDefs.push(fDef);
        }
        messageTypes[localMessageType] = mTypeDef;

        return {
            messageType: 'fieldDescription',
            nextIndex: startIndex + 6 + mTypeDef.numberOfFields * 3
        };
    }

    var messageType: { littleEndian: boolean, globalMessageNumber: number, numberOfFields: number, fieldDefs: any[] } | undefined = void 0;

    if (messageTypes[localMessageType]) {
        messageType = messageTypes[localMessageType];
    } else {
        messageType = messageTypes[0];
    }

    // TODO: handle compressed header ((recordHeader & 128) == 128)

    // uncompressed header
    var messageSize = 0;
    var readDataFromIndex = startIndex + 1;
    var fields: any = {};
    var message = getFitMessage(messageType?.globalMessageNumber);

    for (var _i = 0; _i < (messageType?.fieldDefs.length ?? 0); _i++) {
        var _fDef = messageType?.fieldDefs[_i];
        var data = readData(blob, _fDef, readDataFromIndex);

        if (!isInvalidValue(data, _fDef.type)) {
            var _message$getAttribute2 = message.getAttributes(_fDef.fDefNo),
                field = _message$getAttribute2.field,
                type = _message$getAttribute2.type,
                scale = _message$getAttribute2.scale,
                offset = _message$getAttribute2.offset;

            if (field !== 'unknown' && field !== '' && field !== undefined) {
                fields[field] = applyOptions(formatByType(data, type, scale, offset), field, options);
            }

            if (message.name === 'record' && options.elapsedRecordField) {
                fields.elapsed_time = (fields.timestamp - (startDate ?? 0)) / 1000;
            }
        }

        readDataFromIndex += _fDef.size;
        messageSize += _fDef.size;
    }

    var result = {
        messageType: message.name,
        nextIndex: startIndex + messageSize + 1,
        message: fields
    };

    return result;
}

function getArrayBuffer(buffer: ArrayBuffer | number[]) {
    if (buffer instanceof ArrayBuffer) {
        return buffer;
    }
    var ab = new ArrayBuffer(buffer.length);
    var view = new Uint8Array(ab);
    for (var i = 0; i < buffer.length; ++i) {
        view[i] = buffer[i];
    }
    return ab;
}

function calculateCRC(blob: Uint8Array, start: number, end: number) {
    var crcTable = [0x0000, 0xCC01, 0xD801, 0x1400, 0xF001, 0x3C00, 0x2800, 0xE401, 0xA001, 0x6C00, 0x7800, 0xB401, 0x5000, 0x9C01, 0x8801, 0x4400];

    var crc = 0;
    for (var i = start; i < end; i++) {
        var byte = blob[i];
        var tmp = crcTable[crc & 0xF];
        crc = crc >> 4 & 0x0FFF;
        crc = crc ^ tmp ^ crcTable[byte & 0xF];
        tmp = crcTable[crc & 0xF];
        crc = crc >> 4 & 0x0FFF;
        crc = crc ^ tmp ^ crcTable[byte >> 4 & 0xF];
    }

    return crc;
}