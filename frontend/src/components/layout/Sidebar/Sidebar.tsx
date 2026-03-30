import {NavLink} from "react-router";
import {useAuth} from "../../../hooks/useAuth";
import {canAccessAdmin} from "../../../utils/permissions";
import styles from "./Sidebar.module.css";

interface SidebarProps {
    open: boolean;
    onClose: () => void;
}

export function Sidebar({ open, onClose }: SidebarProps) {
    const { user } = useAuth();

    return (
        <>
            {open && <div className={styles.overlay} onClick={onClose} />}
            <aside className={`${styles.sidebar} ${open ? styles.open : ""}`}>
                <div className={styles.brand}>
                    <NavLink to="/" className={styles.title} onClick={onClose}>
                        Umineko Game Board
                    </NavLink>
                    <span className={styles.subtitle}>Without love, it cannot be seen</span>
                </div>

                <nav className={styles.nav}>
                    <div className={styles.section}>
                        <span className={styles.sectionLabel}>Browse</span>
                        <NavLink
                            to="/"
                            end
                            className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                            onClick={onClose}
                        >
                            Theories
                        </NavLink>
                        <NavLink
                            to="/quotes"
                            className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                            onClick={onClose}
                        >
                            Quotes
                        </NavLink>
                    </div>

                    {user && (
                        <div className={styles.section}>
                            <span className={styles.sectionLabel}>Create</span>
                            <NavLink
                                to="/theory/new"
                                className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                                onClick={onClose}
                            >
                                New Theory
                            </NavLink>
                        </div>
                    )}

                    {user && (
                        <div className={styles.section}>
                            <span className={styles.sectionLabel}>Account</span>
                            <NavLink
                                to="/my-theories"
                                className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                                onClick={onClose}
                            >
                                My Theories
                            </NavLink>
                            <NavLink
                                to={`/user/${user.username}`}
                                className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                                onClick={onClose}
                            >
                                Profile
                            </NavLink>
                            <NavLink
                                to="/settings"
                                className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                                onClick={onClose}
                            >
                                Settings
                            </NavLink>
                        </div>
                    )}

                    {canAccessAdmin(user?.role) && (
                        <div className={styles.section}>
                            <span className={styles.sectionLabel}>Admin</span>
                            <NavLink
                                to="/admin"
                                className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                                onClick={onClose}
                            >
                                Admin Panel
                            </NavLink>
                        </div>
                    )}
                </nav>

                <div className={styles.footer}>Umineko no Naku Koro ni - 07th Expansion</div>
            </aside>
        </>
    );
}
