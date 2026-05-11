import { useEffect } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useCreateJournal } from "../../api/mutations/journal";
import { JournalForm } from "../../components/journal/JournalForm/JournalForm";
import styles from "./CreateJournalPage.module.css";

export function CreateJournalPage() {
    usePageTitle("New Journal");
    const navigate = useNavigate();
    const { user, loading: authLoading } = useAuth();
    const createMutation = useCreateJournal();

    useEffect(() => {
        if (!authLoading && !user) {
            navigate("/login");
        }
    }, [user, authLoading, navigate]);

    if (authLoading || !user) {
        return null;
    }

    return (
        <div className={styles.page}>
            <h2 className={styles.heading}>Start a Reading Journal</h2>
            <p className={styles.intro}>
                Set up the cover for your read-through. After creating the journal, you'll add your first entry. Each
                entry is its own page, and your followers get notified each time you post a new one.
            </p>
            <JournalForm
                submitLabel="Create Journal"
                submittingLabel="Creating..."
                onSubmit={async data => {
                    const result = await createMutation.mutateAsync(data);
                    navigate(`/journals/${result.id}/entry/new`);
                }}
            />
        </div>
    );
}
