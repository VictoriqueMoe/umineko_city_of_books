import {useCallback, useRef, useState} from "react";
import {useAuth} from "../../hooks/useAuth";
import {useClickOutside} from "../../hooks/useClickOutside";
import {useNavigate} from "react-router";

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
        <div className="user-menu" ref={dropdownRef}>
            <button className="user-menu-trigger" onClick={() => setIsOpen(!isOpen)}>
                {user.avatar_url ? (
                    <img className="profile-link-avatar" src={user.avatar_url} alt="" style={{ width: 24, height: 24 }} />
                ) : (
                    <span className="profile-link-avatar-placeholder" style={{ width: 24, height: 24, fontSize: 10 }}>
                        {user.display_name[0]}
                    </span>
                )}
                <span className="user-menu-name">{user.display_name}</span>
                <span className={`theme-chevron${isOpen ? " open" : ""}`}>{"\u25BC"}</span>
            </button>

            {isOpen && (
                <div className="user-menu-dropdown">
                    <button
                        className="user-menu-option"
                        onClick={() => {
                            setIsOpen(false);
                            navigate("/theory/new");
                        }}
                    >
                        New Theory
                    </button>
                    <button
                        className="user-menu-option"
                        onClick={() => {
                            setIsOpen(false);
                            navigate("/my-theories");
                        }}
                    >
                        My Theories
                    </button>
                    <button
                        className="user-menu-option"
                        onClick={() => {
                            setIsOpen(false);
                            navigate(`/user/${user.username}`);
                        }}
                    >
                        Profile
                    </button>
                    <button
                        className="user-menu-option"
                        onClick={() => {
                            setIsOpen(false);
                            navigate("/settings");
                        }}
                    >
                        Settings
                    </button>
                    <div className="theme-dropdown-divider" />
                    <button className="user-menu-option" onClick={handleLogout}>
                        Logout
                    </button>
                </div>
            )}
        </div>
    );
}
