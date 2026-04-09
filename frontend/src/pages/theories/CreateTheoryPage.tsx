import { useEffect } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { createTheory, type Series } from "../../api/endpoints";
import { TheoryForm } from "../../components/theory/TheoryForm/TheoryForm";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import formStyles from "../../components/theory/TheoryForm/TheoryForm.module.css";

export function CreateTheoryPage({ series = "umineko" }: { series?: Series }) {
    usePageTitle("New Theory");
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
        <div className={formStyles.page}>
            <h2 className={formStyles.heading}>Declare Your Blue Truth</h2>
            <RulesBox page={series === "higurashi" ? "theories_higurashi" : "theories"} />

            <TheoryForm
                submitLabel="Declare Blue Truth"
                submittingLabel="Declaring..."
                series={series}
                onSubmit={async data => {
                    const result = await createTheory({ ...data, series });
                    navigate(`/theory/${result.id}`);
                }}
            />
        </div>
    );
}
