import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { messageActions } from "../../../domain/chat/messageActions";
import { renderWithProviders } from "../../../test-utils/render";
import { MessageActionMenu } from "./MessageActionMenu";

const ITEMS = messageActions({
    pinned: false,
    isOwn: true,
    editing: false,
    senderIsStaff: false,
    canToggleReaction: true,
    canPin: true,
    canModerate: true,
    canEdit: true,
    hasReply: true,
    hasPinToggle: true,
    hasEdit: true,
    hasDelete: true,
});

interface MenuHarness {
    onSelect: ReturnType<typeof vi.fn>;
    onDismiss: ReturnType<typeof vi.fn>;
}

function renderMenu(): MenuHarness {
    const onSelect = vi.fn();
    const onDismiss = vi.fn();

    renderWithProviders(
        <MessageActionMenu items={ITEMS} origin={{ x: 120, y: 80 }} onSelect={onSelect} onDismiss={onDismiss} />,
    );

    return { onSelect, onDismiss };
}

describe("MessageActionMenu", () => {
    it("renders every action it was handed as a menu item", () => {
        // given
        const items = ITEMS;

        // when
        renderMenu();

        // then
        expect(screen.getAllByRole("menuitem").map(item => item.textContent)).toEqual(
            items.map(item => `${item.icon}${item.label}`),
        );
    });

    it("leaves the message toolbar's own buttons unambiguous by not claiming the button role", () => {
        // given
        const label = "Reply";

        // when
        renderMenu();

        // then
        expect(screen.getByRole("menuitem", { name: label })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: label })).not.toBeInTheDocument();
    });

    it("places itself at the point it was opened from before it is painted", () => {
        // given
        const origin = { x: 120, y: 80 };

        // when
        renderMenu();

        // then
        const menu = screen.getByRole("menu", { name: "Message actions" });
        expect(menu.style.left).toBe(`${origin.x}px`);
        expect(menu.style.top).toBe(`${origin.y}px`);
        expect(menu.style.visibility).toBe("visible");
    });

    it("moves focus into the menu so a keyboard can drive it", () => {
        // given
        const first = ITEMS[0].label;

        // when
        renderMenu();

        // then
        expect(screen.getByRole("menuitem", { name: first })).toHaveFocus();
    });

    it("walks down the items on ArrowDown", async () => {
        // given
        const user = userEvent.setup();
        renderMenu();

        // when
        await user.keyboard("{ArrowDown}");

        // then
        expect(screen.getByRole("menuitem", { name: ITEMS[1].label })).toHaveFocus();
    });

    it("wraps to the last item on ArrowUp from the first", async () => {
        // given
        const user = userEvent.setup();
        renderMenu();

        // when
        await user.keyboard("{ArrowUp}");

        // then
        expect(screen.getByRole("menuitem", { name: ITEMS[ITEMS.length - 1].label })).toHaveFocus();
    });

    it("jumps to the last item on End and back on Home", async () => {
        // given
        const user = userEvent.setup();
        renderMenu();

        // when
        await user.keyboard("{End}");
        const last = document.activeElement;
        await user.keyboard("{Home}");

        // then
        expect(last).toBe(screen.getByRole("menuitem", { name: ITEMS[ITEMS.length - 1].label }));
        expect(screen.getByRole("menuitem", { name: ITEMS[0].label })).toHaveFocus();
    });

    it("reports the action the reader chose", async () => {
        // given
        const user = userEvent.setup();
        const { onSelect } = renderMenu();

        // when
        await user.click(screen.getByRole("menuitem", { name: "Reply" }));

        // then
        expect(onSelect).toHaveBeenCalledWith("reply", { x: 120, y: 80 });
    });

    it("activates the focused item from the keyboard", async () => {
        // given
        const user = userEvent.setup();
        const { onSelect } = renderMenu();

        // when
        await user.keyboard("{Enter}");

        // then
        expect(onSelect).toHaveBeenCalledWith(ITEMS[0].id, { x: 120, y: 80 });
    });

    it("hands focus back to the message when Escape closes it", async () => {
        // given
        const user = userEvent.setup();
        const { onDismiss } = renderMenu();

        // when
        await user.keyboard("{Escape}");

        // then
        expect(onDismiss).toHaveBeenCalledWith(true);
    });

    it("closes without stealing focus when the page is clicked elsewhere", () => {
        // given
        const { onDismiss } = renderMenu();

        // when
        fireEvent.mouseDown(document.body);

        // then
        expect(onDismiss).toHaveBeenCalledWith(false);
    });

    it("stays open while the message list scrolls underneath it", () => {
        // given
        const { onDismiss } = renderMenu();

        // when
        fireEvent.scroll(document, {});
        fireEvent.scroll(window, {});

        // then
        expect(onDismiss).not.toHaveBeenCalled();
        expect(screen.getByRole("menu", { name: "Message actions" })).toBeInTheDocument();
    });

    it("marks deleting as the destructive choice and reacting as the one that opens more", () => {
        // given
        const destructive = "Delete message";

        // when
        renderMenu();

        // then
        expect(screen.getByRole("menuitem", { name: destructive }).className).toMatch(/itemDestructive/);
        expect(screen.getByRole("menuitem", { name: "React" })).toHaveAttribute("aria-haspopup", "dialog");
    });
});
