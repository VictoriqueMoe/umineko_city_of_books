import { useEffect, useState } from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router";
import { useSiteInfo } from "./hooks/useSiteInfo";
import { useTheme } from "./hooks/useTheme";
import { useAuth } from "./hooks/useAuth";
import { canAccessAdmin } from "./utils/permissions";
import { ensureNotificationPermission } from "./utils/notifications";
import { Header } from "./components/layout/Header/Header";
import { Sidebar } from "./components/layout/Sidebar/Sidebar";
import { Butterflies } from "./components/layout/Butterflies/Butterflies";
import { ProtectedRoute } from "./components/ProtectedRoute/ProtectedRoute";
import { FeedPage } from "./pages/theories/FeedPage";
import { TheoryPage } from "./pages/theories/TheoryPage";
import { CreateTheoryPage } from "./pages/theories/CreateTheoryPage";
import { LoginPage } from "./pages/auth/LoginPage";
import { QuoteBrowserPage } from "./pages/quotes/QuoteBrowserPage";
import { EditTheoryPage } from "./pages/theories/EditTheoryPage";
import { ProfilePage } from "./pages/profile/ProfilePage";
import { SettingsPage } from "./pages/profile/SettingsPage";
import { AdminLayout } from "./pages/admin/AdminLayout";
import { AdminDashboard } from "./pages/admin/AdminDashboard";
import { AdminUsers } from "./pages/admin/AdminUsers";
import { AdminUserDetail } from "./pages/admin/AdminUserDetail";
import { AdminSettings } from "./pages/admin/AdminSettings";
import { AdminAuditLog } from "./pages/admin/AdminAuditLog";
import { AdminInvites } from "./pages/admin/AdminInvites";
import { AdminReports } from "./pages/admin/AdminReports";
import { AdminContentRules } from "./pages/admin/AdminContentRules";
import { AdminVanityRoles } from "./pages/admin/AdminVanityRoles";
import { AdminAnnouncements as AdminAnnouncementsPage } from "./pages/admin/AdminAnnouncements";
import { AnnouncementsPage as AnnouncementsListPage } from "./pages/announcements/AnnouncementsPage";
import { linkify } from "./utils/linkify";
import { AnnouncementDetailPage } from "./pages/announcements/AnnouncementDetailPage";
import { MysteryListPage } from "./pages/mysteries/MysteryListPage";
import { MysteryDetailPage } from "./pages/mysteries/MysteryDetailPage";
import { CreateMysteryPage } from "./pages/mysteries/CreateMysteryPage";
import { ShipsListPage } from "./pages/ships/ShipsListPage";
import { FanfictionListPage } from "./pages/fanfiction/FanfictionListPage";
import { FanficDetailPage } from "./pages/fanfiction/FanficDetailPage";
import { FanficChapterPage } from "./pages/fanfiction/FanficChapterPage";
import { FanficEditorPage } from "./pages/fanfiction/FanficEditorPage";
import { ChapterEditorPage } from "./pages/fanfiction/ChapterEditorPage";
import { ShipDetailPage } from "./pages/ships/ShipDetailPage";
import { CreateShipPage } from "./pages/ships/CreateShipPage";
import { SuggestionsPage } from "./pages/suggestions/SuggestionsPage";
import { SocialFeedPage } from "./pages/feed/SocialFeedPage";
import { PostDetailPage } from "./pages/feed/PostDetailPage";
import { UsersPage } from "./pages/users/UsersPage";
import { ChatPage } from "./pages/chat/ChatPage";
import { ArtGalleryPage } from "./pages/gallery/ArtGalleryPage";
import { ArtDetailPage } from "./pages/gallery/ArtDetailPage";
import { GalleryDetailPage } from "./pages/gallery/GalleryDetailPage";
import { MaintenancePage } from "./pages/maintenance/MaintenancePage";
import { NotificationsPage } from "./pages/notifications/NotificationsPage";
import { JournalsFeedPage } from "./pages/journals/FeedPage";
import { JournalPage } from "./pages/journals/JournalPage";
import { CreateJournalPage } from "./pages/journals/CreateJournalPage";
import { EditJournalPage } from "./pages/journals/EditJournalPage";
import { RoomsListPage } from "./pages/rooms/RoomsListPage";
import { RoomPage } from "./pages/rooms/RoomPage";

const homePageRoutes: Record<string, string> = {
    theories: "/theories",
    theories_higurashi: "/theories/higurashi",
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
};

function HomePage() {
    const { user } = useAuth();
    const target = homePageRoutes[user?.home_page ?? "game_board"] ?? "/game-board";
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
            {particlesEnabled && <Butterflies />}
            <Sidebar open={sidebarOpen} onClose={() => setSidebarOpen(false)} />
            <div className="app-main">
                <Header onToggleSidebar={() => setSidebarOpen(prev => !prev)} />
                <AnnouncementBanner />
                <main className="main-content">
                    <Routes>
                        <Route path="/" element={<HomePage />} />
                        <Route path="/theories" element={<FeedPage />} />
                        <Route path="/theories/higurashi" element={<FeedPage series="higurashi" />} />
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
                        <Route path="/quotes" element={<QuoteBrowserPage />} />
                        <Route path="/users" element={<UsersPage />} />
                        <Route path="/user/:username" element={<ProfilePage />} />
                        <Route path="/login" element={<LoginPage />} />

                        <Route element={<ProtectedRoute />}>
                            <Route path="/notifications" element={<NotificationsPage />} />
                            <Route path="/theory/new" element={<CreateTheoryPage />} />
                            <Route path="/theory/higurashi/new" element={<CreateTheoryPage series="higurashi" />} />
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
                                <Route path="announcements" element={<AdminAnnouncementsPage />} />
                                <Route path="audit-log" element={<AdminAuditLog />} />
                                <Route path="vanity-roles" element={<AdminVanityRoles />} />
                            </Route>
                        </Route>
                    </Routes>
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
