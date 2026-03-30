import { useCallback, useRef, useState } from "react";
import { useAuth } from "../../../hooks/useAuth";
import { useClickOutside } from "../../../hooks/useClickOutside";
import { useNavigate } from "react-router";
import styles from "./UserMenu.module.css";

export function UserMenu() {
    const { user, logoutUser } = useAuth();
    const navigate = useNavigate();
    const [isOpen, setIsOpen] = useState(false);
    const dropdownRef = useRef<HTMLDivElement>(null);

    useClickOutside(
        dropdownRef,
        useCallback(() => setIsOpen(false), []),
    );

    if (!user) {
        return null;
    }

    async function handleLogout() {
        await logoutUser();
        navigate("/");
    }

    return (
        <div className={styles.menu} ref={dropdownRef}>
            <button className={styles.trigger} onClick={() => setIsOpen(!isOpen)}>
                {user.avatar_url ? (
                    <img className={styles.avatar} src={user.avatar_url} alt="" style={{ width: 24, height: 24 }} />
                ) : (
                    <span className={styles.avatarPlaceholder} style={{ width: 24, height: 24, fontSize: 10 }}>
                        {user.display_name[0]}
                    </span>
                )}
                <span className={styles.name}>{user.display_name}</span>
                <span className={`${styles.chevron}${isOpen ? ` ${styles.chevronOpen}` : ""}`}>{"\u25BC"}</span>
            </button>

            {isOpen && (
                <div className={styles.dropdown}>
                    <button className={styles.option} onClick={handleLogout}>
                        Logout
                    </button>
                </div>
            )}
        </div>
    );
}
