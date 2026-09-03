import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils/render";
import { TabStrip } from "./TabStrip";
import { tabId, tabPanelId } from "./tabIds";

const TABS = [
    { id: "search" as const, label: "Search" },
    { id: "pins" as const, label: "Pins" },
    { id: "media" as const, label: "Media" },
];

function renderStrip(active: (typeof TABS)[number]["id"] = "search") {
    const onSelect = vi.fn();
    renderWithProviders(
        <TabStrip tabs={TABS} active={active} onSelect={onSelect} idPrefix="room" ariaLabel="Room information" />,
    );

    return { onSelect };
}

describe("TabStrip", () => {
    it("marks only the active tab as selected and keeps it the only stop in the tab order", () => {
        // given
        renderStrip("pins");

        // when
        const tabs = screen.getAllByRole("tab");

        // then
        expect(tabs.map(t => t.getAttribute("aria-selected"))).toEqual(["false", "true", "false"]);
        expect(tabs.map(t => t.getAttribute("tabindex"))).toEqual(["-1", "0", "-1"]);
    });

    it("points each tab at the panel it controls", () => {
        // given
        renderStrip();

        // when
        const tab = screen.getByRole("tab", { name: "Search" });

        // then
        expect(tab).toHaveAttribute("id", tabId("room", "search"));
        expect(tab).toHaveAttribute("aria-controls", tabPanelId("room", "search"));
    });

    it("moves right with the arrow key and wraps at the end", async () => {
        // given
        const user = userEvent.setup();
        const { onSelect } = renderStrip("media");
        await user.click(screen.getByRole("tab", { name: "Media" }));
        onSelect.mockClear();

        // when
        await user.keyboard("{ArrowRight}");

        // then
        expect(onSelect).toHaveBeenCalledWith("search");
    });

    it("moves left with the arrow key and wraps at the start", async () => {
        // given
        const user = userEvent.setup();
        const { onSelect } = renderStrip("search");
        await user.click(screen.getByRole("tab", { name: "Search" }));
        onSelect.mockClear();

        // when
        await user.keyboard("{ArrowLeft}");

        // then
        expect(onSelect).toHaveBeenCalledWith("media");
    });

    it("jumps to the first and last tabs with Home and End", async () => {
        // given
        const user = userEvent.setup();
        const { onSelect } = renderStrip("pins");
        await user.click(screen.getByRole("tab", { name: "Pins" }));
        onSelect.mockClear();

        // when
        await user.keyboard("{End}");
        await user.keyboard("{Home}");

        // then
        expect(onSelect).toHaveBeenNthCalledWith(1, "media");
        expect(onSelect).toHaveBeenNthCalledWith(2, "search");
    });
});
