import { Button } from "../../components/Button/Button";
import { useFanficEditor } from "../../hooks/useFanficEditor";
import { usePageTitle } from "../../hooks/usePageTitle";
import { FanficDetailsStep } from "./FanficDetailsStep";
import { FanficStoryStep } from "./FanficStoryStep";
import styles from "./FanficPages.module.css";

export function FanficEditorPage() {
    const editor = useFanficEditor();
    usePageTitle(editor.isEdit ? "Edit Fanfic" : "New Fanfic");

    if (editor.loading) {
        return <div className="loading">Loading...</div>;
    }

    if (editor.notFound) {
        return <div className="empty-state">Fanfic not found.</div>;
    }

    if (editor.draftPrompt) {
        return (
            <div className={styles.formPage}>
                <h1 className={styles.formHeading}>Unfinished Draft</h1>
                <p style={{ color: "var(--text)", marginBottom: "1rem" }}>
                    You have an unfinished draft:{" "}
                    <strong>
                        <bdi>{editor.draftPrompt.title}</bdi>
                    </strong>
                </p>
                <div className={styles.formActions}>
                    <Button variant="ghost" onClick={editor.startFresh}>
                        Start Fresh
                    </Button>
                    <Button variant="primary" onClick={editor.resumeDraft}>
                        Continue Draft
                    </Button>
                </div>
            </div>
        );
    }

    if (editor.hidden) {
        return null;
    }

    if (editor.step === 1) {
        return (
            <FanficDetailsStep
                isEdit={editor.isEdit}
                error={editor.error}
                submitting={editor.submitting}
                details={editor.details}
                onBack={editor.back}
                onCancel={editor.cancel}
            />
        );
    }

    return (
        <FanficStoryStep
            isEdit={editor.isEdit}
            isOneshot={editor.details.fields.isOneshot}
            error={editor.error}
            submitting={editor.submitting}
            story={editor.story}
        />
    );
}
