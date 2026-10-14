import { dispatch } from "./bus";
import { passesRealtimeGuard } from "./guards";
import { decodeServerFrame, isSimulationFrame, parseServerValue } from "./parse";
import { onFrame, onStatus } from "./socket";
import { reportClientError } from "../telemetry";

type EpochListener = () => void;

const epochListeners = new Set<EpochListener>();

let epoch = 0;
let installed = false;

const reportedUnknownTypes = new Set<string>();

function reportUnknownEventOnce(value: unknown): void {
    const type = readUnknownType(value);
    if (type === null || reportedUnknownTypes.has(type)) {
        return;
    }

    reportedUnknownTypes.add(type);
    reportClientError(new Error(`unknown realtime event "${type}"`), { source: "realtime-contract" });
}

function readUnknownType(value: unknown): string | null {
    if (value === null || typeof value !== "object") {
        return null;
    }

    const { type } = value as { type?: unknown };

    return typeof type === "string" ? type : null;
}

function receiveFrame(raw: string): void {
    const decoded = decodeServerFrame(raw);
    if (!decoded.ok) {
        return;
    }

    for (const value of decoded.values) {
        receiveMessage(value);
    }
}

function receiveMessage(value: unknown): void {
    const parsed = parseServerValue(value);

    if (!parsed.ok) {
        if (parsed.reason === "unknown-type") {
            reportUnknownEventOnce(value);
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
