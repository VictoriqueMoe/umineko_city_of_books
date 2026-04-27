import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useMutualFollowers, useSearchUsers } from "../../api/queries/misc";
import { useInviteToGame } from "../../api/mutations/gameRoom";
import type { User } from "../../types/api";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import styles from "./GamesPages.module.css";

export function NewChessGamePage() {
    usePageTitle("New Chess Game");
    const { user, loading: authLoading } = useAuth();
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

    useEffect(() => {
        if (!authLoading && !user) {
            navigate("/login");
        }
    }, [user, authLoading, navigate]);

    const { mutuals } = useMutualFollowers(!!user);
    const { users: results } = useSearchUsers(debouncedSearch, !!user);

    if (authLoading || !user) {
        return null;
    }

    const rawCandidates = search.trim() ? results : mutuals;
    const candidates = rawCandidates.filter(u => u.id !== user.id);

    async function handleInvite() {
        if (!selected || inviteMutation.isPending) {
            return;
        }
        setError("");
        try {
            const room = await inviteMutation.mutateAsync({ opponentId: selected.id, gameType: "chess" });
            navigate(`/games/chess/${room.id}`);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to invite");
        }
    }

    return (
        <div className={styles.page}>
            <h2 className={styles.heading}>New Chess Game</h2>
            <p>Pick an opponent to invite. They'll get a notification and the game starts when they accept.</p>

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
