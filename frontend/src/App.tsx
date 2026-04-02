import { useState } from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router";
import { useSiteInfo } from "./hooks/useSiteInfo";
import { useTheme } from "./hooks/useTheme";
import { useAuth } from "./hooks/useAuth";
import { canAccessAdmin } from "./utils/permissions";
import { Header } from "./components/layout/Header/Header";
import { Sidebar } from "./components/layout/Sidebar/Sidebar";
import { Butterflies } from "./components/layout/Butterflies/Butterflies";
import { ProtectedRoute } from "./components/ProtectedRoute/ProtectedRoute";
import { FeedPage } from "./pages/theories/FeedPage";
import { TheoryPage } from "./pages/theories/TheoryPage";
import { CreateTheoryPage } from "./pages/theories/CreateTheoryPage";
import { LoginPage } from "./pages/auth/LoginPage";
import { QuoteBrowserPage } from "./pages/quotes/QuoteBrowserPage";
import { MyTheoriesPage } from "./pages/theories/MyTheoriesPage";
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
import { SocialFeedPage } from "./pages/feed/SocialFeedPage";
import { PostDetailPage } from "./pages/feed/PostDetailPage";
import { UsersPage } from "./pages/users/UsersPage";
import { ChatPage } from "./pages/chat/ChatPage";
import { ArtGalleryPage } from "./pages/gallery/ArtGalleryPage";
import { ArtDetailPage } from "./pages/gallery/ArtDetailPage";
import { GalleryDetailPage } from "./pages/gallery/GalleryDetailPage";
import { MaintenancePage } from "./pages/maintenance/MaintenancePage";

const homePageRoutes: Record<string, string> = {
    theories: "/theories",
    game_board: "/game-board",
};

function HomePage() {
    const { user } = useAuth();
    const target = homePageRoutes[user?.home_page ?? "theories"] ?? "/theories";
    return <Navigate to={target} replace />;
}

function AnnouncementBanner() {
    const siteInfo = useSiteInfo();
    const banner = siteInfo.announcement_banner ?? "";

    if (!banner) {
        return null;
    }

    return (
        <div
            style={{
                background: "linear-gradient(90deg, var(--gold-dark), var(--gold), var(--gold-dark))",
                color: "var(--bg-void)",
                padding: "0.5rem 1rem",
                textAlign: "center",
                fontWeight: 600,
                fontSize: "0.95rem",
                width: "100%",
            }}
        >
            {banner}
        </div>
    );
}

function AppLayout() {
    const siteInfo = useSiteInfo();
    const { particlesEnabled } = useTheme();
    const { user, loading: authLoading } = useAuth();
    const [sidebarOpen, setSidebarOpen] = useState(false);

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
                        <Route path="/game-board" element={<SocialFeedPage />} />
                        <Route path="/game-board/umineko" element={<SocialFeedPage corner="umineko" />} />
                        <Route path="/game-board/higurashi" element={<SocialFeedPage corner="higurashi" />} />
                        <Route path="/game-board/ciconia" element={<SocialFeedPage corner="ciconia" />} />
                        <Route path="/game-board/:id" element={<PostDetailPage />} />
                        <Route path="/gallery" element={<ArtGalleryPage />} />
                        <Route path="/gallery/umineko" element={<ArtGalleryPage corner="umineko" />} />
                        <Route path="/gallery/higurashi" element={<ArtGalleryPage corner="higurashi" />} />
                        <Route path="/gallery/ciconia" element={<ArtGalleryPage corner="ciconia" />} />
                        <Route path="/gallery/art/:id" element={<ArtDetailPage />} />
                        <Route path="/gallery/view/:id" element={<GalleryDetailPage />} />
                        <Route path="/theory/:id" element={<TheoryPage />} />
                        <Route path="/quotes" element={<QuoteBrowserPage />} />
                        <Route path="/users" element={<UsersPage />} />
                        <Route path="/user/:username" element={<ProfilePage />} />
                        <Route path="/login" element={<LoginPage />} />

                        <Route element={<ProtectedRoute />}>
                            <Route path="/theory/new" element={<CreateTheoryPage />} />
                            <Route path="/theory/:id/edit" element={<EditTheoryPage />} />
                            <Route path="/my-theories" element={<MyTheoriesPage />} />
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
                                <Route path="audit-log" element={<AdminAuditLog />} />
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
