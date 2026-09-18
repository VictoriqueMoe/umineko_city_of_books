import { createRef } from "react";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { WatchPartyMobileView, type WatchPartyMobileViewProps } from "./WatchPartyMobileView";

const viewportDescriptor = Object.getOwnPropertyDescriptor(window, "visualViewport");
const innerHeightDescriptor = Object.getOwnPropertyDescriptor(window, "innerHeight");

function restore(key: string, descriptor: PropertyDescriptor | undefined): void {
    if (descriptor) {
        Object.defineProperty(window, key, descriptor);
        return;
    }

    Reflect.deleteProperty(window, key);
}

function installViewport(height: number, innerHeight: number): void {
    Object.defineProperty(window, "innerHeight", { configurable: true, writable: true, value: innerHeight });
    Object.defineProperty(window, "visualViewport", {
        configurable: true,
        writable: true,
        value: { height, offsetTop: 0, addEventListener: () => {}, removeEventListener: () => {} },
    });
}

function renderView(overrides: Partial<WatchPartyMobileViewProps> = {}) {
    const props: WatchPartyMobileViewProps = {
        title: "Chiru night",
        hasControl: false,
        canEnd: true,
        busy: false,
        copied: false,
        watcherCount: 4,
        voiceCount: 2,
        onCopyInvite: vi.fn(),
        onHide: vi.fn(),
        onLeave: vi.fn(),
        onEnd: vi.fn(),
        stageRef: createRef<HTMLDivElement>(),
        stage: <div>stage node</div>,
        chat: <div>chat pane</div>,
        voice: <div>voice pane</div>,
        people: <div>people pane</div>,
        ...overrides,
    };

    const result = renderWithProviders(<WatchPartyMobileView {...props} />);

    return { ...result, props };
}

async function openMenu(user: ReturnType<typeof userEvent.setup>) {
    await user.click(screen.getByRole("button", { name: "Party actions" }));
}

describe("WatchPartyMobileView", () => {
    afterEach(() => {
        restore("visualViewport", viewportDescriptor);
        restore("innerHeight", innerHeightDescriptor);
        document.documentElement.style.removeProperty("--chat-vh");
        document.documentElement.style.removeProperty("--kb-inset");
    });

    it("sizes itself to the visible viewport so the on-screen keyboard cannot cover the composer", () => {
        // given
        installViewport(520, 900);

        // when
        renderView();

        // then
        expect(document.documentElement.style.getPropertyValue("--chat-vh")).toBe("520px");
        expect(document.documentElement.style.getPropertyValue("--kb-inset")).toBe("380px");
    });

    it("shows the party title and the stage", () => {
        // given
        const title = "Chiru night";

        // when
        renderView({ title });

        // then
        expect(screen.getByText(title)).toBeInTheDocument();
        expect(screen.getByText("stage node")).toBeInTheDocument();
    });

    it("shows the watcher count and the voice count", () => {
        // given
        const counts = { watcherCount: 7, voiceCount: 3 };

        // when
        renderView(counts);

        // then
        expect(screen.getByLabelText("7 watching")).toHaveTextContent("7");
        expect(screen.getByRole("tab", { name: "Voice (3)" })).toBeInTheDocument();
        expect(screen.getByRole("tab", { name: "People (7)" })).toBeInTheDocument();
    });

    it("drops the count from the voice tab while nobody is in voice", () => {
        // given
        const counts = { voiceCount: 0 };

        // when
        renderView(counts);

        // then
        expect(screen.getByRole("tab", { name: "Voice" })).toBeInTheDocument();
    });

    it("shows chat first and swaps the visible pane when another tab is chosen", async () => {
        // given
        const user = userEvent.setup();
        renderView();

        // when
        await user.click(screen.getByRole("tab", { name: "People (4)" }));

        // then
        expect(screen.getByText("people pane")).toBeVisible();
        expect(screen.getByText("chat pane")).not.toBeVisible();
        expect(screen.getByRole("tab", { name: "People (4)" })).toHaveAttribute("aria-selected", "true");
        expect(screen.getByRole("tab", { name: "Chat" })).toHaveAttribute("aria-selected", "false");
    });

    it("keeps the chat pane mounted while another pane is on top", async () => {
        // given
        const user = userEvent.setup();
        renderView();

        // when
        await user.click(screen.getByRole("tab", { name: "Voice (2)" }));

        // then
        expect(screen.getByText("voice pane")).toBeVisible();
        expect(screen.getByText("chat pane")).toBeInTheDocument();
    });

    it("hides the party actions until the overflow button is pressed", async () => {
        // given
        const user = userEvent.setup();
        renderView();

        // when
        await openMenu(user);

        // then
        expect(screen.getByRole("menu")).toBeInTheDocument();
        expect(screen.getByRole("menuitem", { name: "Copy invite" })).toBeInTheDocument();
        expect(screen.getByRole("menuitem", { name: "Hide" })).toBeInTheDocument();
        expect(screen.getByRole("menuitem", { name: "Leave" })).toBeInTheDocument();
        expect(screen.getByRole("menuitem", { name: "End for everyone" })).toBeInTheDocument();
    });

    it("copies the invite and closes the menu", async () => {
        // given
        const user = userEvent.setup();
        const { props } = renderView();

        // when
        await openMenu(user);
        await user.click(screen.getByRole("menuitem", { name: "Copy invite" }));

        // then
        expect(props.onCopyInvite).toHaveBeenCalledOnce();
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    });

    it("hides the party window from the menu", async () => {
        // given
        const user = userEvent.setup();
        const { props } = renderView();

        // when
        await openMenu(user);
        await user.click(screen.getByRole("menuitem", { name: "Hide" }));

        // then
        expect(props.onHide).toHaveBeenCalledOnce();
    });

    it("leaves the party from the menu", async () => {
        // given
        const user = userEvent.setup();
        const { props } = renderView();

        // when
        await openMenu(user);
        await user.click(screen.getByRole("menuitem", { name: "Leave" }));

        // then
        expect(props.onLeave).toHaveBeenCalledOnce();
    });

    it("ends the party for everyone from the menu", async () => {
        // given
        const user = userEvent.setup();
        const { props } = renderView();

        // when
        await openMenu(user);
        await user.click(screen.getByRole("menuitem", { name: "End for everyone" }));

        // then
        expect(props.onEnd).toHaveBeenCalledOnce();
    });

    it("never offers to end the party to someone who may not end it", async () => {
        // given
        const user = userEvent.setup();
        renderView({ canEnd: false });

        // when
        await openMenu(user);

        // then
        expect(screen.queryByRole("menuitem", { name: "End for everyone" })).not.toBeInTheDocument();
        expect(screen.getByRole("menuitem", { name: "Leave" })).toBeInTheDocument();
    });

    it("blocks leaving and ending while a party action is in flight", async () => {
        // given
        const user = userEvent.setup();
        renderView({ busy: true });

        // when
        await openMenu(user);

        // then
        expect(screen.getByRole("menuitem", { name: "Leave" })).toBeDisabled();
        expect(screen.getByRole("menuitem", { name: "End for everyone" })).toBeDisabled();
        expect(screen.getByRole("menuitem", { name: "Copy invite" })).toBeEnabled();
    });

    it("tells the reader the invite link was copied", async () => {
        // given
        const user = userEvent.setup();
        renderView({ copied: true });

        // when
        await openMenu(user);

        // then
        expect(screen.getByRole("status")).toHaveTextContent("Link copied");
        expect(screen.getByRole("menuitem", { name: "Link copied" })).toBeInTheDocument();
    });

    it("closes the menu on Escape", async () => {
        // given
        const user = userEvent.setup();
        renderView();
        await openMenu(user);

        // when
        await user.keyboard("{Escape}");

        // then
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Party actions" })).toHaveFocus();
    });

    it("badges the viewer who holds control", () => {
        // given
        const holder = { hasControl: true };

        // when
        renderView(holder);

        // then
        expect(screen.getByText("Controller")).toBeInTheDocument();
    });

    it("leaves the controller badge off everyone else", () => {
        // given
        const watcher = { hasControl: false };

        // when
        renderView(watcher);

        // then
        expect(screen.queryByText("Controller")).not.toBeInTheDocument();
    });
});
