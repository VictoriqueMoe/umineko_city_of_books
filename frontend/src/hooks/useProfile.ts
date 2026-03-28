import {useCallback, useEffect, useState} from "react";
import type {UserProfile} from "../types/api";
import {getUserProfile} from "../api/endpoints";

export function useProfile(username: string) {
    const [profile, setProfile] = useState<UserProfile | null>(null);
    const [loading, setLoading] = useState(true);

    const fetchProfile = useCallback(async (name: string) => {
        setLoading(true);
        try {
            const result = await getUserProfile(name);
            setProfile(result);
        } catch {
            setProfile(null);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchProfile(username);
    }, [username, fetchProfile]);

    return { profile, loading };
}
