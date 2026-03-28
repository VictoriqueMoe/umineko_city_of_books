import type { PropsWithChildren } from "react";

interface ModalProps {
    isOpen: boolean;
    onClose: () => void;
    title: string;
}

export function Modal({ isOpen, onClose, title, children }: PropsWithChildren<ModalProps>) {
    if (!isOpen) {
        return null;
    }

    return (
        <div className="picker-overlay" onClick={onClose}>
            <div className="picker-modal" onClick={e => e.stopPropagation()}>
                <div className="picker-header">
                    <h3>{title}</h3>
                    <button className="picker-close" onClick={onClose}>
                        {"\u2715"}
                    </button>
                </div>
                {children}
            </div>
        </div>
    );
}
