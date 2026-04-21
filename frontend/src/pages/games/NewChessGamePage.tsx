import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import * as api from "../../api/endpoints";
import type { User } from "../../types/api";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import styles from "./GamesPages.module.css";

export function NewChessGamePage() {
    usePageTitle("New Chess Game");
    const { user, loading: authLoading } = useAuth();
    const navigate = useNavigate();
    const [search, setSearch] = useState("");
    const [results, setResults] = useState<User[]>([]);
    const [mutuals, setMutuals] = useState<User[]>([]);
    const [selected, setSelected] = useState<User | null>(null);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

    useEffect(() => {
        if (!authLoading && !user) {
            navigate("/login");
        }
    }, [user, authLoading, navigate]);

    useEffect(() => {
        api.getMutualFollowers()
            .then(setMutuals)
            .catch(() => setMutuals([]));
    }, []);

    useEffect(() => {
        clearTimeout(debounceRef.current);
        if (!search.trim()) {
            setResults([]);
            return;
        }
        debounceRef.current = setTimeout(() => {
            api.searchUsers(search)
                .then(setResults)
                .catch(() => setResults([]));
        }, 200);
        return () => clearTimeout(debounceRef.current);
    }, [search]);

    if (authLoading || !user) {
        return null;
    }

    const rawCandidates = search.trim() ? results : mutuals;
    const candidates = rawCandidates.filter(u => u.id !== user.id);

    async function handleInvite() {
        if (!selected || submitting) {
            return;
        }
        setSubmitting(true);
        setError("");
        try {
            const room = await api.inviteToGame(selected.id, "chess");
            navigate(`/games/chess/${room.id}`);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to invite");
        } finally {
            setSubmitting(false);
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
                    onChange={e => setSearch(e.target.value)}
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
                    <Button variant="primary" onClick={handleInvite} disabled={!selected || submitting}>
                        {submitting ? "Sending..." : selected ? `Invite ${selected.display_name}` : "Pick a player"}
                    </Button>
                </div>
            </div>
        </div>
    );
}
