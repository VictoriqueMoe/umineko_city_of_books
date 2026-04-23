import { Suspense, useEffect, useState } from "react";
import { BrowserRouter, Link, Navigate, Route, Routes } from "react-router";
import { useSiteInfo } from "./hooks/useSiteInfo";
import { useTheme } from "./hooks/useTheme";
import { useAuth } from "./hooks/useAuth";
import { canAccessAdmin } from "./utils/permissions";
import { ensureNotificationPermission } from "./utils/notifications";
import { Header } from "./components/layout/Header/Header";
import { Sidebar } from "./components/layout/Sidebar/Sidebar";
import { Butterflies } from "./components/layout/Butterflies/Butterflies";
import { CanonicalTag } from "./components/CanonicalTag/CanonicalTag";
import { ProtectedRoute } from "./components/ProtectedRoute/ProtectedRoute";
import { StaleVersionBanner } from "./components/StaleVersionBanner/StaleVersionBanner";
import { Toast } from "./components/Toast/Toast";
import { MaintenancePage } from "./pages/maintenance/MaintenancePage";
import { LandingPage } from "./pages/landing/LandingPage";
import { linkify } from "./utils/linkify";
import {
    AdminAnnouncementsPage,
    AdminAuditLog,
    AdminBannedGifs,
    AdminBannedWords,
    AdminContentRules,
    AdminDashboard,
    AdminInvites,
    AdminLayout,
    AdminReports,
    AdminSettings,
    AdminUserDetail,
    AdminUsers,
    AdminVanityRoles,
    AnnouncementDetailPage,
    AnnouncementsListPage,
    ArtDetailPage,
    ArtGalleryPage,
    ChapterEditorPage,
    ChatPage,
    ChessGamePage,
    CreateJournalPage,
    CreateMysteryPage,
    CreateShipPage,
    CreateTheoryPage,
    EditJournalPage,
    EditTheoryPage,
    FanficChapterPage,
    FanficDetailPage,
    FanficEditorPage,
    FanfictionListPage,
    FeedPage,
    GalleryDetailPage,
    GameHubPage,
    GamesListPage,
    JournalPage,
    JournalsFeedPage,
    LiveGamesPage,
    LoginPage,
    MysteryDetailPage,
    MysteryListPage,
    NewChessGamePage,
    NotFoundPage,
    NotificationsPage,
    PastGamesPage,
    PostDetailPage,
    ProfilePage,
    QuoteBrowserPage,
    RoomPage,
    RoomsListPage,
    SecretDetailPage,
    SecretsListPage,
    SettingsPage,
    ShipDetailPage,
    ShipsListPage,
    SocialFeedPage,
    SuggestionsPage,
    TheoryPage,
    UsersPage,
} from "./pages/lazyPages";

const homePageRoutes: Record<string, string> = {
    landing: "/welcome",
    theories: "/theories",
    theories_higurashi: "/theories/higurashi",
    theories_ciconia: "/theories/ciconia",
    game_board: "/game-board",
    game_board_umineko: "/game-board/umineko",
    game_board_higurashi: "/game-board/higurashi",
    game_board_ciconia: "/game-board/ciconia",
    game_board_higanbana: "/game-board/higanbana",
    game_board_roseguns: "/game-board/roseguns",
    gallery: "/gallery",
    gallery_umineko: "/gallery/umineko",
    gallery_higurashi: "/gallery/higurashi",
    gallery_ciconia: "/gallery/ciconia",
    quotes: "/quotes",
    mysteries: "/mysteries",
    ships: "/ships",
    fanfiction: "/fanfiction",
    journals: "/journals",
    games: "/games",
};

function HomePage() {
    const { user } = useAuth();
    const target = homePageRoutes[user?.home_page ?? "landing"] ?? "/welcome";
    if (target === "/welcome") {
        return <LandingPage />;
    }
    return <Navigate to={target} replace />;
}

function AnnouncementBanner() {
    const siteInfo = useSiteInfo();
    const banner = siteInfo.announcement_banner ?? "";

    if (!banner) {
        return null;
    }

    return <div className="announcement-banner">{linkify(banner)}</div>;
}

function RouteFallback() {
    return <div className="loading">Loading...</div>;
}

interface SecretClosedDetail {
    secret_id: string;
    secret_title: string;
    solver: { display_name: string; username: string };
}

function SecretClosedToast() {
    const [event, setEvent] = useState<SecretClosedDetail | null>(null);

    useEffect(() => {
        function handler(e: Event) {
            const detail = (e as CustomEvent<SecretClosedDetail>).detail;
            if (detail && detail.secret_id && detail.solver) {
                setEvent(detail);
            }
        }
        window.addEventListener("secret-closed", handler);
        return () => window.removeEventListener("secret-closed", handler);
    }, []);

    if (!event) {
        return null;
    }
    const name = event.solver.display_name || event.solver.username;
    return (
        <Toast variant="arcane" duration={10000} onDismiss={() => setEvent(null)}>
            <Link to={`/secrets/${event.secret_id}`} style={{ color: "inherit" }}>
                Uu~ <strong>{name}</strong> solved <em>{event.secret_title}</em> before you could. Try again next time.
            </Link>
        </Toast>
    );
}

function AppLayout() {
    const siteInfo = useSiteInfo();
    const { particlesEnabled } = useTheme();
    const { user, loading: authLoading } = useAuth();
    const [sidebarOpen, setSidebarOpen] = useState(false);

    useEffect(() => {
        if (user) {
            ensureNotificationPermission().catch(() => {});
        }
    }, [user]);

    if (authLoading) {
        return null;
    }

    if (siteInfo.maintenance_mode && !canAccessAdmin(user?.role)) {
        return (
            <MaintenancePage title={siteInfo.maintenance_title ?? ""} message={siteInfo.maintenance_message ?? ""} />
        );
    }

    return (
        <div className="app-layout">
            <CanonicalTag />
            {particlesEnabled && <Butterflies />}
            <Sidebar open={sidebarOpen} onClose={() => setSidebarOpen(false)} />
            <div className="app-main">
                <Header onToggleSidebar={() => setSidebarOpen(prev => !prev)} />
                <StaleVersionBanner />
                <AnnouncementBanner />
                <SecretClosedToast />
                <main className="main-content">
                    <Suspense fallback={<RouteFallback />}>
                        <Routes>
                            <Route path="/" element={<HomePage />} />
                            <Route path="/welcome" element={<LandingPage />} />
                            <Route path="/theories" element={<FeedPage />} />
                            <Route path="/theories/higurashi" element={<FeedPage series="higurashi" />} />
                            <Route path="/theories/ciconia" element={<FeedPage series="ciconia" />} />
                            <Route path="/game-board" element={<SocialFeedPage />} />
                            <Route path="/game-board/umineko" element={<SocialFeedPage corner="umineko" />} />
                            <Route path="/game-board/higurashi" element={<SocialFeedPage corner="higurashi" />} />
                            <Route path="/game-board/ciconia" element={<SocialFeedPage corner="ciconia" />} />
                            <Route path="/game-board/higanbana" element={<SocialFeedPage corner="higanbana" />} />
                            <Route path="/game-board/roseguns" element={<SocialFeedPage corner="roseguns" />} />
                            <Route path="/game-board/:id" element={<PostDetailPage />} />
                            <Route path="/gallery" element={<ArtGalleryPage />} />
                            <Route path="/gallery/umineko" element={<ArtGalleryPage corner="umineko" />} />
                            <Route path="/gallery/higurashi" element={<ArtGalleryPage corner="higurashi" />} />
                            <Route path="/gallery/ciconia" element={<ArtGalleryPage corner="ciconia" />} />
                            <Route path="/gallery/art/:id" element={<ArtDetailPage />} />
                            <Route path="/gallery/view/:id" element={<GalleryDetailPage />} />
                            <Route path="/theory/:id" element={<TheoryPage />} />
                            <Route path="/announcements" element={<AnnouncementsListPage />} />
                            <Route path="/announcements/:id" element={<AnnouncementDetailPage />} />
                            <Route path="/suggestions" element={<SuggestionsPage />} />
                            <Route path="/suggestions/:id" element={<PostDetailPage />} />
                            <Route path="/mysteries" element={<MysteryListPage />} />
                            <Route path="/mystery/:id" element={<MysteryDetailPage />} />
                            <Route path="/ships" element={<ShipsListPage />} />
                            <Route path="/ships/:id" element={<ShipDetailPage />} />
                            <Route path="/fanfiction" element={<FanfictionListPage />} />
                            <Route path="/fanfiction/:id" element={<FanficDetailPage />} />
                            <Route path="/fanfiction/:id/chapter/:number" element={<FanficChapterPage />} />
                            <Route path="/journals" element={<JournalsFeedPage />} />
                            <Route path="/journals/:id" element={<JournalPage />} />
                            <Route path="/rooms" element={<RoomsListPage />} />
                            <Route path="/secrets" element={<SecretsListPage />} />
                            <Route path="/secrets/:id" element={<SecretDetailPage />} />
                            <Route path="/quotes" element={<QuoteBrowserPage />} />
                            <Route path="/games/chess/scoreboard" element={<Navigate to="/games/chess" replace />} />
                            <Route path="/games/live" element={<LiveGamesPage />} />
                            <Route path="/games/past" element={<PastGamesPage />} />
                            <Route path="/games/chess/:id" element={<ChessGamePage />} />
                            <Route path="/games/:type" element={<GameHubPage />} />
                            <Route path="/users" element={<UsersPage />} />
                            <Route path="/user/:username" element={<ProfilePage />} />
                            <Route path="/login" element={<LoginPage />} />

                            <Route element={<ProtectedRoute />}>
                                <Route path="/notifications" element={<NotificationsPage />} />
                                <Route path="/theory/new" element={<CreateTheoryPage />} />
                                <Route path="/theory/higurashi/new" element={<CreateTheoryPage series="higurashi" />} />
                                <Route path="/theory/ciconia/new" element={<CreateTheoryPage series="ciconia" />} />
                                <Route path="/mystery/new" element={<CreateMysteryPage />} />
                                <Route element={<ProtectedRoute permission="edit_any_theory" />}>
                                    <Route path="/mystery/:id/edit" element={<CreateMysteryPage />} />
                                </Route>
                                <Route path="/ships/new" element={<CreateShipPage />} />
                                <Route path="/fanfiction/new" element={<FanficEditorPage />} />
                                <Route path="/fanfiction/:id/edit" element={<FanficEditorPage />} />
                                <Route path="/fanfiction/:id/chapter/new" element={<ChapterEditorPage />} />
                                <Route path="/fanfiction/:id/chapter/:number/edit" element={<ChapterEditorPage />} />
                                <Route path="/journals/new" element={<CreateJournalPage />} />
                                <Route path="/journals/:id/edit" element={<EditJournalPage />} />
                                <Route path="/games" element={<GamesListPage />} />
                                <Route path="/games/chess/new" element={<NewChessGamePage />} />
                                <Route path="/rooms/:roomId" element={<RoomPage />} />
                                <Route path="/theory/:id/edit" element={<EditTheoryPage />} />
                                <Route path="/settings" element={<SettingsPage />} />
                                <Route path="/chat" element={<ChatPage />} />
                                <Route path="/chat/:roomId" element={<ChatPage />} />
                            </Route>

                            <Route element={<ProtectedRoute permission="view_admin_panel" />}>
                                <Route path="/admin" element={<AdminLayout />}>
                                    <Route index element={<AdminDashboard />} />
                                    <Route path="users" element={<AdminUsers />} />
                                    <Route path="users/:id" element={<AdminUserDetail />} />
                                    <Route path="invites" element={<AdminInvites />} />
                                    <Route path="settings" element={<AdminSettings />} />
                                    <Route path="reports" element={<AdminReports />} />
                                    <Route path="content-rules" element={<AdminContentRules />} />
                                    <Route path="banned-gifs" element={<AdminBannedGifs />} />
                                    <Route path="banned-words" element={<AdminBannedWords />} />
                                    <Route path="announcements" element={<AdminAnnouncementsPage />} />
                                    <Route path="audit-log" element={<AdminAuditLog />} />
                                    <Route path="vanity-roles" element={<AdminVanityRoles />} />
                                </Route>
                            </Route>

                            <Route path="*" element={<NotFoundPage />} />
                        </Routes>
                    </Suspense>
                </main>
            </div>
        </div>
    );
}

export default function App() {
    return (
        <BrowserRouter>
            <AppLayout />
        </BrowserRouter>
    );
}
