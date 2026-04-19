import { PieceTrigger } from "../../../features/easterEgg";
import styles from "./Footer.module.css";

export function Footer() {
    return (
        <footer className={styles.footer}>
            Umineko no Naku Koro ni - 07th Expansion <PieceTrigger pieceId="piece_02" />
        </footer>
    );
}
