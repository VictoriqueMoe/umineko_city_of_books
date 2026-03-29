import {useEffect} from "react";
import {useNavigate} from "react-router";
import {useAuth} from "../../hooks/useAuth";
import {useTheoryFeed} from "../../hooks/useTheoryFeed";
import {Button} from "../../components/Button/Button";
import {TheoryCard} from "../../components/theory/TheoryCard/TheoryCard";
import {Pagination} from "../../components/Pagination/Pagination";
import styles from "./MyTheoriesPage.module.css";

export function MyTheoriesPage() {
    const navigate = useNavigate();
    const { user, loading: authLoading } = useAuth();
    const { theories, total, loading, offset, limit, goNext, goPrev, hasNext, hasPrev } = useTheoryFeed(
        "new",
        0,
        user?.id,
    );

    useEffect(() => {
        if (!authLoading && !user) {
            navigate("/login");
        }
    }, [user, authLoading, navigate]);

    if (!user) {
        return null;
    }

    return (
        <div>
            <h2 className={styles.heading}>My Theories</h2>

            {loading && <div className="loading">Consulting the game board...</div>}

            {!loading && theories.length === 0 && (
                <div className="empty-state">
                    You haven't declared any theories yet.
                    <br />
                    <Button variant="primary" onClick={() => navigate("/theory/new")}>
                        Declare Your First Theory
                    </Button>
                </div>
            )}

            {!loading && theories.map(theory => <TheoryCard key={theory.id} theory={theory} />)}

            {!loading && (
                <Pagination
                    offset={offset}
                    limit={limit}
                    total={total}
                    hasNext={hasNext}
                    hasPrev={hasPrev}
                    onNext={goNext}
                    onPrev={goPrev}
                />
            )}
        </div>
    );
}
