import { act, fireEvent, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { WSMessageHandler } from "../../../context/notificationContextValue";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders, type ProviderOptions } from "../../../test-utils/render";
import type { WSMessage } from "../../../types/api";
import { Sidebar } from "./Sidebar";

const { badges, counts } = vi.hoisted(() => ({
    badges: {
        hasUnread: vi.fn(),
        hasAnyUnread: vi.fn(),
        markVisited: vi.fn(),
        markAllVisited: vi.fn(),
        anyUnread: false,
    },
    counts: {
        corner: {} as Record<string, number>,
        art: {} as Record<string, number>,
    },
}));

const chatbotList = vi.hoisted(() => ({
    value: [] as { user_id: string; username: string; display_name: string; avatar_url: string }[],
}));

vi.mock("../../../hooks/useSidebarBadges", () => ({ useSidebarBadges: () => badges }));

vi.mock("../../../api/queries/misc", () => ({
    useCornerCounts: () => ({ counts: counts.corner, loading: false }),
    useArtCornerCounts: () => ({ counts: counts.art, loading: false }),
}));

vi.mock("../../../api/queries/chatbot", () => ({
    useChatbotList: () => ({ chatbots: chatbotList.value, loading: false }),
}));

vi.mock("../../../features/easterEgg", () => ({ PieceTrigger: () => null }));

vi.mock("../../AppVersionInfo/AppVersionInfo", () => ({ AppVersionInfo: () => null }));

interface SidebarHandlers {
    open?: boolean;
    onClose?: () => void;
    onCollapse?: () => void;
}

function noop() {}

function renderSidebar(options: ProviderOptions = {}, handlers: SidebarHandlers = {}) {
    return renderWithProviders(
        <Sidebar
            open={handlers.open ?? false}
            onClose={handlers.onClose ?? noop}
            onCollapse={handlers.onCollapse ?? noop}
        />,
        options,
    );
}

function captureWS() {
    const handlers: WSMessageHandler[] = [];

    function addWSListener(handler: WSMessageHandler) {
        handlers.push(handler);
        return () => {};
    }

    function emit(msg: WSMessage) {
        act(() => {
            for (const handler of handlers) {
                handler(msg);
            }
        });
    }

    return { addWSListener, emit };
}

beforeEach(() => {
    badges.hasUnread.mockReturnValue(false);
    badges.hasAnyUnread.mockReturnValue(false);
    badges.anyUnread = false;
    counts.corner = {};
    counts.art = {};
});

describe("Sidebar for a signed out visitor", () => {
    it("shows the browse links but neither the create nor the account section", () => {
        // given
        const user = null;

        // when
        renderSidebar({ user });

        // then
        expect(screen.getByRole("link", { name: "Welcome" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Mysteries" })).toBeInTheDocument();
        expect(screen.queryByText("Create")).not.toBeInTheDocument();
        expect(screen.queryByText("Account")).not.toBeInTheDocument();
        expect(screen.queryByRole("link", { name: "Settings" })).not.toBeInTheDocument();
        expect(screen.queryByRole("link", { name: "Profile" })).not.toBeInTheDocument();
    });

    it("hides the rules link while the site has no rules page", () => {
        // given
        const siteInfo = { rules_page: "" };

        // when
        renderSidebar({ siteInfo });

        // then
        expect(screen.queryByRole("link", { name: "Rules" })).not.toBeInTheDocument();
    });

    it("treats a rules page of nothing but whitespace as no rules page", () => {
        // given
        const siteInfo = { rules_page: "   \n  " };

        // when
        renderSidebar({ siteInfo });

        // then
        expect(screen.queryByRole("link", { name: "Rules" })).not.toBeInTheDocument();
    });

    it("shows the rules link once the site has a rules page", () => {
        // given
        const siteInfo = { rules_page: "Be kind to the furniture." };

        // when
        renderSidebar({ siteInfo });

        // then
        expect(screen.getByRole("link", { name: "Rules" })).toHaveAttribute("href", "/rules");
    });

    it("keeps My Games out of the games group for a signed out visitor", () => {
        // given
        const route = "/games/live";

        // when
        renderSidebar({ user: null, route });

        // then
        expect(screen.getByRole("link", { name: "Live Games" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Past Games" })).toBeInTheDocument();
        expect(screen.queryByRole("link", { name: "My Games" })).not.toBeInTheDocument();
    });

    it("keeps the staff section away from a signed out visitor", () => {
        // given
        const user = null;

        // when
        renderSidebar({ user });

        // then
        expect(screen.queryByText("Admin")).not.toBeInTheDocument();
        expect(screen.queryByText("Moderation")).not.toBeInTheDocument();
    });
});

describe("Sidebar for a signed in member", () => {
    it("adds the create and account sections", () => {
        // given
        const user = makeUser();

        // when
        renderSidebar({ user });

        // then
        expect(screen.getByText("Create")).toBeInTheDocument();
        expect(screen.getByText("Account")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "New Mystery" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Settings" })).toBeInTheDocument();
    });

    it("points the profile link at the member's own page", () => {
        // given
        const user = makeUser({ username: "lambdadelta" });

        // when
        renderSidebar({ user });

        // then
        expect(screen.getByRole("link", { name: "Profile" })).toHaveAttribute("href", "/user/lambdadelta");
    });

    it("offers My Games to a signed in member", () => {
        // given
        const user = makeUser();

        // when
        renderSidebar({ user, route: "/games" });

        // then
        expect(screen.getByRole("link", { name: "My Games" })).toHaveAttribute("href", "/games");
    });

    it("still keeps the staff section away from an ordinary member", () => {
        // given
        const user = makeUser({ role: undefined });

        // when
        renderSidebar({ user });

        // then
        expect(screen.queryByRole("link", { name: /Panel/ })).not.toBeInTheDocument();
    });
});

describe("Sidebar staff section", () => {
    it("calls the section Moderation and links to the moderator panel for a moderator", () => {
        // given
        const user = makeUser({ role: "moderator" });

        // when
        renderSidebar({ user });

        // then
        expect(screen.getByText("Moderation")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Moderator Panel" })).toHaveAttribute("href", "/admin");
        expect(screen.queryByRole("link", { name: "Admin Panel" })).not.toBeInTheDocument();
    });

    it("calls the section Admin and links to the admin panel for an admin", () => {
        // given
        const user = makeUser({ role: "admin" });

        // when
        renderSidebar({ user });

        // then
        expect(screen.getByText("Admin")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Admin Panel" })).toHaveAttribute("href", "/admin");
    });

    it("gives a super admin the admin panel too", () => {
        // given
        const user = makeUser({ role: "super_admin" });

        // when
        renderSidebar({ user });

        // then
        expect(screen.getByRole("link", { name: "Admin Panel" })).toBeInTheDocument();
    });
});

describe("Sidebar unread counts", () => {
    it("shows the unread notification count beside the notifications link", () => {
        // given
        const notification = { unreadCount: 4 };

        // when
        renderSidebar({ user: makeUser(), notification });

        // then
        expect(screen.getByText("4").closest("a")).toHaveAttribute("href", "/notifications");
    });

    it("caps the notification count once it passes ninety nine", () => {
        // given
        const notification = { unreadCount: 250 };

        // when
        renderSidebar({ user: makeUser(), notification });

        // then
        expect(screen.getByText("99+").closest("a")).toHaveAttribute("href", "/notifications");
        expect(screen.queryByText("250")).not.toBeInTheDocument();
    });

    it("shows the unread chat count beside the chat link", () => {
        // given
        const notification = { chatUnreadCount: 12 };

        // when
        renderSidebar({ user: makeUser(), notification });

        // then
        expect(screen.getByText("12").closest("a")).toHaveAttribute("href", "/chat");
    });

    it("shows how many streams are live beside the live link", () => {
        // given
        const notification = { liveStreamsCount: 3 };

        // when
        renderSidebar({ notification });

        // then
        expect(screen.getByText("3").closest("a")).toHaveAttribute("href", "/live");
    });

    it("shows how many games are live beside the live games link", () => {
        // given
        const notification = { liveGamesCount: 120 };

        // when
        renderSidebar({ notification, route: "/games/live" });

        // then
        expect(screen.getByText("99+").closest("a")).toHaveAttribute("href", "/games/live");
    });

    it("leaves every badge off when nothing is unread or live", () => {
        // given
        const notification = { unreadCount: 0, chatUnreadCount: 0, liveStreamsCount: 0, liveGamesCount: 0 };

        // when
        renderSidebar({ user: makeUser(), notification, route: "/games/live" });

        // then
        expect(screen.getByRole("link", { name: "Notifications" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Chat" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Live" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Live Games" })).toBeInTheDocument();
    });
});

describe("Sidebar activity badges", () => {
    it("marks the game board group when one of its corners has new content", () => {
        // given
        badges.hasAnyUnread.mockImplementation((keys: string[]) => keys.includes("game_board_umineko"));

        // when
        renderSidebar({ user: makeUser() });

        // then
        const gameBoard = screen.getByRole("button", { name: /^Game Board/ });
        expect(within(gameBoard).getByLabelText("new content")).toBeInTheDocument();
    });

    it("leaves every group unmarked when nothing new has arrived", () => {
        // given
        badges.hasAnyUnread.mockReturnValue(false);

        // when
        renderSidebar({ user: makeUser() });

        // then
        expect(screen.queryAllByLabelText("new content")).toHaveLength(0);
    });

    it("marks an individual browse link that has new content", () => {
        // given
        badges.hasUnread.mockImplementation((key: string) => key === "mysteries");

        // when
        renderSidebar({ user: makeUser() });

        // then
        const mysteries = screen.getByRole("link", { name: /^Mysteries/ });
        expect(within(mysteries).getByLabelText("new content")).toBeInTheDocument();
    });

    it("records a visit when a browse link is followed", async () => {
        // given
        const user = userEvent.setup();
        renderSidebar({ user: makeUser() });

        // when
        await user.click(screen.getByRole("link", { name: "Secrets" }));

        // then
        expect(badges.markVisited).toHaveBeenCalledWith("secrets");
    });

    it("hides the mark all as read control while nothing is unread", () => {
        // given
        badges.anyUnread = false;

        // when
        renderSidebar({ user: makeUser() });

        // then
        expect(screen.queryByRole("button", { name: "Mark all as read" })).not.toBeInTheDocument();
    });

    it("marks everything as read when the control is pressed", async () => {
        // given
        badges.anyUnread = true;
        const user = userEvent.setup();
        renderSidebar({ user: makeUser() });

        // when
        await user.click(screen.getByRole("button", { name: "Mark all as read" }));

        // then
        expect(badges.markAllVisited).toHaveBeenCalledOnce();
    });
});

describe("Sidebar expandable groups", () => {
    it("opens the game board corners on a game board route and shows their post counts", () => {
        // given
        counts.corner = { general: 3, umineko: 12, higurashi: 4, ciconia: 1 };

        // when
        renderSidebar({ route: "/game-board" });

        // then
        expect(screen.getByText("12").closest("a")).toHaveAttribute("href", "/game-board/umineko");
        expect(screen.getByText("3").closest("a")).toHaveAttribute("href", "/game-board");
        expect(screen.getByText("1").closest("a")).toHaveAttribute("href", "/game-board/ciconia");
    });

    it("falls back to zero for a corner the counts do not mention", () => {
        // given
        counts.corner = { umineko: 12 };

        // when
        renderSidebar({ route: "/game-board" });

        // then
        expect(within(screen.getByRole("link", { name: /^Ciconia/ })).getByText("0")).toBeInTheDocument();
    });

    it("keeps the corners collapsed on an unrelated route until the group is pressed", async () => {
        // given
        const user = userEvent.setup();
        renderSidebar({ route: "/" });
        expect(screen.queryByRole("link", { name: /^Higurashi/ })).not.toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: /^Game Board/ }));

        // then
        expect(screen.getByRole("link", { name: /^Higurashi/ })).toHaveAttribute("href", "/game-board/higurashi");
    });

    it("closes an automatically opened group when its header is pressed", async () => {
        // given
        const user = userEvent.setup();
        renderSidebar({ route: "/game-board" });

        // when
        await user.click(screen.getByRole("button", { name: /^Game Board/ }));

        // then
        expect(screen.queryByRole("link", { name: /^Higurashi/ })).not.toBeInTheDocument();
    });

    it("opens Ryukishi's other works instead of the corners on one of those routes", () => {
        // given
        const route = "/game-board/higanbana";

        // when
        renderSidebar({ route });

        // then
        expect(screen.getByRole("link", { name: /^Higanbana/ })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /^Rose Guns Days/ })).toBeInTheDocument();
        expect(screen.queryByRole("link", { name: /^Ciconia/ })).not.toBeInTheDocument();
    });

    it("opens the gallery with its own art counts on a gallery route", () => {
        // given
        counts.corner = { umineko: 12 };
        counts.art = { umineko: 7 };

        // when
        renderSidebar({ route: "/gallery" });

        // then
        expect(screen.getByText("7").closest("a")).toHaveAttribute("href", "/gallery/umineko");
        expect(screen.queryByText("12")).not.toBeInTheDocument();
    });

    it("opens the theory links on a theories route", () => {
        // given
        const route = "/theories";

        // when
        renderSidebar({ route });

        // then
        expect(screen.getByRole("link", { name: /^Higurashi/ })).toHaveAttribute("href", "/theories/higurashi");
        expect(screen.getByRole("link", { name: /^Ciconia/ })).toHaveAttribute("href", "/theories/ciconia");
    });

    it("records a visit against the matching theory corner when its link is followed", async () => {
        // given
        const user = userEvent.setup();
        renderSidebar({ user: makeUser(), route: "/theories" });

        // when
        await user.click(screen.getByRole("link", { name: /^Ciconia/ }));

        // then
        expect(badges.markVisited).toHaveBeenCalledWith("theories_ciconia");
    });

    it("opens the new theory links for a member writing a theory", () => {
        // given
        const route = "/theory/new";

        // when
        renderSidebar({ user: makeUser(), route });

        // then
        const hrefs = screen.getAllByRole("link", { name: "Higurashi" }).map(link => link.getAttribute("href"));
        expect(hrefs).toContain("/theory/higurashi/new");
    });

    it("keeps the new theory links away from a signed out visitor", () => {
        // given
        const route = "/theory/new";

        // when
        renderSidebar({ user: null, route });

        // then
        const hrefs = screen.getAllByRole("link").map(link => link.getAttribute("href"));
        expect(hrefs).not.toContain("/theory/higurashi/new");
    });

    it("keeps the games group closed on an unrelated route", () => {
        // given
        const route = "/";

        // when
        renderSidebar({ user: makeUser(), route });

        // then
        expect(screen.queryByRole("link", { name: /^Live Games/ })).not.toBeInTheDocument();
    });
});

describe("Sidebar announcements", () => {
    it("flags an announcement written by somebody else", () => {
        // given
        const ws = captureWS();
        renderSidebar({ user: makeUser(), notification: { addWSListener: ws.addWSListener } });

        // when
        ws.emit({ type: "new_announcement", data: { author_id: "someone-else" } });

        // then
        const link = screen.getByRole("link", { name: /^Announcements/ });
        expect(within(link).getByText("New")).toBeInTheDocument();
    });

    it("ignores an announcement the signed in member wrote themselves", () => {
        // given
        const ws = captureWS();
        const account = makeUser({ id: "author-1" });
        renderSidebar({ user: account, notification: { addWSListener: ws.addWSListener } });

        // when
        ws.emit({ type: "new_announcement", data: { author_id: "author-1" } });

        // then
        const link = screen.getByRole("link", { name: /^Announcements/ });
        expect(within(link).queryByText("New")).not.toBeInTheDocument();
    });

    it("ignores an announcement while the announcements page is already open", () => {
        // given
        const ws = captureWS();
        renderSidebar({ notification: { addWSListener: ws.addWSListener }, route: "/announcements" });

        // when
        ws.emit({ type: "new_announcement", data: { author_id: "someone-else" } });

        // then
        const link = screen.getByRole("link", { name: /^Announcements/ });
        expect(within(link).queryByText("New")).not.toBeInTheDocument();
    });

    it("ignores websocket traffic that is not an announcement", () => {
        // given
        const ws = captureWS();
        renderSidebar({ notification: { addWSListener: ws.addWSListener } });

        // when
        ws.emit({ type: "chat_message", data: { author_id: "someone-else" } });

        // then
        const link = screen.getByRole("link", { name: /^Announcements/ });
        expect(within(link).queryByText("New")).not.toBeInTheDocument();
    });

    it("clears the flag once the announcements link is followed", async () => {
        // given
        const ws = captureWS();
        const clicker = userEvent.setup();
        renderSidebar({ notification: { addWSListener: ws.addWSListener } });
        ws.emit({ type: "new_announcement", data: { author_id: "someone-else" } });

        // when
        await clicker.click(screen.getByRole("link", { name: /^Announcements/ }));

        // then
        const link = screen.getByRole("link", { name: /^Announcements/ });
        expect(within(link).queryByText("New")).not.toBeInTheDocument();
    });
});

describe("Sidebar shell", () => {
    it("leaves the page uncovered while the sidebar is closed", () => {
        // given
        const open = false;

        // when
        const { container } = renderSidebar({}, { open });

        // then
        expect(container.firstElementChild?.tagName).toBe("ASIDE");
    });

    it("covers the page with a dismissable overlay while the sidebar is open", () => {
        // given
        const onClose = vi.fn();
        const { container } = renderSidebar({}, { open: true, onClose });

        // when
        const overlay = container.firstElementChild as HTMLElement;
        fireEvent.click(overlay);

        // then
        expect(overlay.tagName).toBe("DIV");
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("asks its owner to collapse when the collapse control is pressed", async () => {
        // given
        const onCollapse = vi.fn();
        const clicker = userEvent.setup();
        renderSidebar({}, { onCollapse });

        // when
        await clicker.click(screen.getByRole("button", { name: "Collapse sidebar" }));

        // then
        expect(onCollapse).toHaveBeenCalledOnce();
    });

    it("closes itself when a navigation link is followed", async () => {
        // given
        const onClose = vi.fn();
        const clicker = userEvent.setup();
        renderSidebar({}, { onClose });

        // when
        await clicker.click(screen.getByRole("link", { name: "Welcome" }));

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("closes itself when the board title is followed", async () => {
        // given
        const onClose = vi.fn();
        const clicker = userEvent.setup();
        renderSidebar({}, { onClose });

        // when
        await clicker.click(screen.getByRole("link", { name: "Umineko Game Board" }));

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });
});

describe("Sidebar chatbots", () => {
    beforeEach(() => {
        chatbotList.value = [];
    });

    it("hides the section entirely when the site offers no characters", () => {
        // given
        chatbotList.value = [];

        // when
        renderSidebar();

        // then
        expect(screen.queryByRole("button", { name: /^Chatbots/ })).not.toBeInTheDocument();
    });

    it("lists each character and links to their profile", async () => {
        // given
        chatbotList.value = [
            { user_id: "u1", username: "beato", display_name: "Beatrice", avatar_url: "" },
            { user_id: "u2", username: "bern", display_name: "Bernkastel", avatar_url: "" },
        ];
        const user = userEvent.setup();
        renderSidebar();

        // when
        await user.click(screen.getByRole("button", { name: /^Chatbots/ }));

        // then
        expect(screen.getByRole("link", { name: "Beatrice" })).toHaveAttribute("href", "/user/beato");
        expect(screen.getByRole("link", { name: "Bernkastel" })).toHaveAttribute("href", "/user/bern");
    });

    it("keeps the list collapsed until it is opened", () => {
        // given
        chatbotList.value = [{ user_id: "u1", username: "beato", display_name: "Beatrice", avatar_url: "" }];

        // when
        renderSidebar();

        // then
        expect(screen.getByRole("button", { name: /^Chatbots/ })).toBeInTheDocument();
        expect(screen.queryByRole("link", { name: "Beatrice" })).not.toBeInTheDocument();
    });
});
