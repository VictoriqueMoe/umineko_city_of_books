import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { OverlayConnection } from "../types/api";
import { useStreamOverlay } from "./useStreamOverlay";

const mocks = vi.hoisted(() => ({
    useOverlayConnection: vi.fn(),
    fetchConnector: vi.fn(),
    resetOverlayToken: vi.fn(),
    testOverlay: vi.fn(),
    downloadBlob: vi.fn(),
}));

vi.mock("./queries/overlay", () => ({
    useOverlayConnection: mocks.useOverlayConnection,
}));

vi.mock("./mutations/overlay", () => ({
    OVERLAY_CONNECTOR_FILENAME: "overlay-connector.sef",
    useOverlayConnectorFile: () => ({ mutateAsync: mocks.fetchConnector, isPending: false }),
    useResetOverlayToken: () => ({ mutateAsync: mocks.resetOverlayToken, isPending: false }),
    useTestOverlay: () => ({ mutateAsync: mocks.testOverlay, isPending: false }),
}));

vi.mock("../utils/download", () => ({
    downloadBlob: mocks.downloadBlob,
}));

function makeConnection(overrides: Partial<OverlayConnection> = {}): OverlayConnection {
    return {
        token: "sammi-token",
        connect_url: "wss://overlay.example",
        connected: false,
        ...overrides,
    };
}

beforeEach(() => {
    mocks.useOverlayConnection.mockReturnValue({ connection: makeConnection(), loading: false, error: "" });
    mocks.fetchConnector.mockResolvedValue(new Blob(["sef-file-body"], { type: "text/plain" }));
    mocks.resetOverlayToken.mockResolvedValue(makeConnection({ token: "rotated-token" }));
    mocks.testOverlay.mockResolvedValue({ ok: true });
});

afterEach(() => {
    vi.restoreAllMocks();
});

describe("useStreamOverlay", () => {
    it("passes the load failure straight through instead of apologising for it", () => {
        // given
        mocks.useOverlayConnection.mockReturnValue({
            connection: null,
            loading: false,
            error: "Your streaming permission was revoked.",
        });

        // when
        const { result } = renderHook(() => useStreamOverlay());

        // then
        expect(result.current.loadError).toBe("Your streaming permission was revoked.");
        expect(result.current.connection).toBeNull();
    });

    it("reports loading separately from failing", () => {
        // given
        mocks.useOverlayConnection.mockReturnValue({ connection: null, loading: true, error: "" });

        // when
        const { result } = renderHook(() => useStreamOverlay());

        // then
        expect(result.current.loading).toBe(true);
        expect(result.current.loadError).toBe("");
    });

    it("hands the connector to the download util under the SAMMI filename", async () => {
        // given
        const blob = new Blob(["sef-file-body"], { type: "text/plain" });
        mocks.fetchConnector.mockResolvedValue(blob);
        const { result } = renderHook(() => useStreamOverlay());

        // when
        act(() => result.current.downloadConnector());

        // then
        await waitFor(() => {
            expect(mocks.downloadBlob).toHaveBeenCalledWith(blob, "overlay-connector.sef");
        });
    });

    it("surfaces the reason the connector could not be fetched", async () => {
        // given
        mocks.fetchConnector.mockRejectedValue(new Error("The connector is unavailable."));
        const { result } = renderHook(() => useStreamOverlay());

        // when
        act(() => result.current.downloadConnector());

        // then
        await waitFor(() => {
            expect(result.current.actionError).toBe("The connector is unavailable.");
        });
        expect(mocks.downloadBlob).not.toHaveBeenCalled();
    });

    it("names the action when the failure carries no message", async () => {
        // given
        mocks.fetchConnector.mockRejectedValue(new Error(""));
        const { result } = renderHook(() => useStreamOverlay());

        // when
        act(() => result.current.downloadConnector());

        // then
        await waitFor(() => {
            expect(result.current.actionError).toBe("Could not download the connector.");
        });
    });

    it("leaves the token alone when the streamer backs out of the reset", () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(false);
        const { result } = renderHook(() => useStreamOverlay());

        // when
        act(() => result.current.resetToken());

        // then
        expect(mocks.resetOverlayToken).not.toHaveBeenCalled();
        expect(result.current.success).toBe("");
    });

    it("tells the streamer to download the new connector after a reset", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const { result } = renderHook(() => useStreamOverlay());

        // when
        act(() => result.current.resetToken());

        // then
        await waitFor(() => {
            expect(result.current.success).toBe("Token reset. Download the new connector below.");
        });
        expect(result.current.actionError).toBe("");
    });

    it("surfaces the reason the token could not be reset", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(true);
        mocks.resetOverlayToken.mockRejectedValue(new Error("Try again later."));
        const { result } = renderHook(() => useStreamOverlay());

        // when
        act(() => result.current.resetToken());

        // then
        await waitFor(() => {
            expect(result.current.actionError).toBe("Try again later.");
        });
        expect(result.current.success).toBe("");
    });

    it("surfaces the reason the test overlay never fired", async () => {
        // given
        mocks.testOverlay.mockRejectedValue(new Error("SAMMI is not listening."));
        const { result } = renderHook(() => useStreamOverlay());

        // when
        act(() => result.current.sendTestOverlay());

        // then
        await waitFor(() => {
            expect(result.current.actionError).toBe("SAMMI is not listening.");
        });
    });

    it("clears the previous failure before trying again", async () => {
        // given
        mocks.testOverlay.mockRejectedValueOnce(new Error("SAMMI is not listening."));
        const { result } = renderHook(() => useStreamOverlay());
        act(() => result.current.sendTestOverlay());
        await waitFor(() => {
            expect(result.current.actionError).toBe("SAMMI is not listening.");
        });

        // when
        act(() => result.current.sendTestOverlay());

        // then
        await waitFor(() => {
            expect(result.current.success).toBe("Test overlay sent. Check your SAMMI overlay.");
        });
        expect(result.current.actionError).toBe("");
    });

    it("says copied once the token reaches the clipboard", async () => {
        // given
        const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
        const { result } = renderHook(() => useStreamOverlay());

        // when
        act(() => result.current.copyToken());

        // then
        await waitFor(() => {
            expect(result.current.copied).toBe(true);
        });
        expect(writeText).toHaveBeenCalledWith("sammi-token");
    });

    it("stops saying copied once the notice has run its course", async () => {
        // given
        vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
        const { result } = renderHook(() => useStreamOverlay());
        act(() => result.current.copyToken());
        await waitFor(() => {
            expect(result.current.copied).toBe(true);
        });

        // when
        await act(async () => {
            await new Promise(resolve => setTimeout(resolve, 2100));
        });

        // then
        expect(result.current.copied).toBe(false);
    });

    it("stays quiet when a clipboard write is refused", async () => {
        // given
        vi.spyOn(navigator.clipboard, "writeText").mockRejectedValue(new Error("clipboard blocked"));
        const { result } = renderHook(() => useStreamOverlay());

        // when
        act(() => result.current.copyToken());
        await act(async () => {
            await Promise.resolve();
        });

        // then
        expect(result.current.copied).toBe(false);
        expect(result.current.actionError).toBe("");
    });

    it("does not reach for the clipboard when there is no token yet", () => {
        // given
        mocks.useOverlayConnection.mockReturnValue({ connection: null, loading: false, error: "" });
        const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
        const { result } = renderHook(() => useStreamOverlay());

        // when
        act(() => result.current.copyToken());

        // then
        expect(writeText).not.toHaveBeenCalled();
    });
});
