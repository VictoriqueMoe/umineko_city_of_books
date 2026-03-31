import {useEffect, useState} from "react";
import {getRules} from "../../api/endpoints";
import styles from "./RulesBox.module.css";

interface RulesBoxProps {
    page: string;
}

export function RulesBox({ page }: RulesBoxProps) {
    const [rules, setRules] = useState("");

    useEffect(() => {
        getRules(page)
            .then(r => setRules(r.rules))
            .catch(() => {});
    }, [page]);

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
