import type { Art } from "../../../types/api";
import { ArtCard } from "../ArtCard/ArtCard";
import styles from "./ArtGrid.module.css";

interface ArtGridProps {
    art: Art[];
}

export function ArtGrid({ art }: ArtGridProps) {
    return (
        <div className={styles.grid}>
            {art.map(a => (
                <ArtCard key={a.id} art={a} />
            ))}
        </div>
    );
}
