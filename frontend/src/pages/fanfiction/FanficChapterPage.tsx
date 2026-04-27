import { useMemo } from "react";
import { useNavigate, useParams } from "react-router";
import DOMPurify from "dompurify";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useFanfic, useFanficChapter } from "../../api/queries/fanfic";
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
    const chapterNumber = Number(numParam);
    const { chapter, loading: chapterLoading } = useFanficChapter(fanficId ?? "", chapterNumber);
    const { fanfic, loading: fanficLoading } = useFanfic(fanficId ?? "");
    const loading = chapterLoading || fanficLoading;
    usePageTitle(chapter?.title || (chapter ? `Chapter ${chapter.chapter_number}` : "Chapter"));

    const safeBody = useMemo(() => DOMPurify.sanitize(chapter?.body ?? ""), [chapter?.body]);

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
                    onClick={() => {
                        navigate(`/fanfiction/${fanficId}/chapter/${chapterNumber - 1}`);
                        window.scrollTo({ top: 0, behavior: "smooth" });
                    }}
                >
                    &larr; Previous
                </Button>
                <Button
                    variant="secondary"
                    size="small"
                    disabled={!chapter!.has_next}
                    onClick={() => {
                        navigate(`/fanfiction/${fanficId}/chapter/${chapterNumber + 1}`);
                        window.scrollTo({ top: 0, behavior: "smooth" });
                    }}
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

            <div className={styles.chapterBody} dangerouslySetInnerHTML={{ __html: safeBody }} />

            <p className={styles.cardStats} style={{ marginTop: "1.5rem" }}>
                {formatNumber(chapter.word_count)} words
            </p>

            {!isOneshot && navButtons()}
        </div>
    );
}
