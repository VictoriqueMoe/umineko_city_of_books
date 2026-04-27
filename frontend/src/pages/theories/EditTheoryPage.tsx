import { useEffect } from "react";
import { useNavigate, useParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import type { Series } from "../../api/endpoints";
import { useTheory } from "../../api/queries/theory";
import { useUpdateTheory } from "../../api/mutations/theory";
import { Button } from "../../components/Button/Button";
import { TheoryForm } from "../../components/theory/TheoryForm/TheoryForm";
import formStyles from "../../components/theory/TheoryForm/TheoryForm.module.css";

export function EditTheoryPage() {
    usePageTitle("Edit Theory");
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { user, loading: authLoading } = useAuth();
    const theoryId = id ?? "";
    const { theory, loading: theoryLoading } = useTheory(theoryId);
    const updateMutation = useUpdateTheory(theoryId);

    useEffect(() => {
        if (!theoryLoading && !theory && theoryId) {
            navigate("/");
        }
    }, [theoryLoading, theory, theoryId, navigate]);

    useEffect(() => {
        if (!authLoading && !user) {
            navigate("/login");
        }
    }, [user, authLoading, navigate]);

    if (authLoading || !user) {
        return null;
    }

    if (theoryLoading || !theory) {
        return <div className="loading">Loading theory...</div>;
    }

    const series = (theory.series || "umineko") as Series;

    return (
        <div className={formStyles.page}>
            <Button variant="ghost" onClick={() => navigate(-1)}>
                &larr; Cancel
            </Button>
            <h2 className={formStyles.heading}>Edit Theory</h2>

            <TheoryForm
                key={theory.id}
                initialTitle={theory.title}
                initialBody={theory.body}
                initialEpisode={theory.episode}
                initialEvidence={theory.evidence ?? []}
                submitLabel="Save Changes"
                submittingLabel="Saving..."
                series={series}
                onSubmit={async data => {
                    await updateMutation.mutateAsync({ ...data, series });
                    navigate(`/theory/${theoryId}`);
                }}
            />
        </div>
    );
}
