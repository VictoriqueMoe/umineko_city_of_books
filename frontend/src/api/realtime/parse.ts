import { absolutizeMedia } from "../client";
import { REALTIME_EVENTS, type RealtimeEvent, type RealtimeEventName } from "./events";

export const FRAME_TYPE_SUFFIX = "_frame";

export type RealtimeParseFailure = "invalid-json" | "not-an-object" | "missing-type" | "unknown-type";

export type ParseServerEventResult = { ok: true; event: RealtimeEvent } | { ok: false; reason: RealtimeParseFailure };

type ParseFailure = { ok: false; reason: RealtimeParseFailure };

type DecodedJson = { ok: true; value: unknown };

type Envelope = { ok: true; type: RealtimeEventName; data: unknown };

const KNOWN_EVENT_NAMES: ReadonlySet<string> = new Set<string>(Object.values(REALTIME_EVENTS));

export function isRealtimeEventName(type: string): type is RealtimeEventName {
    return KNOWN_EVENT_NAMES.has(type);
}

export function isSimulationFrame(type: string): boolean {
    return type.endsWith(FRAME_TYPE_SUFFIX);
}

function decodeJson(raw: string): DecodedJson | ParseFailure {
    try {
        return { ok: true, value: JSON.parse(raw) as unknown };
    } catch {
        return { ok: false, reason: "invalid-json" };
    }
}

function readEnvelope(value: unknown): Envelope | ParseFailure {
    if (value === null || typeof value !== "object" || Array.isArray(value)) {
        return { ok: false, reason: "not-an-object" };
    }

    const { type, data } = value as { type?: unknown; data?: unknown };

    if (typeof type !== "string") {
        return { ok: false, reason: "missing-type" };
    }

    if (!isRealtimeEventName(type)) {
        return { ok: false, reason: "unknown-type" };
    }

    return { ok: true, type, data };
}

function asRealtimeEvent(type: RealtimeEventName, data: unknown): RealtimeEvent {
    return { type, data } as RealtimeEvent;
}

function normaliseNonFrameEvent(event: RealtimeEvent): RealtimeEvent {
    return absolutizeMedia(event);
}

export function parseServerEvent(raw: string): ParseServerEventResult {
    const decoded = decodeJson(raw);
    if (!decoded.ok) {
        return decoded;
    }

    const envelope = readEnvelope(decoded.value);
    if (!envelope.ok) {
        return envelope;
    }

    const event = asRealtimeEvent(envelope.type, envelope.data);

    if (isSimulationFrame(envelope.type)) {
        return { ok: true, event };
    }

    return { ok: true, event: normaliseNonFrameEvent(event) };
}
