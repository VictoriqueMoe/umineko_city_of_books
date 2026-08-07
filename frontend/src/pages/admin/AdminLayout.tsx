import { NavLink, Outlet } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { can } from "../../utils/permissions";
import styles from "./AdminLayout.module.css";

export function AdminLayout() {
    const { user } = useAuth();

    return (
        <div className={styles.layout}>
            <div className={styles.header}>
                <h2 className={styles.title}>{can(user, "manage_settings") ? "Administration" : "Moderator Panel"}</h2>
                <nav className={styles.tabs}>
                    <NavLink
                        to="/admin"
                        end
                        className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                    >
                        Dashboard
                    </NavLink>
                    {can(user, "view_users") && (
                        <NavLink
                            to="/admin/users"
                            className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                        >
                            Users
                        </NavLink>
                    )}
                    {can(user, "view_users") && (
                        <NavLink
                            to="/admin/reports"
                            className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                        >
                            Reports
                        </NavLink>
                    )}
                    {can(user, "manage_roles") && (
                        <NavLink
                            to="/admin/invites"
                            className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                        >
                            Invites
                        </NavLink>
                    )}
                    {can(user, "manage_settings") && (
                        <NavLink
                            to="/admin/content-rules"
                            className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                        >
                            Content Rules
                        </NavLink>
                    )}
                    {can(user, "manage_settings") && (
                        <NavLink
                            to="/admin/rules"
                            className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                        >
                            Rules Page
                        </NavLink>
                    )}
                    {can(user, "manage_settings") && (
                        <NavLink
                            to="/admin/banned-gifs"
                            className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                        >
                            Banned GIFs
                        </NavLink>
                    )}
                    {can(user, "manage_banned_words") && (
                        <NavLink
                            to="/admin/banned-words"
                            className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                        >
                            Banned Words
                        </NavLink>
                    )}
                    {can(user, "manage_settings") && (
                        <NavLink
                            to="/admin/announcements"
                            className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                        >
                            Announcements
                        </NavLink>
                    )}
                    {can(user, "manage_settings") && (
                        <NavLink
                            to="/admin/settings"
                            className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                        >
                            Settings
                        </NavLink>
                    )}
                    {can(user, "manage_vanity_roles") && (
                        <NavLink
                            to="/admin/vanity-roles"
                            className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                        >
                            Vanity Roles
                        </NavLink>
                    )}
                    {can(user, "manage_roles") && (
                        <NavLink
                            to="/admin/permissions"
                            className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                        >
                            Permissions
                        </NavLink>
                    )}
                    {can(user, "manage_settings") && (
                        <NavLink
                            to="/admin/chatbots"
                            className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                        >
                            Chatbots
                        </NavLink>
                    )}
                    {can(user, "view_audit_log") && (
                        <NavLink
                            to="/admin/audit-log"
                            className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                        >
                            Audit Log
                        </NavLink>
                    )}
                </nav>
            </div>
            <div className={styles.content}>
                <Outlet />
            </div>
        </div>
    );
}
