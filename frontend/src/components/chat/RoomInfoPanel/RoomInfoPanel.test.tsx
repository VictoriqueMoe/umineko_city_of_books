import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../../test-utils/render";
import { RoomInfoPanel, type RoomInfoTab } from "./RoomInfoPanel";

const mocks = vi.hoisted(() => ({
    searchRendered: vi.fn(),
    pinsRendered: vi.fn(),
}));

vi.mock("./tabs/SearchTab", () => ({
    SearchTab: (props: { isActive: boolean }) => {
        mocks.searchRendered(props.isActive);
        return <div data-testid="search-tab" />;
    },
}));

vi.mock("./tabs/PinsTab", () => ({
    PinsTab: (props: { isActive: boolean; canUnpin: boolean }) => {
        mocks.pinsRendered(props.isActive);
        return <div data-testid="pins-tab" data-can-unpin={String(props.canUnpin)} />;
    },
}));

function renderPanel(tab: RoomInfoTab | null = "search") {
    const onTabChange = vi.fn();
    const onClose = vi.fn();
    renderWithProviders(
        <RoomInfoPanel
            roomId="room-1"
            tab={tab}
            onTabChange={onTabChange}
            onClose={onClose}
            onJump={vi.fn()}
            canUnpin
        />,
    );

    return { onTabChange, onClose };
}

describe("RoomInfoPanel", () => {
    it("renders nothing while no tab is open", () => {
        // given
        const { container } = { container: document.body };

        // when
        renderPanel(null);

        // then
        expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
        expect(container).toBeTruthy();
    });

    it("mounts only the active tab, so an unvisited tab fetches nothing", () => {
        // given
        mocks.searchRendered.mockClear();
        mocks.pinsRendered.mockClear();

        // when
        renderPanel("search");

        // then
        expect(screen.getByTestId("search-tab")).toBeInTheDocument();
        expect(screen.queryByTestId("pins-tab")).not.toBeInTheDocument();
        expect(mocks.pinsRendered).not.toHaveBeenCalled();
    });

    it("asks to change tab when another tab is clicked", async () => {
        // given
        const user = userEvent.setup();
        const { onTabChange } = renderPanel("search");

        // when
        await user.click(screen.getByRole("tab", { name: "Pins" }));

        // then
        expect(onTabChange).toHaveBeenCalledWith("pins");
    });

    it("ties the open panel to the tab that controls it", () => {
        // given
        renderPanel("pins");

        // when
        const panel = screen.getByRole("tabpanel");

        // then
        expect(panel).toHaveAttribute("aria-labelledby", screen.getByRole("tab", { name: "Pins" }).id);
    });

    it("closes on Escape", async () => {
        // given
        const user = userEvent.setup();
        const { onClose } = renderPanel("search");

        // when
        await user.keyboard("{Escape}");

        // then
        expect(onClose).toHaveBeenCalled();
    });

    it("closes when the backdrop behind it is clicked", async () => {
        // given
        const user = userEvent.setup();
        const { onClose } = renderPanel("search");

        // when
        await user.click(screen.getByRole("dialog").parentElement as HTMLElement);

        // then
        expect(onClose).toHaveBeenCalled();
    });
});
