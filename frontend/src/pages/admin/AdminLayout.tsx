import {NavLink, Outlet} from "react-router";
import {useAuth} from "../../hooks/useAuth";
import {can} from "../../utils/permissions";
import styles from "./AdminLayout.module.css";

export function AdminLayout() {
    const { user } = useAuth();

    return (
        <div className={styles.layout}>
            <div className={styles.header}>
                <h2 className={styles.title}>Administration</h2>
                <nav className={styles.tabs}>
                    <NavLink
                        to="/admin"
                        end
                        className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                    >
                        Dashboard
                    </NavLink>
                    {can(user?.role, "manage_roles") && (
                        <NavLink
                            to="/admin/users"
                            className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                        >
                            Users
                        </NavLink>
                    )}
                    {can(user?.role, "manage_settings") && (
                        <NavLink
                            to="/admin/settings"
                            className={({ isActive }) => `${styles.tab}${isActive ? ` ${styles.tabActive}` : ""}`}
                        >
                            Settings
                        </NavLink>
                    )}
                    {can(user?.role, "view_audit_log") && (
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
