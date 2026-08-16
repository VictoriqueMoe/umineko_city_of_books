import { useRef, useState, type ReactNode } from "react";
import { useNavigate } from "react-router";
import { useAuthedUser } from "../../hooks/useAuthedUser";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useMutualFollowers, useSearchUsers } from "../../api/queries/misc";
import { useInviteToGame } from "../../api/mutations/gameRoom";
import type { GameType, User } from "../../types/api";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import styles from "./GamesPages.module.css";

const DEFAULT_BLURB = "Pick an opponent to invite. They'll get a notification and the game starts when they accept.";

interface NewGameInvitePageProps {
    gameName: string;
    gameType: GameType;
    blurb?: ReactNode;
}

export function NewGameInvitePage({ gameName, gameType, blurb = DEFAULT_BLURB }: NewGameInvitePageProps) {
    const title = `New ${gameName} Game`;

    usePageTitle(title);
    const user = useAuthedUser();
    const navigate = useNavigate();
    const [search, setSearch] = useState("");
    const [debouncedSearch, setDebouncedSearch] = useState("");
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
    const [selected, setSelected] = useState<User | null>(null);
    const [error, setError] = useState("");
    const inviteMutation = useInviteToGame();

    function handleSearchChange(value: string) {
        setSearch(value);
        clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => setDebouncedSearch(value.trim()), 200);
    }

    const { mutuals } = useMutualFollowers();
    const { users: results } = useSearchUsers(debouncedSearch);

    const rawCandidates = search.trim() ? results : mutuals;
    const candidates = rawCandidates.filter(u => u.id !== user.id);

    async function handleInvite() {
        if (!selected || inviteMutation.isPending) {
            return;
        }
        setError("");
        try {
            const room = await inviteMutation.mutateAsync({ opponentId: selected.id, gameType });
            navigate(`/games/${gameType}/${room.id}`);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to invite");
        }
    }

    return (
        <div className={styles.page}>
            <h2 className={styles.heading}>{title}</h2>
            <p>{blurb}</p>

            <div className={styles.inviteForm}>
                <Input
                    placeholder="Search for a player by username..."
                    value={search}
                    onChange={e => handleSearchChange(e.target.value)}
                />

                {error && <div className={styles.error}>{error}</div>}

                <div className={styles.userList}>
                    {candidates.length === 0 && <p className={styles.empty}>No matches.</p>}
                    {candidates.map(u => (
                        <div
                            key={u.id}
                            className={`${styles.userRow} ${selected?.id === u.id ? styles.userRowSelected : ""}`}
                            onClick={() => setSelected(u)}
                        >
                            <span>{u.display_name}</span>
                            <span className={styles.subline}>@{u.username}</span>
                        </div>
                    ))}
                </div>

                <div className={styles.actions}>
                    <Button variant="ghost" onClick={() => navigate("/games")}>
                        Cancel
                    </Button>
                    <Button variant="primary" onClick={handleInvite} disabled={!selected || inviteMutation.isPending}>
                        {inviteMutation.isPending
                            ? "Sending..."
                            : selected
                              ? `Invite ${selected.display_name}`
                              : "Pick a player"}
                    </Button>
                </div>
            </div>
        </div>
    );
}
