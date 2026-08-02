import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, type RenderOptions, type RenderResult } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import type { SiteInfo } from "../api/endpoints";
import { AuthContext, type AuthContextValue } from "../context/authContextValue";
import { GifFavouritesContext, type GifFavouritesContextValue } from "../context/gifFavouritesContextValue";
import { MentionResolverContext, type MentionResolverContextValue } from "../context/mentionResolverContextValue";
import { NotificationContext, type NotificationContextValue } from "../context/notificationContextValue";
import { ThemeContext, type ThemeContextValue } from "../context/themeContextValue";
import { SiteInfoContext } from "../context/siteInfoContextValue";
import type { UserProfile } from "../types/api";
import {
    makeAuthContext,
    makeGifFavouritesContext,
    makeNotificationContext,
    makeSiteInfo,
    makeThemeContext,
} from "./fixtures";

export interface ProviderOptions {
    user?: UserProfile | null;
    auth?: Partial<AuthContextValue>;
    siteInfo?: Partial<SiteInfo>;
    notification?: Partial<NotificationContextValue>;
    theme?: Partial<ThemeContextValue>;
    gifFavourites?: Partial<GifFavouritesContextValue>;
    mentionResolver?: MentionResolverContextValue;
    queryClient?: QueryClient;
    route?: string;
    path?: string;
}

export function createTestQueryClient(): QueryClient {
    return new QueryClient({
        defaultOptions: {
            queries: { retry: false, gcTime: 0, staleTime: 0, refetchOnWindowFocus: false },
            mutations: { retry: false },
        },
    });
}

const silentResolver: MentionResolverContextValue = {
    isKnown: () => undefined,
    request: () => {},
};

function wrap(children: ReactNode, options: ProviderOptions, queryClient: QueryClient): ReactElement {
    const auth = makeAuthContext({ user: options.user ?? null, ...options.auth });
    const siteInfo = makeSiteInfo(options.siteInfo);
    const notification = makeNotificationContext(options.notification);
    const theme = makeThemeContext(options.theme);
    const gifFavourites = makeGifFavouritesContext(options.gifFavourites);
    const resolver = options.mentionResolver ?? silentResolver;

    const routed = options.path ? <Routes>{<Route path={options.path} element={children} />}</Routes> : children;

    return (
        <QueryClientProvider client={queryClient}>
            <MemoryRouter initialEntries={[options.route ?? "/"]}>
                <SiteInfoContext.Provider value={siteInfo}>
                    <AuthContext.Provider value={auth}>
                        <ThemeContext.Provider value={theme}>
                            <NotificationContext.Provider value={notification}>
                                <GifFavouritesContext.Provider value={gifFavourites}>
                                    <MentionResolverContext.Provider value={resolver}>
                                        {routed}
                                    </MentionResolverContext.Provider>
                                </GifFavouritesContext.Provider>
                            </NotificationContext.Provider>
                        </ThemeContext.Provider>
                    </AuthContext.Provider>
                </SiteInfoContext.Provider>
            </MemoryRouter>
        </QueryClientProvider>
    );
}

export interface RenderWithProvidersResult extends RenderResult {
    queryClient: QueryClient;
}

export function renderWithProviders(
    ui: ReactElement,
    options: ProviderOptions = {},
    renderOptions: Omit<RenderOptions, "wrapper"> = {},
): RenderWithProvidersResult {
    const queryClient = options.queryClient ?? createTestQueryClient();
    const result = render(wrap(ui, options, queryClient), renderOptions);

    return {
        ...result,
        queryClient,
        rerender: (next: ReactNode) => result.rerender(wrap(next, options, queryClient)),
    };
}

export function providerWrapper(
    options: ProviderOptions = {},
): ({ children }: { children: ReactNode }) => ReactElement {
    const queryClient = options.queryClient ?? createTestQueryClient();
    return ({ children }: { children: ReactNode }) => wrap(children, options, queryClient);
}
