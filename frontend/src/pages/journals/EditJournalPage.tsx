import { useEffect } from "react";
import { useNavigate, useParams } from "react-router";
import { useJournal } from "../../api/queries/journal";
import { useUpdateJournal } from "../../api/mutations/journal";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { can } from "../../utils/permissions";
import { JournalForm } from "../../components/journal/JournalForm/JournalForm";
import styles from "./CreateJournalPage.module.css";

export function EditJournalPage() {
    usePageTitle("Edit Journal");
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { user, loading: authLoading } = useAuth();
    const { journal, loading } = useJournal(id ?? "");
    const updateMutation = useUpdateJournal(id ?? "");

    useEffect(() => {
        if (!authLoading && !user) {
            navigate("/login");
        }
    }, [user, authLoading, navigate]);

    if (authLoading || loading) {
        return <div className="loading">Loading...</div>;
    }

    if (!journal || !user) {
        return <div className="empty-state">Journal not found.</div>;
    }

    const isOwner = user.id === journal.author.id;
    if (!isOwner && !can(user.role, "edit_any_journal")) {
        return <div className="empty-state">You can't edit this journal.</div>;
    }

    return (
        <div className={styles.page}>
            <h2 className={styles.heading}>Edit Journal</h2>
            <JournalForm
                initialTitle={journal.title}
                initialBody={journal.body}
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
