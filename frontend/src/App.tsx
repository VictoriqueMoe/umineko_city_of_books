import {useEffect, useState} from "react";
import {BrowserRouter, Route, Routes} from "react-router";
import {getSiteInfo} from "./api/endpoints";
import {useTheme} from "./hooks/useTheme";
import {useAuth} from "./hooks/useAuth";
import {canAccessAdmin} from "./utils/permissions";
import {Header} from "./components/layout/Header/Header";
import {Sidebar} from "./components/layout/Sidebar/Sidebar";
import {Butterflies} from "./components/layout/Butterflies/Butterflies";
import {ProtectedRoute} from "./components/ProtectedRoute/ProtectedRoute";
import {FeedPage} from "./pages/theories/FeedPage";
import {TheoryPage} from "./pages/theories/TheoryPage";
import {CreateTheoryPage} from "./pages/theories/CreateTheoryPage";
import {LoginPage} from "./pages/auth/LoginPage";
import {QuoteBrowserPage} from "./pages/quotes/QuoteBrowserPage";
import {MyTheoriesPage} from "./pages/theories/MyTheoriesPage";
import {EditTheoryPage} from "./pages/theories/EditTheoryPage";
import {ProfilePage} from "./pages/profile/ProfilePage";
import {SettingsPage} from "./pages/profile/SettingsPage";
import {AdminLayout} from "./pages/admin/AdminLayout";
import {AdminDashboard} from "./pages/admin/AdminDashboard";
import {AdminUsers} from "./pages/admin/AdminUsers";
import {AdminUserDetail} from "./pages/admin/AdminUserDetail";
import {AdminSettings} from "./pages/admin/AdminSettings";
import {AdminAuditLog} from "./pages/admin/AdminAuditLog";
import {AdminInvites} from "./pages/admin/AdminInvites";
import {MaintenancePage} from "./pages/maintenance/MaintenancePage";

function AnnouncementBanner() {
    const [banner, setBanner] = useState("");

    useEffect(() => {
        getSiteInfo()
            .then(info => setBanner(info.announcement_banner ?? ""))
            .catch(() => {});
    }, []);

    if (!banner) {
        return null;
    }

    return (
        <div style={{
            background: "linear-gradient(90deg, var(--gold-dark), var(--gold), var(--gold-dark))",
            color: "var(--bg-void)",
            padding: "0.5rem 1rem",
            textAlign: "center",
            fontWeight: 600,
            fontSize: "0.95rem",
            width: "100%",
        }}>
            {banner}
        </div>
    );
}

function AppLayout() {
    const { particlesEnabled } = useTheme();
    const { user, loading: authLoading } = useAuth();
    const [sidebarOpen, setSidebarOpen] = useState(false);
    const [maintenance, setMaintenance] = useState(false);
    const [siteInfoLoaded, setSiteInfoLoaded] = useState(false);

    useEffect(() => {
        getSiteInfo()
            .then(info => setMaintenance(info.maintenance_mode))
            .catch(() => {})
            .finally(() => setSiteInfoLoaded(true));
    }, []);

    if (!siteInfoLoaded || authLoading) {
        return null;
    }

    if (maintenance && !canAccessAdmin(user?.role)) {
        return <MaintenancePage />;
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
                        <Route path="/" element={<FeedPage />} />
                        <Route path="/theory/:id" element={<TheoryPage />} />
                        <Route path="/quotes" element={<QuoteBrowserPage />} />
                        <Route path="/user/:username" element={<ProfilePage />} />
                        <Route path="/login" element={<LoginPage />} />

                        <Route element={<ProtectedRoute />}>
                            <Route path="/theory/new" element={<CreateTheoryPage />} />
                            <Route path="/theory/:id/edit" element={<EditTheoryPage />} />
                            <Route path="/my-theories" element={<MyTheoriesPage />} />
                            <Route path="/settings" element={<SettingsPage />} />
                        </Route>

                        <Route element={<ProtectedRoute permission="view_admin_panel" />}>
                            <Route path="/admin" element={<AdminLayout />}>
                                <Route index element={<AdminDashboard />} />
                                <Route path="users" element={<AdminUsers />} />
                                <Route path="users/:id" element={<AdminUserDetail />} />
                                <Route path="invites" element={<AdminInvites />} />
                                <Route path="settings" element={<AdminSettings />} />
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
