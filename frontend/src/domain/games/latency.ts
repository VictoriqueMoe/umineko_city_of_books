import type { GameType } from "../../types/api";

export const LATENCY_FAIR_MS = 90;
export const LATENCY_POOR_MS = 170;
export const LATENCY_SAMPLES = 5;

export type LatencyGrade = "good" | "fair" | "poor";

export const REALTIME_GAME_TYPES = new Set<GameType>(["pong"]);

export function latencyGrade(roundTripMs: number): LatencyGrade {
    if (roundTripMs >= LATENCY_POOR_MS) {
        return "poor";
    }

    if (roundTripMs >= LATENCY_FAIR_MS) {
        return "fair";
    }

    return "good";
}

export function medianLatency(samples: readonly number[]): number | null {
    if (samples.length === 0) {
        return null;
    }

    const sorted = [...samples].sort((a, b) => a - b);
    const middle = Math.floor(sorted.length / 2);

    if (sorted.length % 2 === 1) {
        return sorted[middle];
    }

    return (sorted[middle - 1] + sorted[middle]) / 2;
}

export function keepLatestSamples(samples: readonly number[], next: number): number[] {
    const kept = [...samples, next];
    if (kept.length > LATENCY_SAMPLES) {
        kept.splice(0, kept.length - LATENCY_SAMPLES);
    }

    return kept;
}
