import Hyperbeam from "@hyperbeam/web";

export type HyperbeamHandle = Awaited<ReturnType<typeof Hyperbeam>>;

export interface MountHyperbeamEmbedOptions {
    container: HTMLDivElement;
    embedURL: string;
    inputEnabled: boolean;
}

export async function mountHyperbeamEmbed(options: MountHyperbeamEmbedOptions): Promise<HyperbeamHandle> {
    return Hyperbeam(options.container, options.embedURL, {
        delegateKeyboard: true,
        disableInput: !options.inputEnabled,
    });
}

export function destroyHyperbeamEmbed(handle: HyperbeamHandle | null | undefined): void {
    if (!handle) {
        return;
    }

    handle.destroy();
}

export function setHyperbeamInputEnabled(handle: HyperbeamHandle | null | undefined, enabled: boolean): void {
    if (!handle) {
        return;
    }

    handle.disableInput = !enabled;
}
