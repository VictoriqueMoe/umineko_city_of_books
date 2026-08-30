import { useNavigate, useParams } from "react-router";
import { useJournal } from "../../hooks/queries/journal";
import { useUpdateJournal } from "../../hooks/mutations/journal";
import { useAuthedUser } from "../../hooks/useAuthedUser";
import { usePageTitle } from "../../hooks/usePageTitle";
import { contentPermissions } from "../../domain/contentPermissions";
import { JournalForm } from "../../components/journal/JournalForm/JournalForm";
import styles from "./CreateJournalPage.module.css";

export function EditJournalPage() {
    usePageTitle("Edit Journal");
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const user = useAuthedUser();
    const { journal, loading } = useJournal(id ?? "");
    const updateMutation = useUpdateJournal(id ?? "");

    if (loading) {
        return <div className="loading">Loading...</div>;
    }

    if (!journal) {
        return <div className="empty-state">Journal not found.</div>;
    }

    const { canEdit } = contentPermissions(user, { family: "journal", authorId: journal.author.id });
    if (!canEdit) {
        return <div className="empty-state">You can't edit this journal.</div>;
    }

    return (
        <div className={styles.page}>
            <h2 className={styles.heading}>Edit Journal</h2>
            <JournalForm
                initialTitle={journal.title}
                initialWork={journal.work}
                submitLabel="Save"
                submittingLabel="Saving..."
                onSubmit={async data => {
                    await updateMutation.mutateAsync(data);
                    navigate(`/journals/${journal.id}`);
                }}
            />
        </div>
    );
}
