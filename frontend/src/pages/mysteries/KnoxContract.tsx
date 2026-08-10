import type { KnoxContract as KnoxContractType } from "../../types/api";
import { swornRules } from "./knoxRules";
import styles from "./MysteryPages.module.css";

interface KnoxContractProps {
    contract: KnoxContractType;
}

export function KnoxContract({ contract }: KnoxContractProps) {
    const sworn = swornRules(contract);

    return (
        <section className={styles.knoxContract} aria-labelledby="knox-contract-title">
            <h3 id="knox-contract-title" className={styles.knoxTitle}>
                Knox's Decalogue
            </h3>
            {sworn.length === 0 ? (
                <p className={styles.knoxIntro}>
                    The Game Master swears nothing. Nothing is forbidden here. Tread carefully.
                </p>
            ) : (
                <>
                    <p className={styles.knoxIntro}>
                        The Game Master swears the following. What is not sworn here is permitted.
                    </p>
                    <ul className={styles.knoxList}>
                        {sworn.map(rule => (
                            <li key={rule.key} className={styles.knoxRule}>
                                <strong>{rule.ordinal}.</strong> {rule.sworn}
                            </li>
                        ))}
                    </ul>
                    <p className={styles.knoxOutro}>
                        These are the terms of this game board. Read them before you declare your blue truth. GOOD?
                    </p>
                </>
            )}
        </section>
    );
}
