import {NavLink} from "react-router";
import {useAuth} from "../../hooks/useAuth";
import {ThemeSelector} from "./ThemeSelector";
import {NotificationBell} from "./NotificationBell";
import {LoginButton} from "../auth/LoginButton";
import {UserMenu} from "../auth/UserMenu";

export function Header() {
    const { user, loading } = useAuth();

    return (
        <header className="header">
            <div className="header-left">
                <NavLink to="/" className="header-title">
                    Umineko Game Board
                </NavLink>
                <span className="header-subtitle">Without love, it cannot be seen</span>
            </div>
            <div className="header-right">
                <NavLink to="/" className={({ isActive }) => `nav-btn${isActive ? " active" : ""}`} end>
                    Theories
                </NavLink>
                <NavLink to="/quotes" className={({ isActive }) => `nav-btn${isActive ? " active" : ""}`}>
                    Quotes
                </NavLink>
                {user && (
                    <NavLink to="/theory/new" className={({ isActive }) => `nav-btn${isActive ? " active" : ""}`}>
                        New Theory
                    </NavLink>
                )}
                {user && <NotificationBell />}
                <ThemeSelector />
                {!loading && (user ? <UserMenu /> : <LoginButton />)}
            </div>
        </header>
    );
}
