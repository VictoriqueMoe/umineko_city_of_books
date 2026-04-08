import { useAuth } from "../../../hooks/useAuth";
import { ThemeSelector } from "../ThemeSelector/ThemeSelector";
import { NotificationBell } from "../NotificationBell/NotificationBell";
import { LoginButton } from "../../auth/LoginButton/LoginButton";
import { UserMenu } from "../../auth/UserMenu/UserMenu";
import styles from "./Header.module.css";

interface HeaderProps {
    onToggleSidebar: () => void;
}

export function Header({ onToggleSidebar }: HeaderProps) {
    const { user, loading } = useAuth();

    return (
        <header className={styles.header}>
            <button className={styles.hamburger} onClick={onToggleSidebar} aria-label="Toggle menu">
                <span className={styles.hamburgerLine} />
                <span className={styles.hamburgerLine} />
                <span className={styles.hamburgerLine} />
            </button>

            <div className={styles.actions}>
                {!loading && user && <NotificationBell />}
                <ThemeSelector />
                {!loading && (user ? <UserMenu /> : <LoginButton />)}
            </div>
        </header>
    );
}
