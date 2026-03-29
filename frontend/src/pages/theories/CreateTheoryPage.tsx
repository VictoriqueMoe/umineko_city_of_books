import {useEffect} from "react";
import {useNavigate} from "react-router";
import {useAuth} from "../../hooks/useAuth";
import {createTheory} from "../../api/endpoints";
import {TheoryForm} from "../../components/theory/TheoryForm/TheoryForm";
import formStyles from "../../components/theory/TheoryForm/TheoryForm.module.css";

export function CreateTheoryPage() {
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

            <TheoryForm
                submitLabel="Declare Blue Truth"
                submittingLabel="Declaring..."
                onSubmit={async data => {
                    const result = await createTheory(data);
                    navigate(`/theory/${result.id}`);
                }}
            />
        </div>
    );
}
