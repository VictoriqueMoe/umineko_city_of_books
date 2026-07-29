import { useNavigate } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import type { Series } from "../../api/endpoints";
import { useCreateTheory } from "../../api/mutations/theory";
import { TheoryForm } from "../../components/theory/TheoryForm/TheoryForm";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import { getSeriesConfig } from "../../utils/seriesConfig";
import formStyles from "../../components/theory/TheoryForm/TheoryForm.module.css";

export function CreateTheoryPage({ series = "umineko" }: { series?: Series }) {
    usePageTitle("New Theory");
    const navigate = useNavigate();
    const createMutation = useCreateTheory();

    return (
        <div className={formStyles.page}>
            <h2 className={formStyles.heading}>Declare Your Blue Truth</h2>
            <RulesBox page={getSeriesConfig(series).theoriesRulesPage} />

            <TheoryForm
                submitLabel="Declare Blue Truth"
                submittingLabel="Declaring..."
                series={series}
                onSubmit={async data => {
                    const result = await createMutation.mutateAsync({ ...data, series });
                    navigate(`/theory/${result.id}`);
                }}
            />
        </div>
    );
}
