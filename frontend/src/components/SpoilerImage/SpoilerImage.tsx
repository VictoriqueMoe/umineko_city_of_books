import { SpoilerCover } from "./SpoilerCover";
import { spoilerBlurClass } from "./spoilerBlur";
import styles from "./SpoilerImage.module.css";

interface SpoilerImageProps {
    src: string;
    alt?: string;
    isSpoiler: boolean;
    className?: string;
    imageClassName?: string;
    compact?: boolean;
    width?: number;
    height?: number;
    onClick?: () => void;
    loading?: "lazy" | "eager";
    onError?: (e: React.SyntheticEvent<HTMLImageElement>) => void;
}

export function SpoilerImage({
    src,
    alt = "",
    isSpoiler,
    className,
    imageClassName,
    compact,
    width,
    height,
    onClick,
    loading,
    onError,
}: SpoilerImageProps) {
    return (
        <SpoilerCover isSpoiler={isSpoiler} className={className} compact={compact} onClick={onClick}>
            {covered => (
                <img
                    src={src}
                    alt={alt}
                    className={`${styles.image} ${spoilerBlurClass(covered)}${imageClassName ? ` ${imageClassName}` : ""}`}
                    width={width}
                    height={height}
                    decoding="async"
                    loading={loading}
                    onError={onError}
                />
            )}
        </SpoilerCover>
    );
}
