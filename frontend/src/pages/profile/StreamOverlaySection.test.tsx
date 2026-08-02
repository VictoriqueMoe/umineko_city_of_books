import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { OverlayConnection } from "../../api/endpoints";
import { renderWithProviders } from "../../test-utils/render";
import { StreamOverlaySection } from "./StreamOverlaySection";

const mocks = vi.hoisted(() => ({
    getOverlayConnection: vi.fn(),
    fetchOverlayConnectorSEF: vi.fn(),
    resetOverlayToken: vi.fn(),
    testOverlay: vi.fn(),
}));

vi.mock("../../api/endpoints", () => ({
    getOverlayConnection: mocks.getOverlayConnection,
    fetchOverlayConnectorSEF: mocks.fetchOverlayConnectorSEF,
    resetOverlayToken: mocks.resetOverlayToken,
    testOverlay: mocks.testOverlay,
}));

function makeConnection(overrides: Partial<OverlayConnection> = {}): OverlayConnection {
    return {
        token: "overlay-token-123",
        connect_url: "wss://example.test/overlay",
        connected: false,
        ...overrides,
    };
}

async function setup() {
    const user = userEvent.setup();
    const result = renderWithProviders(<StreamOverlaySection />);
    await screen.findByRole("heading", { name: "Stream Overlay" });

    return { ...result, user };
}

beforeEach(() => {
    mocks.getOverlayConnection.mockResolvedValue(makeConnection());
    mocks.fetchOverlayConnectorSEF.mockResolvedValue("sef-file-body");
    mocks.resetOverlayToken.mockResolvedValue(makeConnection({ token: "overlay-token-456" }));
    mocks.testOverlay.mockResolvedValue({ ok: true });
    vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
});

afterEach(() => {
    vi.restoreAllMocks();
});

describe("StreamOverlaySection", () => {
    it("apologises when the connection cannot be loaded", async () => {
        // given
        mocks.getOverlayConnection.mockRejectedValue(new Error("nope"));

        // when
        await setup();

        // then
        expect(await screen.findByText("Could not load your overlay connection.")).toBeInTheDocument();
        expect(screen.queryByText("Connection token")).not.toBeInTheDocument();
    });

    it("reports that SAMMI has not connected yet", async () => {
        // given
        mocks.getOverlayConnection.mockResolvedValue(makeConnection({ connected: false }));

        // when
        await setup();

        // then
        expect(await screen.findByText("SAMMI not connected")).toBeInTheDocument();
    });

    it("reports that SAMMI is connected", async () => {
        // given
        mocks.getOverlayConnection.mockResolvedValue(makeConnection({ connected: true }));

        // when
        await setup();

        // then
        expect(await screen.findByText("SAMMI connected")).toBeInTheDocument();
    });

    it("shows the connection token the streamer needs", async () => {
        // given
        await setup();

        // when
        const token = await screen.findByText("overlay-token-123");

        // then
        expect(token).toBeInTheDocument();
    });

    it("copies the token to the clipboard and says so", async () => {
        // given
        const { user } = await setup();
        await screen.findByText("overlay-token-123");
        const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);

        // when
        await user.click(screen.getByText("Copy"));

        // then
        expect(writeText).toHaveBeenCalledWith("overlay-token-123");
        expect(await screen.findByText("Copied")).toBeInTheDocument();
    });

    it("stops saying copied a moment later", async () => {
        // given
        const { user } = await setup();
        await screen.findByText("overlay-token-123");
        vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
        await user.click(screen.getByText("Copy"));
        await screen.findByText("Copied");

        // when
        await waitFor(
            () => {
                expect(screen.getByText("Copy")).toBeInTheDocument();
            },
            { timeout: 3000 },
        );

        // then
        expect(screen.queryByText("Copied")).not.toBeInTheDocument();
    });

    it("stops saying copied once the token has been reset", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const { user } = await setup();
        await screen.findByText("overlay-token-123");
        vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
        await user.click(screen.getByText("Copy"));
        await screen.findByText("Copied");

        // when
        await user.click(screen.getByRole("button", { name: "Reset token" }));

        // then
        expect(await screen.findByText("overlay-token-456")).toBeInTheDocument();
        expect(screen.getByText("Copy")).toBeInTheDocument();
    });

    it("downloads the connector file for SAMMI", async () => {
        // given
        const createObjectURL = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:overlay");
        const { user } = await setup();
        await screen.findByText("overlay-token-123");

        // when
        await user.click(screen.getByRole("button", { name: "Download SAMMI connector (.sef)" }));

        // then
        expect(mocks.fetchOverlayConnectorSEF).toHaveBeenCalledOnce();
        expect(createObjectURL).toHaveBeenCalledOnce();
    });

    it("explains why the connector could not be downloaded", async () => {
        // given
        mocks.fetchOverlayConnectorSEF.mockRejectedValue(new Error("The connector is unavailable."));
        const { user } = await setup();
        await screen.findByText("overlay-token-123");

        // when
        await user.click(screen.getByRole("button", { name: "Download SAMMI connector (.sef)" }));

        // then
        expect(await screen.findByText("The connector is unavailable.")).toBeInTheDocument();
    });

    it("marks the overlay as connected after a successful test", async () => {
        // given
        const { user } = await setup();
        await screen.findByText("overlay-token-123");

        // when
        await user.click(screen.getByRole("button", { name: "Send test overlay" }));

        // then
        expect(await screen.findByText("Test overlay sent. Check your SAMMI overlay.")).toBeInTheDocument();
        expect(screen.getByText("SAMMI connected")).toBeInTheDocument();
    });

    it("marks the overlay as disconnected when the test fails", async () => {
        // given
        mocks.getOverlayConnection.mockResolvedValue(makeConnection({ connected: true }));
        mocks.testOverlay.mockRejectedValue(new Error("SAMMI is not listening."));
        const { user } = await setup();
        await screen.findByText("overlay-token-123");

        // when
        await user.click(screen.getByRole("button", { name: "Send test overlay" }));

        // then
        expect(await screen.findByText("SAMMI is not listening.")).toBeInTheDocument();
        expect(screen.getByText("SAMMI not connected")).toBeInTheDocument();
    });

    it("leaves the token alone when the streamer backs out of the reset", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(false);
        const { user } = await setup();
        await screen.findByText("overlay-token-123");

        // when
        await user.click(screen.getByRole("button", { name: "Reset token" }));

        // then
        expect(mocks.resetOverlayToken).not.toHaveBeenCalled();
        expect(screen.getByText("overlay-token-123")).toBeInTheDocument();
    });

    it("replaces the token once the reset is confirmed", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const { user } = await setup();
        await screen.findByText("overlay-token-123");

        // when
        await user.click(screen.getByRole("button", { name: "Reset token" }));

        // then
        expect(await screen.findByText("overlay-token-456")).toBeInTheDocument();
        expect(screen.getByText("Token reset. Download the new connector below.")).toBeInTheDocument();
    });

    it("explains why the token could not be reset", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(true);
        mocks.resetOverlayToken.mockRejectedValue(new Error("Try again later."));
        const { user } = await setup();
        await screen.findByText("overlay-token-123");

        // when
        await user.click(screen.getByRole("button", { name: "Reset token" }));

        // then
        expect(await screen.findByText("Try again later.")).toBeInTheDocument();
        expect(screen.getByText("overlay-token-123")).toBeInTheDocument();
    });

    it("keeps the setup guide folded away until it is asked for", async () => {
        // given
        await setup();
        await screen.findByText("overlay-token-123");

        // when
        const toggle = screen.getByRole("button", { name: /SAMMI setup guide/ });

        // then
        expect(toggle).toHaveAttribute("aria-expanded", "false");
        expect(screen.queryByRole("list")).not.toBeInTheDocument();
    });

    it("unfolds the setup guide when the streamer asks for it", async () => {
        // given
        const { user } = await setup();
        await screen.findByText("overlay-token-123");

        // when
        await user.click(screen.getByRole("button", { name: /SAMMI setup guide/ }));

        // then
        await waitFor(() => expect(screen.getByRole("list")).toBeInTheDocument());
        expect(screen.getByText(/enable the Bridge \/ Deck websocket server/)).toBeInTheDocument();
    });
});
