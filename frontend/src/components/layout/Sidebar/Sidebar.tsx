import { useEffect, useRef, useState } from "react";
import { NavLink, useLocation } from "react-router";
import { useAuth } from "../../../hooks/useAuth";
import { useNotifications } from "../../../hooks/useNotifications";
import { getArtCornerCounts, getCornerCounts } from "../../../api/endpoints";
import { can, canAccessAdmin } from "../../../utils/permissions";
import { PieceTrigger } from "../../../features/easterEgg";
import styles from "./Sidebar.module.css";

interface SidebarProps {
    open: boolean;
    onClose: () => void;
}

const CORNERS = [
    { path: "/game-board", label: "General", key: "general" },
    { path: "/game-board/umineko", label: "Umineko", key: "umineko" },
    { path: "/game-board/higurashi", label: "Higurashi", key: "higurashi" },
    { path: "/game-board/ciconia", label: "Ciconia", key: "ciconia" },
];

const RYUKISHI_CORNERS = [
    { path: "/game-board/higanbana", label: "Higanbana", key: "higanbana" },
    { path: "/game-board/roseguns", label: "Rose Guns Days", key: "roseguns" },
];

const GALLERY_CORNERS = [
    { path: "/gallery", label: "General", key: "general" },
    { path: "/gallery/umineko", label: "Umineko", key: "umineko" },
    { path: "/gallery/higurashi", label: "Higurashi", key: "higurashi" },
    { path: "/gallery/ciconia", label: "Ciconia", key: "ciconia" },
];

const NEW_THEORY_LINKS = [
    { path: "/theory/new", label: "Umineko" },
    { path: "/theory/higurashi/new", label: "Higurashi" },
    { path: "/theory/ciconia/new", label: "Ciconia" },
];

const GAMES_LINKS = [
    { path: "/games/live", label: "Live Games", authRequired: false },
    { path: "/games/past", label: "Past Games", authRequired: false },
    { path: "/games", label: "My Games", authRequired: true },
];

export function Sidebar({ open, onClose }: SidebarProps) {
    const { user } = useAuth();
    const {
        addWSListener,
        unreadCount: unreadNotifs,
        chatUnreadCount: unreadChat,
        liveGamesCount,
    } = useNotifications();
    const location = useLocation();
    const [newAnnouncement, setNewAnnouncement] = useState(false);
    const isRyukishiPath = RYUKISHI_CORNERS.some(c => location.pathname === c.path);
    const [cornersOpen, setCornersOpen] = useState(location.pathname.startsWith("/game-board") && !isRyukishiPath);
    const [ryukishiOpen, setRyukishiOpen] = useState(isRyukishiPath);
    const [galleryOpen, setGalleryOpen] = useState(location.pathname.startsWith("/gallery"));
    const [theoriesOpen, setTheoriesOpen] = useState(location.pathname.startsWith("/theor"));
    const isNewTheoryPath = NEW_THEORY_LINKS.some(l => location.pathname === l.path);
    const [newTheoryOpen, setNewTheoryOpen] = useState(isNewTheoryPath);
    const [gamesOpen, setGamesOpen] = useState(location.pathname.startsWith("/games"));
    const [cornerCounts, setCornerCounts] = useState<Record<string, number>>({});
    const [artCounts, setArtCounts] = useState<Record<string, number>>({});
    const pathnameRef = useRef(location.pathname);

    useEffect(() => {
        pathnameRef.current = location.pathname;
    }, [location.pathname]);

    const [prevPath, setPrevPath] = useState(location.pathname);
    if (prevPath !== location.pathname) {
        setPrevPath(location.pathname);
        if (location.pathname.startsWith("/game-board") && !isRyukishiPath) {
            setCornersOpen(true);
        }
        if (isRyukishiPath) {
            setRyukishiOpen(true);
        }
        if (location.pathname.startsWith("/gallery")) {
            setGalleryOpen(true);
        }
        if (location.pathname.startsWith("/theor")) {
            setTheoriesOpen(true);
        }
        if (isNewTheoryPath) {
            setNewTheoryOpen(true);
        }
        if (location.pathname.startsWith("/games")) {
            setGamesOpen(true);
        }
    }

    useEffect(() => {
        getCornerCounts()
            .then(setCornerCounts)
            .catch(() => {});
        getArtCornerCounts()
            .then(setArtCounts)
            .catch(() => {});
    }, []);

    useEffect(() => {
        return addWSListener(msg => {
            if (msg.type === "new_announcement") {
                const data = msg.data as { author_id?: string };
                if (data.author_id !== user?.id && !pathnameRef.current.startsWith("/announcement")) {
                    setNewAnnouncement(true);
                }
            }
        });
    }, [addWSListener, user?.id]);

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
                    <NavLink
                        to="/welcome"
                        className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                        onClick={onClose}
                    >
                        Welcome
                    </NavLink>
                    <div className={styles.divider} />
                    <NavLink
                        to="/announcements"
                        className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                        onClick={() => {
                            setNewAnnouncement(false);
                            onClose();
                        }}
                    >
                        Announcements
                        {newAnnouncement && <span className={styles.newBadge}>New</span>}
                    </NavLink>
                    <NavLink
                        to="/suggestions"
                        className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                        onClick={onClose}
                    >
                        Site Improvements
                    </NavLink>
                    <div className={styles.section}>
                        <span className={styles.sectionLabel}>Browse</span>
                        <button
                            className={`${styles.link} ${styles.expandBtn}${cornersOpen ? ` ${styles.expandOpen}` : ""}`}
                            onClick={() => setCornersOpen(prev => !prev)}
                        >
                            Game Board
                            <span className={styles.expandIcon}>{cornersOpen ? "\u25B4" : "\u25BE"}</span>
                        </button>
                        {cornersOpen && (
                            <div className={styles.subLinks}>
                                {CORNERS.map(c => (
                                    <NavLink
                                        key={c.path}
                                        to={c.path}
                                        end
                                        className={({ isActive }) =>
                                            `${styles.link} ${styles.subLink}${isActive ? ` ${styles.active}` : ""}`
                                        }
                                        onClick={onClose}
                                    >
                                        {c.label}
                                        <span className={styles.cornerCount}>{cornerCounts[c.key] ?? 0}</span>
                                    </NavLink>
                                ))}
                            </div>
                        )}
                        <button
                            className={`${styles.link} ${styles.expandBtn}${ryukishiOpen ? ` ${styles.expandOpen}` : ""}`}
                            onClick={() => setRyukishiOpen(prev => !prev)}
                        >
                            Ryukishi's Other Works
                            <span className={styles.expandIcon}>{ryukishiOpen ? "\u25B4" : "\u25BE"}</span>
                        </button>
                        {ryukishiOpen && (
                            <div className={styles.subLinks}>
                                {RYUKISHI_CORNERS.map(c => (
                                    <NavLink
                                        key={c.path}
                                        to={c.path}
                                        end
                                        className={({ isActive }) =>
                                            `${styles.link} ${styles.subLink}${isActive ? ` ${styles.active}` : ""}`
                                        }
                                        onClick={onClose}
                                    >
                                        {c.label}
                                        <span className={styles.cornerCount}>{cornerCounts[c.key] ?? 0}</span>
                                    </NavLink>
                                ))}
                            </div>
                        )}
                        <button
                            className={`${styles.link} ${styles.expandBtn}${galleryOpen ? ` ${styles.expandOpen}` : ""}`}
                            onClick={() => setGalleryOpen(prev => !prev)}
                        >
                            Gallery
                            <span className={styles.expandIcon}>{galleryOpen ? "\u25B4" : "\u25BE"}</span>
                        </button>
                        {galleryOpen && (
                            <div className={styles.subLinks}>
                                {GALLERY_CORNERS.map(c => (
                                    <NavLink
                                        key={c.path}
                                        to={c.path}
                                        end
                                        className={({ isActive }) =>
                                            `${styles.link} ${styles.subLink}${isActive ? ` ${styles.active}` : ""}`
                                        }
                                        onClick={onClose}
                                    >
                                        {c.label}
                                        <span className={styles.cornerCount}>{artCounts[c.key] ?? 0}</span>
                                    </NavLink>
                                ))}
                            </div>
                        )}
                        <button
                            className={`${styles.link} ${styles.expandBtn}${theoriesOpen ? ` ${styles.expandOpen}` : ""}`}
                            onClick={() => setTheoriesOpen(prev => !prev)}
                        >
                            Theories
                            <span className={styles.expandIcon}>{theoriesOpen ? "\u25B4" : "\u25BE"}</span>
                        </button>
                        {theoriesOpen && (
                            <div className={styles.subLinks}>
                                <NavLink
                                    to="/theories"
                                    end
                                    className={({ isActive }) =>
                                        `${styles.link} ${styles.subLink}${isActive ? ` ${styles.active}` : ""}`
                                    }
                                    onClick={onClose}
                                >
                                    Umineko
                                </NavLink>
                                <NavLink
                                    to="/theories/higurashi"
                                    end
                                    className={({ isActive }) =>
                                        `${styles.link} ${styles.subLink}${isActive ? ` ${styles.active}` : ""}`
                                    }
                                    onClick={onClose}
                                >
                                    Higurashi
                                </NavLink>
                                <NavLink
                                    to="/theories/ciconia"
                                    end
                                    className={({ isActive }) =>
                                        `${styles.link} ${styles.subLink}${isActive ? ` ${styles.active}` : ""}`
                                    }
                                    onClick={onClose}
                                >
                                    Ciconia
                                </NavLink>
                            </div>
                        )}
                        <NavLink
                            to="/mysteries"
                            className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                            onClick={onClose}
                        >
                            Mysteries
                        </NavLink>
                        <NavLink
                            to="/secrets"
                            className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                            onClick={onClose}
                        >
                            Secrets
                        </NavLink>
                        <NavLink
                            to="/ships"
                            className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                            onClick={onClose}
                        >
                            Ships
                        </NavLink>
                        <NavLink
                            to="/fanfiction"
                            className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                            onClick={onClose}
                        >
                            Fanfictions
                        </NavLink>
                        <NavLink
                            to="/journals"
                            className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                            onClick={onClose}
                        >
                            Reading Journals
                        </NavLink>
                        <NavLink
                            to="/rooms"
                            className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                            onClick={onClose}
                        >
                            Chat Rooms
                        </NavLink>
                        <button
                            className={`${styles.link} ${styles.expandBtn}${gamesOpen ? ` ${styles.expandOpen}` : ""}`}
                            onClick={() => setGamesOpen(prev => !prev)}
                        >
                            Games
                            <span className={styles.expandIcon}>{gamesOpen ? "\u25B4" : "\u25BE"}</span>
                        </button>
                        {gamesOpen && (
                            <div className={styles.subLinks}>
                                {GAMES_LINKS.filter(l => !l.authRequired || user).map(l => (
                                    <NavLink
                                        key={l.path}
                                        to={l.path}
                                        end
                                        className={({ isActive }) =>
                                            `${styles.link} ${styles.subLink}${isActive ? ` ${styles.active}` : ""}`
                                        }
                                        onClick={onClose}
                                    >
                                        {l.label}
                                        {l.path === "/games/live" && liveGamesCount > 0 && (
                                            <span className={styles.chatBadge}>
                                                {liveGamesCount > 99 ? "99+" : liveGamesCount}
                                            </span>
                                        )}
                                    </NavLink>
                                ))}
                            </div>
                        )}
                        <NavLink
                            to="/users"
                            className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                            onClick={onClose}
                        >
                            Players
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
                            <button
                                className={`${styles.link} ${styles.expandBtn}${newTheoryOpen ? ` ${styles.expandOpen}` : ""}`}
                                onClick={() => setNewTheoryOpen(prev => !prev)}
                            >
                                New Theory
                                <span className={styles.expandIcon}>{newTheoryOpen ? "\u25B4" : "\u25BE"}</span>
                            </button>
                            {newTheoryOpen && (
                                <div className={styles.subLinks}>
                                    {NEW_THEORY_LINKS.map(l => (
                                        <NavLink
                                            key={l.path}
                                            to={l.path}
                                            end
                                            className={({ isActive }) =>
                                                `${styles.link} ${styles.subLink}${isActive ? ` ${styles.active}` : ""}`
                                            }
                                            onClick={onClose}
                                        >
                                            {l.label}
                                        </NavLink>
                                    ))}
                                </div>
                            )}
                            <NavLink
                                to="/mystery/new"
                                className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                                onClick={onClose}
                            >
                                New Mystery
                            </NavLink>
                            <NavLink
                                to="/ships/new"
                                className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                                onClick={onClose}
                            >
                                New Ship
                            </NavLink>
                            <NavLink
                                to="/fanfiction/new"
                                className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                                onClick={onClose}
                            >
                                New Fanfic
                            </NavLink>
                            <NavLink
                                to="/journals/new"
                                className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                                onClick={onClose}
                            >
                                New Journal
                            </NavLink>
                        </div>
                    )}

                    {user && (
                        <div className={styles.section}>
                            <span className={styles.sectionLabel}>Account</span>
                            <NavLink
                                to="/notifications"
                                className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                                onClick={onClose}
                            >
                                Notifications
                                {unreadNotifs > 0 && (
                                    <span className={styles.chatBadge}>{unreadNotifs > 99 ? "99+" : unreadNotifs}</span>
                                )}
                            </NavLink>
                            <NavLink
                                to={`/user/${user.username}`}
                                className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                                onClick={onClose}
                            >
                                Profile
                            </NavLink>
                            <NavLink
                                to="/chat"
                                className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                                onClick={onClose}
                            >
                                Chat
                                {unreadChat > 0 && (
                                    <span className={styles.chatBadge}>{unreadChat > 99 ? "99+" : unreadChat}</span>
                                )}
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
                            <span className={styles.sectionLabel}>
                                {can(user?.role, "manage_settings") ? "Admin" : "Moderation"}
                            </span>
                            <NavLink
                                to="/admin"
                                className={({ isActive }) => `${styles.link}${isActive ? ` ${styles.active}` : ""}`}
                                onClick={onClose}
                            >
                                {can(user?.role, "manage_settings") ? "Admin Panel" : "Moderator Panel"}
                            </NavLink>
                        </div>
                    )}
                </nav>

                <div className={styles.footer}>
                    <div className={styles.footerOrnament}>{"\u2666 \u2663 \u2665 \u2660"}</div>
                    <p>Without love, it cannot be seen.</p>
                    <p>
                        {"Umineko no Naku Koro ni \u00A9 "}
                        <a href="https://07th-expansion.net/" target="_blank" rel="noopener">
                            07th Expansion
                        </a>
                    </p>
                    <p>
                        {"Made with \u2764 by "}
                        <a href="https://x.com/FeatherineFAA" target="_blank" rel="noopener">
                            Featherine Augustus Aurora
                        </a>{" "}
                        <PieceTrigger pieceId="piece_02" />
                    </p>
                    <div className={styles.footerLinks}>
                        <a
                            href="https://github.com/VictoriqueMoe/umineko_city_of_books"
                            target="_blank"
                            rel="noopener"
                            className={styles.footerLink}
                        >
                            <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
                                <path
                                    fill="currentColor"
                                    d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"
                                />
                            </svg>
                            Source
                        </a>
                    </div>
                    <p>
                        {"Support 07th Expansion - "}
                        <a href="https://store.steampowered.com/app/406550/" target="_blank" rel="noopener">
                            get the game on Steam
                        </a>
                    </p>
                </div>
            </aside>
        </>
    );
}
