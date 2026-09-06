import styles from "./SpoilerImage.module.css";

export function spoilerBlurClass(covered: boolean): string {
    return covered ? styles.blurred : "";
}
