import { NavLink, Outlet } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { adminTabsFor, staffPanelLabel } from "../../domain/adminTargets";
import styles from "./AdminLayout.module.css";

export function AdminLayout() {
    const { user } = useAuth();
    const staffPanel = staffPanelLabel(user);
    const tabs = adminTabsFor(user);

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
