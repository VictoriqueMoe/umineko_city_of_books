import { useEffect } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { createJournal } from "../../api/endpoints";
import { JournalForm } from "../../components/journal/JournalForm/JournalForm";
import styles from "./CreateJournalPage.module.css";

export function CreateJournalPage() {
    usePageTitle("New Journal");
    const navigate = useNavigate();
    const { user, loading: authLoading } = useAuth();

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
                Document your read-through. After creating the journal, post updates by commenting on it. Your followers
                will be notified each time you do.
            </p>
            <JournalForm
                submitLabel="Create Journal"
                submittingLabel="Creating..."
                onSubmit={async data => {
                    const result = await createJournal(data);
                    navigate(`/journals/${result.id}`);
                }}
            />
        </div>
    );
}
