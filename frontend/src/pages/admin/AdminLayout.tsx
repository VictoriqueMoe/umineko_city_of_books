import { NavLink, Outlet } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { can, type Permission } from "../../domain/permissions";
import { staffPanelLabel } from "../../domain/adminTargets";
import styles from "./AdminLayout.module.css";

interface AdminTab {
    to: string;
    label: string;
    permission?: Permission;
    end?: boolean;
}

const ADMIN_TABS: AdminTab[] = [
    { to: "/admin", label: "Dashboard", end: true },
    { to: "/admin/users", label: "Users", permission: "view_users" },
    { to: "/admin/reports", label: "Reports", permission: "view_users" },
    { to: "/admin/invites", label: "Invites", permission: "manage_roles" },
    { to: "/admin/content-rules", label: "Content Rules", permission: "manage_settings" },
    { to: "/admin/rules", label: "Rules Page", permission: "manage_settings" },
    { to: "/admin/banned-gifs", label: "Banned GIFs", permission: "manage_settings" },
    { to: "/admin/banned-words", label: "Banned Words", permission: "manage_banned_words" },
    { to: "/admin/announcements", label: "Announcements", permission: "manage_settings" },
    { to: "/admin/settings", label: "Settings", permission: "manage_settings" },
    { to: "/admin/vanity-roles", label: "Vanity Roles", permission: "manage_vanity_roles" },
    { to: "/admin/permissions", label: "Permissions", permission: "manage_roles" },
    { to: "/admin/chatbots", label: "Chatbots", permission: "manage_settings" },
    { to: "/admin/audit-log", label: "Audit Log", permission: "view_audit_log" },
];

export function AdminLayout() {
    const { user } = useAuth();
    const staffPanel = staffPanelLabel(user);
    const tabs = ADMIN_TABS.filter(tab => !tab.permission || can(user, tab.permission));

    return (
        <div className={styles.layout}>
            <div className={styles.header}>
                <h2 className={styles.title}>{staffPanel.heading}</h2>
                <nav className={styles.tabs}>
                    {tabs.map(tab => (
                        <NavLink
                            key={tab.to}
                            to={tab.to}
                            end={tab.end}
                            className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                        >
                            {tab.label}
                        </NavLink>
                    ))}
                </nav>
            </div>
            <div className={styles.content}>
                <Outlet />
            </div>
        </div>
    );
}
