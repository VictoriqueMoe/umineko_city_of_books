import { useRules } from "../../api/queries/misc";
import styles from "./RulesBox.module.css";

interface RulesBoxProps {
    page: string;
}

export function RulesBox({ page }: RulesBoxProps) {
    const { rules } = useRules(page);

    if (!rules) {
        return null;
    }

    return (
        <div className={styles.box}>
            <div className={styles.label}>Rules</div>
            <div className={styles.content}>{rules}</div>
        </div>
    );
}
