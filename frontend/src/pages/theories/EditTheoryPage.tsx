import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import type { Series } from "../../api/endpoints";
import { getTheory, updateTheory } from "../../api/endpoints";
import { Button } from "../../components/Button/Button";
import { TheoryForm } from "../../components/theory/TheoryForm/TheoryForm";
import type { EvidenceItem } from "../../types/api";
import formStyles from "../../components/theory/TheoryForm/TheoryForm.module.css";

export function EditTheoryPage() {
    usePageTitle("Edit Theory");
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { user, loading: authLoading } = useAuth();
    const theoryId = id ?? "";
    const [initialData, setInitialData] = useState<{
        title: string;
        body: string;
        episode: number;
        series: string;
        evidence: EvidenceItem[];
    } | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        if (!theoryId) {
            return;
        }
        getTheory(theoryId)
            .then(theory => {
                setInitialData({
                    title: theory.title,
                    body: theory.body,
                    episode: theory.episode,
                    series: theory.series || "umineko",
                    evidence: theory.evidence ?? [],
                });
                setLoading(false);
            })
            .catch(() => {
                navigate("/");
            });
    }, [theoryId, navigate]);

    useEffect(() => {
        if (!authLoading && !user) {
            navigate("/login");
        }
    }, [user, authLoading, navigate]);

    if (authLoading || !user) {
        return null;
    }

    if (loading || !initialData) {
        return <div className="loading">Loading theory...</div>;
    }

    return (
        <div className={formStyles.page}>
            <Button variant="ghost" onClick={() => navigate(-1)}>
                &larr; Cancel
            </Button>
            <h2 className={formStyles.heading}>Edit Theory</h2>

            <TheoryForm
                initialTitle={initialData.title}
                initialBody={initialData.body}
                initialEpisode={initialData.episode}
                initialEvidence={initialData.evidence}
                submitLabel="Save Changes"
                submittingLabel="Saving..."
                series={(initialData.series || "umineko") as Series}
                onSubmit={async data => {
                    await updateTheory(theoryId, { ...data, series: initialData.series || "umineko" });
                    navigate(`/theory/${theoryId}`);
                }}
            />
        </div>
    );
}
