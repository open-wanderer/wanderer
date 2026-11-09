import { beforeEach, describe, expect, it, vi } from "vitest";
import ExifReader from "exifreader";
import heic2any from "heic2any";
import { prepareWaypointDevicePhoto, prepareWaypointDevicePhotos, DEVICE_PHOTO_ACCEPT } from "./waypoint_device_photo_util";

vi.mock("exifreader", () => ({ default: { load: vi.fn() } }));
vi.mock("heic2any", () => ({ default: vi.fn() }));

beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(ExifReader.load).mockResolvedValue({
        gps: { Latitude: 47, Longitude: 8 },
        exif: { GPSDateStamp: { value: ["2026:06:25"] }, GPSTimeStamp: { value: [[10, 1], [30, 1], [0, 1]] } },
    } as never);
    vi.mocked(heic2any).mockResolvedValue(new Blob(["converted"], { type: "image/jpeg" }));
});

describe("device photos used to create waypoints", () => {
    it.each(["image/gif", "image/avif", "video/mp4", "image/svg+xml"])("rejects %s before reading GPS or uploading", async (type) => {
        expect(await prepareWaypointDevicePhoto(new File(["image"], "photo.jpg", { type }))).toBeUndefined();
        expect(ExifReader.load).not.toHaveBeenCalled();
        expect(heic2any).not.toHaveBeenCalled();
    });

    it.each(["image/heic", "image/heif", ""])("converts %s with original GPS and capture metadata retained", async (type) => {
        const original = new File(["original EXIF"], "PHOTO.HEIF", { type });
        const result = await prepareWaypointDevicePhoto(original);

        expect(result?.file.name).toBe("PHOTO.jpg");
        expect(result?.file.type).toBe("image/jpeg");
        expect(await result?.file.text()).toBe("converted");
        expect(result?.coordinates).toEqual({ lat: 47, lon: 8, time: new Date("2026-06-25T10:30:00Z") });
        expect(result?.metadata).toMatchObject({ coordinates: { lat: 47, lon: 8 } });
        expect(heic2any).toHaveBeenCalledWith({ blob: original, toType: "image/jpeg" });
        expect(ExifReader.load).toHaveBeenCalledTimes(1);
        expect(vi.mocked(ExifReader.load).mock.invocationCallOrder[0]).toBeLessThan(vi.mocked(heic2any).mock.invocationCallOrder[0]);
    });
});

it("continues the actual device batch after a named HEIC conversion failure", async () => {
    vi.mocked(heic2any).mockRejectedValueOnce(new Error("libheif decode failed"));
    const files = [new File(["broken"], "broken.heic", { type: "image/heic" }), new File(["valid"], "valid.jpg", { type: "image/jpeg" })];
    const failure = vi.fn();
    const result = await prepareWaypointDevicePhotos(files, failure);
    expect(failure).toHaveBeenCalledWith(files[0], "conversion");
    expect(result).toHaveLength(1);
    expect(result[0]).toMatchObject({ index: 1, file: files[1], coordinates: { lat: 47, lon: 8 } });
});

it("offers only device images and reports each file before conversion", async () => {
    expect(DEVICE_PHOTO_ACCEPT).not.toMatch(/video|svg/);
    const photo = new File(["photo"], "photo.heic", { type: "image/heic" });
    const progress = vi.fn();
    await prepareWaypointDevicePhotos([photo], vi.fn(), progress);
    expect(progress).toHaveBeenCalledWith(photo, 1, 1);
    expect(progress.mock.invocationCallOrder[0]).toBeLessThan(vi.mocked(heic2any).mock.invocationCallOrder[0]);
});

it("keeps supported image content when its declared type asks for HEIC conversion", async () => {
    vi.mocked(ExifReader.load).mockResolvedValue({ gps: { Latitude: 47, Longitude: 8 }, file: { FileType: { value: "jpeg" } } } as never);
    const original = new File(["JPEG content"], "photo.jpeg", { type: "image/heic" });
    const result = await prepareWaypointDevicePhoto(original);
    expect(result?.file.type).toBe("image/jpeg");
    expect(result?.coordinates).toMatchObject({ lat: 47, lon: 8 });
    expect(heic2any).not.toHaveBeenCalled();
});
