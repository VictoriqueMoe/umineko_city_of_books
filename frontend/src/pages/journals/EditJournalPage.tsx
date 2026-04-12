import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import type { JournalDetail } from "../../types/api";
import { getJournal, updateJournal } from "../../api/endpoints";
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
    const [journal, setJournal] = useState<JournalDetail | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        if (!id) {
            return;
        }
        getJournal(id)
            .then(setJournal)
            .catch(() => setJournal(null))
            .finally(() => setLoading(false));
    }, [id]);

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
                    await updateJournal(journal.id, data);
                    navigate(`/journals/${journal.id}`);
                }}
            />
        </div>
    );
}
