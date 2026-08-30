import { dispatch } from "./bus";
import { passesRealtimeGuard } from "./guards";
import { isSimulationFrame, parseServerEvent } from "./parse";
import { onFrame, onStatus } from "./socket";
import { reportClientError } from "../telemetry";

type EpochListener = () => void;

const epochListeners = new Set<EpochListener>();

let epoch = 0;
let installed = false;

const reportedUnknownTypes = new Set<string>();

function reportUnknownEventOnce(raw: string): void {
    const type = readUnknownType(raw);
    if (type === null || reportedUnknownTypes.has(type)) {
        return;
    }

    reportedUnknownTypes.add(type);
    reportClientError(new Error(`unknown realtime event "${type}"`), { source: "realtime-contract" });
}

function readUnknownType(raw: string): string | null {
    try {
        const value: unknown = JSON.parse(raw);
        if (value === null || typeof value !== "object") {
            return null;
        }

        const { type } = value as { type?: unknown };

        return typeof type === "string" ? type : null;
    } catch {
        return null;
    }
}

function receiveFrame(raw: string): void {
    const parsed = parseServerEvent(raw);

    if (!parsed.ok) {
        if (parsed.reason === "unknown-type") {
            reportUnknownEventOnce(raw);
        }

        return;
    }

    if (isSimulationFrame(parsed.event.type)) {
        dispatch(parsed.event);
        return;
    }

    if (!passesRealtimeGuard(parsed.event)) {
        return;
    }

    dispatch(parsed.event);
}

function bumpEpoch(): void {
    epoch += 1;

    for (const listener of Array.from(epochListeners)) {
        listener();
    }
}

export function ensureRealtimePipeline(): void {
    if (installed) {
        return;
    }

    installed = true;

    onFrame(receiveFrame);
    onStatus(bumpEpoch);
}

export function getRealtimeEpoch(): number {
    return epoch;
}

export function subscribeRealtimeEpoch(listener: EpochListener): () => void {
    ensureRealtimePipeline();

    epochListeners.add(listener);

    return () => {
        epochListeners.delete(listener);
    };
}
