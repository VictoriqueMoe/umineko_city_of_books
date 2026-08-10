import { Link } from "react-router";
import type { User } from "../../../types/api";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { formatFullDateTime } from "../../../utils/time";
import styles from "./RefutationStamp.module.css";

interface RefutationStampProps {
    responseId?: string;
    refutedBy?: User;
    refutedAt?: string;
}

export function RefutationStamp({ responseId, refutedBy, refutedAt }: RefutationStampProps) {
    return (
        <div className={styles.stamp}>
            <span className={styles.label}>Refuted</span>
            {refutedBy && (
                <span className={styles.by}>
                    by <ProfileLink user={refutedBy} size="small" />
                </span>
            )}
            {responseId ? (
                <Link to={`#response-${responseId}`} className={styles.link}>
                    read the refutation
                </Link>
            ) : (
                refutedBy && <span className={styles.gone}>(the refuting response was deleted)</span>
            )}
            {refutedAt && <span className={styles.when}>{formatFullDateTime(refutedAt)}</span>}
        </div>
    );
}
