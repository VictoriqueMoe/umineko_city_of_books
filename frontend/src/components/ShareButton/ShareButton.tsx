import { useState } from "react";
import { useAuth } from "../../hooks/useAuth";
import { Button } from "../Button/Button";
import { ShareDialog } from "../post/ShareDialog/ShareDialog";

interface ShareButtonProps {
    contentId: string;
    contentType: string;
    contentTitle?: string;
    shareCount?: number;
    onShared?: () => void;
}

export function ShareButton({ contentId, contentType, contentTitle, shareCount, onShared }: ShareButtonProps) {
    const { user } = useAuth();
    const [open, setOpen] = useState(false);

    if (!user) {
        return null;
    }

    return (
        <>
            <Button variant="ghost" size="small" onClick={() => setOpen(true)}>
                Share{shareCount != null && shareCount > 0 ? ` ${shareCount}` : ""}
            </Button>
            {open && (
                <ShareDialog
                    isOpen={open}
                    onClose={() => setOpen(false)}
                    contentId={contentId}
                    contentType={contentType}
                    contentTitle={contentTitle}
                    onShared={onShared}
                />
            )}
        </>
    );
}
