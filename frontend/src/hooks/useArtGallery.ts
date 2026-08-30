import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import { groupByAuthor, type ArtistGalleries } from "../domain/art/groupByAuthor";
import type { Art, Gallery, TagCount, UserProfile } from "../types/api";
import { errorMessage } from "../utils/errorMessage";
import { useAuth } from "./useAuth";
import { useSearchParamPage, type PageOffset } from "./usePageOffset";
import { useCreateGallery } from "./mutations/art";
import { useAllGalleries, useArtFeed } from "./queries/art";
import { usePopularTags } from "./queries/art";
import { useUserGalleries } from "./queries/user";

export const ART_LIMIT = 24;
const SEARCH_DEBOUNCE_MS = 300;

export type ParamUpdates = Record<string, string | undefined>;

export interface ArtGallery {
    viewer: UserProfile | null;
    viewMode: string;
    sort: string;
    search: string;
    activeTag: string;
    activeType: string;
    searchInput: string;
    setSearchInput: (value: string) => void;
    updateParams: (updates: ParamUpdates) => void;
    page: PageOffset;
    art: Art[];
    total: number;
    feedLoading: boolean;
    popularTags: TagCount[];
    artists: ArtistGalleries[];
    galleriesLoading: boolean;
    userGalleries: Gallery[];
    selectedGallery: string;
    setSelectedGallery: (id: string) => void;
    showUpload: boolean;
    setShowUpload: (open: boolean) => void;
    newGalleryName: string;
    setNewGalleryName: (name: string) => void;
    galleryError: string;
    creatingGallery: boolean;
    createGallery: () => Promise<void>;
}

export function useArtGallery(corner: string): ArtGallery {
    const { user } = useAuth();
    const [searchParams, setSearchParams] = useSearchParams();

    const viewMode = searchParams.get("view") || "galleries";
    const sort = searchParams.get("sort") || "new";
    const search = searchParams.get("search") || "";
    const activeTag = searchParams.get("tag") || "";
    const activeType = searchParams.get("type") || "";

    const page = useSearchParamPage({ limit: ART_LIMIT, replace: true });

    const [searchInput, setSearchInput] = useState(search);
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

    const [selectedGalleryOverride, setSelectedGalleryOverride] = useState<string | null>(null);
    const [showUpload, setShowUpload] = useState(false);
    const [newGalleryName, setNewGalleryName] = useState("");
    const [galleryError, setGalleryError] = useState("");

    const feed = useArtFeed(
        corner,
        activeType || undefined,
        search || undefined,
        activeTag || undefined,
        sort,
        page.offset,
        page.limit,
    );

    const { tags: popularTags } = usePopularTags(corner);
    const { galleries: userGalleries, refresh: refreshUserGalleries } = useUserGalleries(user?.id ?? "");
    const { galleries: allGalleries, loading: galleriesLoading } = useAllGalleries(corner, viewMode === "galleries");
    const createGalleryMutation = useCreateGallery();

    const selectedGallery = selectedGalleryOverride ?? userGalleries[0]?.id ?? "";

    const updateParams = useCallback(
        (updates: ParamUpdates) => {
            setSearchParams(
                prev => {
                    const next = new URLSearchParams(prev);
                    for (const [key, value] of Object.entries(updates)) {
                        if (
                            value &&
                            value !== "" &&
                            !(key === "sort" && value === "new") &&
                            !(key === "page" && value === "1") &&
                            !(key === "view" && value === "galleries")
                        ) {
                            next.set(key, value);
                        } else {
                            next.delete(key);
                        }
                    }
                    return next;
                },
                { replace: true },
            );
        },
        [setSearchParams],
    );

    useEffect(() => {
        if (searchInput === search) {
            return;
        }

        clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => {
            updateParams({ search: searchInput || undefined, page: "1" });
        }, SEARCH_DEBOUNCE_MS);

        return () => clearTimeout(debounceRef.current);
    }, [searchInput, search, updateParams]);

    async function createGallery() {
        if (!newGalleryName.trim()) {
            return;
        }

        setGalleryError("");

        try {
            const { id } = await createGalleryMutation.mutateAsync({ name: newGalleryName.trim() });
            setNewGalleryName("");
            await refreshUserGalleries();
            setSelectedGalleryOverride(id);
        } catch (e) {
            setGalleryError(errorMessage(e, "Failed to create the gallery"));
        }
    }

    return {
        viewer: user,
        viewMode,
        sort,
        search,
        activeTag,
        activeType,
        searchInput,
        setSearchInput,
        updateParams,
        page,
        art: feed.art,
        total: feed.total,
        feedLoading: feed.loading,
        popularTags,
        artists: groupByAuthor(allGalleries),
        galleriesLoading,
        userGalleries,
        selectedGallery,
        setSelectedGallery: setSelectedGalleryOverride,
        showUpload,
        setShowUpload,
        newGalleryName,
        setNewGalleryName,
        galleryError,
        creatingGallery: createGalleryMutation.isPending,
        createGallery,
    };
}
