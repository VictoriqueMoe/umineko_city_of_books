import { act, renderHook, waitFor } from "@testing-library/react";
import type { Room } from "livekit-client";
import { beforeEach, describe, expect, it, vi } from "vitest";
import * as livekit from "../api/livekit/connect";
import type { ConnectRoomOptions } from "../api/livekit/connect";
import * as telemetry from "../api/telemetry";
import { ROOM_CONNECT_FAILED, useLiveKitRoom, type UseLiveKitRoomOptions } from "./useLiveKitRoom";

vi.mock("../api/livekit/connect", () => ({
    connectRoom: vi.fn(),
    disconnectRoom: vi.fn(),
}));

vi.mock("../api/telemetry", () => ({
    reportClientError: vi.fn(),
}));

const connectRoom = vi.mocked(livekit.connectRoom);
const disconnectRoom = vi.mocked(livekit.disconnectRoom);
const reportClientError = vi.mocked(telemetry.reportClientError);

function makeRoom(name: string): Room {
    return { name } as unknown as Room;
}

function lastOptions(): ConnectRoomOptions {
    const { calls } = connectRoom.mock;
    if (calls.length === 0) {
        throw new Error("connectRoom was never called");
    }

    return calls[calls.length - 1][0];
}

let requestToken: ReturnType<typeof vi.fn>;

function mount(overrides: Partial<UseLiveKitRoomOptions> = {}) {
    const initialProps: UseLiveKitRoomOptions = {
        streamId: "stream-1",
        isLive: true,
        wantsRoom: true,
        wantsMedia: true,
        requestToken: requestToken as unknown as UseLiveKitRoomOptions["requestToken"],
        ...overrides,
    };

    return renderHook((props: UseLiveKitRoomOptions) => useLiveKitRoom(props), { initialProps });
}

beforeEach(() => {
    connectRoom.mockReset();
    disconnectRoom.mockReset();
    reportClientError.mockReset();
    connectRoom.mockResolvedValue(makeRoom("room-1"));
    requestToken = vi.fn().mockResolvedValue({ token: "tok", url: "wss://livekit.test" });
});

describe("useLiveKitRoom", () => {
    it("builds no room while the stream is offline", async () => {
        // given
        mount({ isLive: false });

        // when
        await act(async () => {});

        // then
        expect(requestToken).not.toHaveBeenCalled();
        expect(connectRoom).not.toHaveBeenCalled();
    });

    it("builds no room while nothing on the page wants one", async () => {
        // given
        mount({ wantsRoom: false });

        // when
        await act(async () => {});

        // then
        expect(connectRoom).not.toHaveBeenCalled();
    });

    it("builds no room without a stream to watch", async () => {
        // given
        mount({ streamId: undefined });

        // when
        await act(async () => {});

        // then
        expect(connectRoom).not.toHaveBeenCalled();
    });

    it("asks for a token for the stream it is watching", async () => {
        // given
        mount();

        // when
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalled();
        });

        // then
        expect(requestToken).toHaveBeenCalledWith("stream-1");
        expect(lastOptions().url).toBe("wss://livekit.test");
        expect(lastOptions().token).toBe("tok");
    });

    it("subscribes to the media when the viewer is watching the video", async () => {
        // given
        mount({ wantsMedia: true });

        // when
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalled();
        });

        // then
        expect(lastOptions().autoSubscribe).toBe(true);
    });

    it("joins without subscribing when the page wants the room but not the video", async () => {
        // given
        mount({ wantsMedia: false });

        // when
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalled();
        });

        // then
        expect(lastOptions().autoSubscribe).toBe(false);
    });

    it("hands the room over once it has connected", async () => {
        // given
        const room = makeRoom("room-1");
        const view = mount();
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalled();
        });

        // when
        act(() => {
            lastOptions().on?.onConnected?.(room);
        });

        // then
        expect(view.result.current.room).toBe(room);
    });

    it("drops the room again when it disconnects", async () => {
        // given
        const room = makeRoom("room-1");
        const view = mount();
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalled();
        });
        act(() => {
            lastOptions().on?.onConnected?.(room);
        });

        // when
        act(() => {
            lastOptions().on?.onDisconnected?.(room);
        });

        // then
        expect(view.result.current.room).toBeNull();
    });

    it("keeps the room it is showing when an older room disconnects", async () => {
        // given
        const room = makeRoom("room-1");
        const stale = makeRoom("room-0");
        const view = mount();
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalled();
        });
        act(() => {
            lastOptions().on?.onConnected?.(room);
        });

        // when
        act(() => {
            lastOptions().on?.onDisconnected?.(stale);
        });

        // then
        expect(view.result.current.room).toBe(room);
    });

    it("tears the room down and builds another when the wanted media changes", async () => {
        // given
        const room = makeRoom("room-1");
        const view = mount({ wantsMedia: false });
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalledTimes(1);
        });
        act(() => {
            lastOptions().on?.onConnected?.(room);
        });

        // when
        view.rerender({
            streamId: "stream-1",
            isLive: true,
            wantsRoom: true,
            wantsMedia: true,
            requestToken: requestToken as unknown as UseLiveKitRoomOptions["requestToken"],
        });

        // then
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalledTimes(2);
        });
        expect(disconnectRoom).toHaveBeenCalledWith(room);
        expect(lastOptions().autoSubscribe).toBe(true);
    });

    it("tears the room down and builds another when the address moves to a different stream", async () => {
        // given
        const view = mount();
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalledTimes(1);
        });

        // when
        view.rerender({
            streamId: "stream-2",
            isLive: true,
            wantsRoom: true,
            wantsMedia: true,
            requestToken: requestToken as unknown as UseLiveKitRoomOptions["requestToken"],
        });

        // then
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalledTimes(2);
        });
        expect(requestToken).toHaveBeenLastCalledWith("stream-2");
    });

    it("disconnects the room it joined when the viewer leaves the page", async () => {
        // given
        const room = makeRoom("room-1");
        const view = mount();
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalled();
        });
        act(() => {
            lastOptions().on?.onConnected?.(room);
        });

        // when
        view.unmount();

        // then
        expect(disconnectRoom).toHaveBeenCalledWith(room);
    });

    it("tells the connection it has been abandoned when the viewer leaves before it is up", async () => {
        // given
        const view = mount();
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalled();
        });
        const options = lastOptions();
        expect(options.signal?.aborted).toBe(false);

        // when
        view.unmount();

        // then
        expect(options.signal?.aborted).toBe(true);
    });

    it("never adopts a room that connects after the viewer has left", async () => {
        // given
        const room = makeRoom("room-1");
        const view = mount();
        await waitFor(() => {
            expect(connectRoom).toHaveBeenCalled();
        });
        const options = lastOptions();

        // when
        view.unmount();
        act(() => {
            options.on?.onConnected?.(room);
        });

        // then
        expect(view.result.current.room).toBeNull();
    });

    it("admits it could not reach the stream", async () => {
        // given
        requestToken.mockRejectedValue(new Error("no token for you"));

        // when
        const view = mount();

        // then
        await waitFor(() => {
            expect(view.result.current.error).toBe(ROOM_CONNECT_FAILED);
        });
    });

    it("reports the real reason it could not connect", async () => {
        // given
        const cause = new Error("no token for you");
        requestToken.mockRejectedValue(cause);

        // when
        mount();

        // then
        await waitFor(() => {
            expect(reportClientError).toHaveBeenCalledWith(cause, { source: "caught" });
        });
    });

    it("stays quiet about a failure the viewer has already walked away from", async () => {
        // given
        let reject: (reason: unknown) => void = () => {};
        requestToken.mockReturnValue(
            new Promise((_resolve, rejectToken) => {
                reject = rejectToken;
            }),
        );
        const view = mount();
        await act(async () => {});

        // when
        view.unmount();
        await act(async () => {
            reject(new Error("too late"));
        });

        // then
        expect(view.result.current.error).toBeNull();
    });
});
