import { act, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { PropsWithChildren } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import { makeUser } from "./test-utils/fixtures";
import { renderWithProviders, type ProviderOptions } from "./test-utils/render";

const { initPush, ensureNotificationPermission } = vi.hoisted(() => ({
    initPush: vi.fn(),
    ensureNotificationPermission: vi.fn(),
}));

vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, BrowserRouter: ({ children }: PropsWithChildren) => <>{children}</> };
});

vi.mock("./pages/lazyPages", async () => {
    const { Outlet } = await import("react-router");
    const names = [
        "AdminAnnouncementsPage",
        "AdminAuditLog",
        "AdminBannedGifs",
        "AdminBannedWords",
        "AdminChatbots",
        "AdminContentRules",
        "AdminDashboard",
        "AdminInvites",
        "AdminPermissions",
        "AdminReports",
        "AdminRulesPage",
        "AdminSettings",
        "AdminUserDetail",
        "AdminUsers",
        "AdminVanityRoles",
        "AnnouncementDetailPage",
        "AnnouncementsListPage",
        "ArtDetailPage",
        "ArtGalleryPage",
        "ChapterEditorPage",
        "ChatPage",
        "CheckersGamePage",
        "ChessGamePage",
        "CreateJournalPage",
        "CreateMysteryPage",
        "CreateOCPage",
        "CreateShipPage",
        "CreateTheoryPage",
        "EditJournalPage",
        "EditTheoryPage",
        "FanficChapterPage",
        "FanficDetailPage",
        "FanficEditorPage",
        "FanfictionListPage",
        "FeedPage",
        "ForgotPasswordPage",
        "GalleryDetailPage",
        "GameHubPage",
        "GamesListPage",
        "JournalEntryEditorPage",
        "JournalEntryPage",
        "JournalPage",
        "JournalsFeedPage",
        "LiveDirectoryPage",
        "LiveGamesPage",
        "LiveWatchPage",
        "LoginPage",
        "MinesweeperGamePage",
        "MysteryDetailPage",
        "MysteryListPage",
        "NewCheckersGamePage",
        "NewChessGamePage",
        "NewMinesweeperGamePage",
        "NewOthelloGamePage",
        "NewSnakesAndLaddersGamePage",
        "NotFoundPage",
        "NotificationsPage",
        "OCDetailPage",
        "OCListPage",
        "OthelloGamePage",
        "PastGamesPage",
        "PostDetailPage",
        "ProfilePage",
        "QuoteBrowserPage",
        "ResetPasswordPage",
        "RoomPage",
        "RoomsListPage",
        "RulesPage",
        "SearchPage",
        "SecretDetailPage",
        "SecretsListPage",
        "SetEmailPage",
        "SettingsPage",
        "ShipDetailPage",
        "ShipsListPage",
        "SnakesAndLaddersGamePage",
        "SocialFeedPage",
        "SuggestionsPage",
        "TheoryPage",
        "UsersPage",
        "VerifyEmailPage",
    ];

    const pages: Record<string, unknown> = {
        AdminLayout: () => (
            <div data-testid="page-AdminLayout">
                <Outlet />
            </div>
        ),
    };
    for (const name of names) {
        pages[name] = (props: Record<string, unknown>) => (
            <div data-testid={`page-${name}`} data-props={JSON.stringify(props)} />
        );
    }

    return pages;
});

vi.mock("./pages/landing/LandingPage", () => ({ LandingPage: () => <div data-testid="page-LandingPage" /> }));
vi.mock("./components/layout/Header/Header", () => ({
    Header: ({ onToggleSidebar }: { onToggleSidebar: () => void }) => (
        <button type="button" onClick={onToggleSidebar}>
            toggle sidebar
        </button>
    ),
}));
vi.mock("./components/layout/Sidebar/Sidebar", () => ({ Sidebar: () => null }));
vi.mock("./components/layout/Butterflies/Butterflies", () => ({ Butterflies: () => null }));
vi.mock("./components/CanonicalTag/CanonicalTag", () => ({ CanonicalTag: () => null }));
vi.mock("./components/StaleVersionBanner/StaleVersionBanner", () => ({ StaleVersionBanner: () => null }));
vi.mock("./components/NativeUpdateBanner/NativeUpdateBanner", () => ({ NativeUpdateBanner: () => null }));
vi.mock("./components/NativeLinkInterceptor/NativeLinkInterceptor", () => ({ NativeLinkInterceptor: () => null }));
vi.mock("./components/LastLocationTracker/LastLocationTracker", () => ({ LastLocationTracker: () => null }));
vi.mock("./components/LockBanner/LockBanner", () => ({ LockBanner: () => null }));
vi.mock("./components/VerifyEmailBanner/VerifyEmailBanner", () => ({ VerifyEmailBanner: () => null }));
vi.mock("./components/GameForfeitWarning/GameForfeitWarning", () => ({ GameForfeitWarning: () => null }));
vi.mock("./components/PullToRefresh/PullToRefresh", () => ({
    PullToRefresh: ({ children }: PropsWithChildren) => <>{children}</>,
}));
vi.mock("./utils/push", () => ({ initPush }));
vi.mock("./utils/notifications", () => ({ ensureNotificationPermission }));

const member = makeUser({ id: "user-1", username: "battler" });
const admin = makeUser({ id: "user-2", username: "beatrice", role: "admin" });

function renderApp(route: string, options: ProviderOptions = {}) {
    return renderWithProviders(<App />, { route, ...options });
}

beforeEach(() => {
    initPush.mockResolvedValue(undefined);
    ensureNotificationPermission.mockResolvedValue(undefined);
});

describe("App", () => {
    it("shows the landing page to a visitor who has expressed no preference", () => {
        // given
        const visitor = null;

        // when
        renderApp("/", { user: visitor });

        // then
        expect(screen.getByTestId("page-LandingPage")).toBeInTheDocument();
    });

    it("shows the landing page at the welcome route", () => {
        // given
        const visitor = null;

        // when
        renderApp("/welcome", { user: visitor });

        // then
        expect(screen.getByTestId("page-LandingPage")).toBeInTheDocument();
    });

    it("sends a member straight to the home page they chose", () => {
        // given
        const theorist = makeUser({ id: "user-3", private: { home_page: "theories" } });

        // when
        renderApp("/", { user: theorist });

        // then
        expect(screen.getByTestId("page-FeedPage")).toBeInTheDocument();
    });

    it("falls back to the landing page when the chosen home page is unknown", () => {
        // given
        const lost = makeUser({ id: "user-4", private: { home_page: "the golden land" } });

        // when
        renderApp("/", { user: lost });

        // then
        expect(screen.getByTestId("page-LandingPage")).toBeInTheDocument();
    });

    it("tells the theories feed which series it is showing", () => {
        // given
        const route = "/theories/ciconia";

        // when
        renderApp(route);

        // then
        expect(screen.getByTestId("page-FeedPage")).toHaveAttribute(
            "data-props",
            JSON.stringify({ series: "ciconia" }),
        );
    });

    it("tells the game board feed which corner it is showing", () => {
        // given
        const route = "/game-board/higurashi";

        // when
        renderApp(route);

        // then
        expect(screen.getByTestId("page-SocialFeedPage")).toHaveAttribute(
            "data-props",
            JSON.stringify({ corner: "higurashi" }),
        );
    });

    it("treats an unrecognised game board segment as a post id", () => {
        // given
        const route = "/game-board/019283";

        // when
        renderApp(route);

        // then
        expect(screen.getByTestId("page-PostDetailPage")).toBeInTheDocument();
    });

    it("shows the not found page for a route nobody claims", () => {
        // given
        const route = "/the-golden-land";

        // when
        renderApp(route);

        // then
        expect(screen.getByTestId("page-NotFoundPage")).toBeInTheDocument();
    });

    it("redirects the retired scoreboard link to the chess hub", () => {
        // given
        const route = "/games/chess/scoreboard";

        // when
        renderApp(route, { user: member });

        // then
        expect(screen.getByTestId("page-GameHubPage")).toBeInTheDocument();
    });

    it("sends a signed out visitor from a members only route to the login page", () => {
        // given
        const visitor = null;

        // when
        renderApp("/settings", { user: visitor });

        // then
        expect(screen.getByTestId("page-LoginPage")).toBeInTheDocument();
        expect(screen.queryByTestId("page-SettingsPage")).not.toBeInTheDocument();
    });

    it("opens a members only route for a signed in member", () => {
        // given
        const route = "/settings";

        // when
        renderApp(route, { user: member });

        // then
        expect(screen.getByTestId("page-SettingsPage")).toBeInTheDocument();
    });

    it("sends a member without admin rights back to the home page", () => {
        // given
        const route = "/admin";

        // when
        renderApp(route, { user: member });

        // then
        expect(screen.getByTestId("page-LandingPage")).toBeInTheDocument();
        expect(screen.queryByTestId("page-AdminDashboard")).not.toBeInTheDocument();
    });

    it("opens the admin dashboard inside the admin layout for an admin", () => {
        // given
        const route = "/admin";

        // when
        renderApp(route, { user: admin });

        // then
        expect(screen.getByTestId("page-AdminLayout")).toBeInTheDocument();
        expect(screen.getByTestId("page-AdminDashboard")).toBeInTheDocument();
    });

    it("opens a nested admin page for an admin", () => {
        // given
        const route = "/admin/users/user-7";

        // when
        renderApp(route, { user: admin });

        // then
        expect(screen.getByTestId("page-AdminUserDetail")).toBeInTheDocument();
    });

    it("renders nothing at all while the session is still being resolved", () => {
        // given
        const stillLoading = { loading: true };

        // when
        const { container } = renderApp("/welcome", { auth: stillLoading });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("shows the maintenance notice to a member while maintenance is on", () => {
        // given
        const maintenance = { maintenance_mode: true, maintenance_title: "Back shortly", maintenance_message: "Uu~" };

        // when
        renderApp("/quotes", { user: member, siteInfo: maintenance });

        // then
        expect(screen.getByRole("heading", { name: "Back shortly" })).toBeInTheDocument();
        expect(screen.queryByTestId("page-QuoteBrowserPage")).not.toBeInTheDocument();
    });

    it("lets an admin keep working while maintenance is on", () => {
        // given
        const maintenance = { maintenance_mode: true, maintenance_title: "Back shortly" };

        // when
        renderApp("/quotes", { user: admin, siteInfo: maintenance });

        // then
        expect(screen.getByTestId("page-QuoteBrowserPage")).toBeInTheDocument();
    });

    it("shows the announcement banner when the site has something to say", () => {
        // given
        const announcing = { announcement_banner: "[red]Beware[/red] the witch" };

        // when
        renderApp("/welcome", { siteInfo: announcing });

        // then
        expect(screen.getByText("Beware")).toBeInTheDocument();
        expect(screen.getByText(/the witch/)).toBeInTheDocument();
    });

    it("keeps the announcement banner away when there is nothing to say", () => {
        // given
        const quiet = { announcement_banner: "" };

        // when
        const { container } = renderApp("/welcome", { siteInfo: quiet });

        // then
        expect(container.querySelector(".announcement-banner")).toBeNull();
    });

    it("announces a secret that somebody else solved first", () => {
        // given
        renderApp("/welcome");

        // when
        act(() => {
            window.dispatchEvent(
                new CustomEvent("secret-closed", {
                    detail: {
                        secret_id: "secret-1",
                        secret_title: "The Golden Truth",
                        solver: { display_name: "Beatrice", username: "beato" },
                    },
                }),
            );
        });

        // then
        expect(screen.getByRole("status")).toHaveTextContent("Beatrice solved The Golden Truth");
        expect(screen.getByRole("link", { name: /Beatrice/ })).toHaveAttribute("href", "/secrets/secret-1");
    });

    it("ignores a secret closed event that names no solver", () => {
        // given
        renderApp("/welcome");

        // when
        act(() => {
            window.dispatchEvent(new CustomEvent("secret-closed", { detail: { secret_id: "secret-1" } }));
        });

        // then
        expect(screen.queryByRole("status")).not.toBeInTheDocument();
    });

    it("switches to the chat layout on a room route", () => {
        // given
        const route = "/rooms/room-1";

        // when
        const { container } = renderApp(route, { user: member });

        // then
        expect(container.querySelector("main")).toHaveClass("main-content-chat");
        expect(document.body.dataset.chatPage).toBe("true");
    });

    it("keeps the ordinary layout on an ordinary route", () => {
        // given
        const route = "/quotes";

        // when
        const { container } = renderApp(route, { user: member });

        // then
        expect(container.querySelector("main")).not.toHaveClass("main-content-chat");
        expect(document.body.dataset.chatPage).toBeUndefined();
    });

    it("asks for notification permission and wires up push once somebody is signed in", () => {
        // given
        const route = "/welcome";

        // when
        renderApp(route, { user: member });

        // then
        expect(ensureNotificationPermission).toHaveBeenCalledOnce();
        expect(initPush).toHaveBeenCalledOnce();
    });

    it("leaves notifications alone for a signed out visitor", () => {
        // given
        const visitor = null;

        // when
        renderApp("/welcome", { user: visitor });

        // then
        expect(ensureNotificationPermission).not.toHaveBeenCalled();
        expect(initPush).not.toHaveBeenCalled();
    });

    it("collapses the sidebar when the header toggle is used on a wide screen", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderApp("/welcome");

        // when
        await user.click(screen.getByRole("button", { name: "toggle sidebar" }));

        // then
        expect(container.querySelector(".app-layout")).toHaveAttribute("data-sidebar-collapsed", "true");
    });
});
