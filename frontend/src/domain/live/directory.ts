import type { LiveStream, LiveStreamListResponse } from "../../types/api";

export function applyStreamPatch(
    list: LiveStreamListResponse | undefined,
    streamId: string,
    patch: (stream: LiveStream) => LiveStream,
): LiveStreamListResponse | undefined {
    if (!list) {
        return list;
    }

    return {
        ...list,
        streams: list.streams.map(stream => (stream.id === streamId ? patch(stream) : stream)),
    };
}

export function applyViewerCount(
    list: LiveStreamListResponse | undefined,
    streamId: string,
    viewerCount: number,
): LiveStreamListResponse | undefined {
    return applyStreamPatch(list, streamId, stream => ({ ...stream, viewerCount }));
}

export function applyTitle(
    list: LiveStreamListResponse | undefined,
    streamId: string,
    title: string,
): LiveStreamListResponse | undefined {
    return applyStreamPatch(list, streamId, stream => ({ ...stream, title }));
}
