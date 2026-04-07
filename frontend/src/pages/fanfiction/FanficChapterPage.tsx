import { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import type { FanficChapter, FanficDetail } from "../../types/api";
import { getFanfic, getFanficChapter } from "../../api/endpoints";
import { Button } from "../../components/Button/Button";
import styles from "./FanficPages.module.css";

function formatNumber(n: number): string {
    if (n >= 1_000_000) {
        return (n / 1_000_000).toFixed(1) + "M";
    }
    if (n >= 1_000) {
        return (n / 1_000).toFixed(1) + "K";
    }
    return n.toLocaleString();
}

export function FanficChapterPage() {
    const { id: fanficId, number: numParam } = useParams<{ id: string; number: string }>();
    const navigate = useNavigate();
    const [chapter, setChapter] = useState<FanficChapter | null>(null);
    const [fanfic, setFanfic] = useState<FanficDetail | null>(null);
    const [loading, setLoading] = useState(true);

    const chapterNumber = Number(numParam);

    const fetchChapter = useCallback(() => {
        if (!fanficId || !numParam || isNaN(chapterNumber)) {
            return;
        }
        Promise.all([getFanficChapter(fanficId, chapterNumber), getFanfic(fanficId)])
            .then(([ch, f]) => {
                setChapter(ch);
                setFanfic(f);
            })
            .catch(() => {
                setChapter(null);
                setFanfic(null);
            })
            .finally(() => setLoading(false));
    }, [fanficId, numParam, chapterNumber]);

    useEffect(() => {
        fetchChapter();
    }, [fetchChapter]);

    if (loading) {
        return <div className="loading">Loading...</div>;
    }

    if (!chapter || !fanfic) {
        return <div className="empty-state">Chapter not found.</div>;
    }

    const isOneshot = fanfic.is_oneshot;
    const title = isOneshot ? fanfic.title : `Chapter ${chapter.chapter_number}: ${chapter.title}`;

    function navButtons() {
        return (
            <div className={styles.chapterNav}>
                <Button
                    variant="secondary"
                    size="small"
                    disabled={!chapter!.has_prev}
                    onClick={() => navigate(`/fanfiction/${fanficId}/chapter/${chapterNumber - 1}`)}
                >
                    &larr; Previous
                </Button>
                <Button
                    variant="secondary"
                    size="small"
                    disabled={!chapter!.has_next}
                    onClick={() => navigate(`/fanfiction/${fanficId}/chapter/${chapterNumber + 1}`)}
                >
                    Next &rarr;
                </Button>
            </div>
        );
    }

    return (
        <div className={styles.chapterPage}>
            <span className={styles.back} onClick={() => navigate(`/fanfiction/${fanficId}`)}>
                &larr; Back to {fanfic.title}
            </span>

            <h1 className={styles.chapterTitle}>{title}</h1>

            {!isOneshot && navButtons()}

            <div className={styles.chapterBody} dangerouslySetInnerHTML={{ __html: chapter.body }} />

            <p className={styles.cardStats} style={{ marginTop: "1.5rem" }}>
                {formatNumber(chapter.word_count)} words
            </p>

            {!isOneshot && navButtons()}
        </div>
    );
}
