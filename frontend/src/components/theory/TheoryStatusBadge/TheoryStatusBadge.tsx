import type { Theory } from "../../../types/api";
import styles from "./TheoryStatusBadge.module.css";

interface TheoryStatusBadgeProps {
    status: Theory["status"];
}

const labels: Record<Theory["status"], string> = {
    open: "Open",
    contested: "Contested",
    refuted: "Refuted",
};

function statusClass(status: Theory["status"]): string {
    if (status === "refuted") {
        return styles.refuted;
    }
    if (status === "contested") {
        return styles.contested;
    }
    return styles.open;
}

export function TheoryStatusBadge({ status }: TheoryStatusBadgeProps) {
    return <span className={`${styles.badge} ${statusClass(status)}`}>{labels[status] ?? labels.open}</span>;
}
