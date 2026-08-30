export interface StreamResolution {
    label: string;
    pixels: number;
}

export interface BitrateRecommendation {
    low: number;
    typical: number;
    high: number;
}

export const STREAM_RESOLUTIONS: StreamResolution[] = [
    { label: "720p", pixels: 1280 * 720 },
    { label: "1080p", pixels: 1920 * 1080 },
    { label: "1440p", pixels: 2560 * 1440 },
    { label: "4K", pixels: 3840 * 2160 },
];

export const FPS_OPTIONS = [30, 60];

export const BITRATE_STORAGE_KEY = "stream.bitrateKbps";

export const MIN_BITRATE = 500;

export const MAX_BITRATE = 50000;

export const BITRATE_STEP_KBPS = 500;

export const BITS_PER_PIXEL_LOW = 0.07;

export const BITS_PER_PIXEL_TYPICAL = 0.095;

export const BITS_PER_PIXEL_HIGH = 0.12;

export const DEFAULT_RESOLUTION_INDEX = 1;

export const DEFAULT_CALCULATOR_FPS = 60;

export function parseBitrate(raw: string): number {
    return Number(raw);
}

export function isBitrateValid(raw: string): boolean {
    const value = parseBitrate(raw);

    return Number.isFinite(value) && value >= MIN_BITRATE && value <= MAX_BITRATE;
}

export function kbpsForBitsPerPixel(pixelsPerSecond: number, bitsPerPixel: number): number {
    return Math.round((pixelsPerSecond * bitsPerPixel) / 1000 / BITRATE_STEP_KBPS) * BITRATE_STEP_KBPS;
}

export function recommendedBitrate(pixels: number, fps: number): BitrateRecommendation {
    const pixelsPerSecond = pixels * fps;

    return {
        low: kbpsForBitsPerPixel(pixelsPerSecond, BITS_PER_PIXEL_LOW),
        typical: kbpsForBitsPerPixel(pixelsPerSecond, BITS_PER_PIXEL_TYPICAL),
        high: kbpsForBitsPerPixel(pixelsPerSecond, BITS_PER_PIXEL_HIGH),
    };
}

export function recommendedBitrateForResolution(resolutionIndex: number, fps: number): BitrateRecommendation {
    const resolution = STREAM_RESOLUTIONS[resolutionIndex];

    return recommendedBitrate(resolution.pixels, fps);
}
