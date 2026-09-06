export function mediaFormData(file: File, isSpoiler: boolean): FormData {
    const formData = new FormData();
    formData.append("media", file);
    if (isSpoiler) {
        formData.append("is_spoiler", "true");
    }

    return formData;
}
