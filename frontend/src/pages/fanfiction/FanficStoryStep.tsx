import { Button } from "../../components/Button/Button";
import { ErrorBanner } from "../../components/ErrorBanner/ErrorBanner";
import { RichTextEditor } from "../../components/RichTextEditor/RichTextEditor";
import type { FanficStoryStep as FanficStoryStepState } from "../../hooks/useFanficEditor";
import styles from "./FanficPages.module.css";

interface FanficStoryStepProps {
    isEdit: boolean;
    isOneshot: boolean;
    error: string;
    submitting: boolean;
    story: FanficStoryStepState;
}

export function FanficStoryStep({ isEdit, isOneshot, error, submitting, story }: FanficStoryStepProps) {
    return (
        <div className={styles.formPage}>
            <span className={styles.back} onClick={story.back}>
                &larr; Back to Details
            </span>
            <h1 className={styles.formHeading}>
                {isEdit ? "Edit Story" : isOneshot ? "Write Your Story" : "Write First Chapter"}
            </h1>

            <RichTextEditor
                content={story.body}
                onChange={story.setBody}
                placeholder={isOneshot ? "Write your story here..." : "Write your first chapter here..."}
            />

            {error && <ErrorBanner message={error} />}

            <div className={styles.formActions} style={{ marginTop: "1rem" }}>
                <Button variant="ghost" onClick={story.back}>
                    Back
                </Button>
                {isEdit ? (
                    <Button variant="primary" onClick={story.saveChapter} disabled={submitting || !story.body.trim()}>
                        {submitting ? "Saving..." : "Save Changes"}
                    </Button>
                ) : (
                    <>
                        <Button variant="secondary" onClick={story.saveAsDraft} disabled={submitting}>
                            {submitting ? "Saving..." : "Save as Draft"}
                        </Button>
                        <Button variant="primary" onClick={story.publish} disabled={submitting}>
                            {submitting ? "Publishing..." : "Publish"}
                        </Button>
                    </>
                )}
            </div>
        </div>
    );
}
