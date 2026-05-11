import { marked } from "marked";
import DOMPurify from "dompurify";
import { useSiteInfo } from "../../hooks/useSiteInfo";
import { usePageTitle } from "../../hooks/usePageTitle";
import { NotFoundPage } from "../notfound/NotFoundPage";
import styles from "../announcements/AnnouncementsPage.module.css";

function renderMarkdown(md: string): string {
    const raw = marked.parse(md, { async: false }) as string;
    return DOMPurify.sanitize(raw);
}

export function RulesPage() {
    const siteInfo = useSiteInfo();
    const body = siteInfo.rules_page ?? "";
    usePageTitle("Rules");

    if (!body.trim()) {
        return <NotFoundPage />;
    }

    return (
        <div className={styles.page}>
            <div className={styles.detail}>
                <h1 className={styles.detailTitle}>Rules</h1>
                <div className={styles.body} dangerouslySetInnerHTML={{ __html: renderMarkdown(body) }} />
            </div>
        </div>
    );
}
