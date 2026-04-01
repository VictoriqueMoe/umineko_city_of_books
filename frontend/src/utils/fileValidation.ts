function formatSize(bytes: number): string {
    if (bytes < 1024) {
        return `${bytes} B`;
    }
    if (bytes < 1024 * 1024) {
        return `${(bytes / 1024).toFixed(1)} KB`;
    }
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function validateFileSize(file: File, maxImageSize: number, maxVideoSize: number): string | null {
    const isVideo = file.type.startsWith("video/");
    const maxSize = isVideo ? maxVideoSize : maxImageSize;

    if (file.size > maxSize) {
        return `${file.name} is too large (${formatSize(file.size)}). Maximum ${isVideo ? "video" : "image"} size is ${formatSize(maxSize)}.`;
    }

    return null;
}
