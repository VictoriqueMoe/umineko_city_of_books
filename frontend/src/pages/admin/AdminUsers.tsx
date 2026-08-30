import React, { useState } from "react";
import { useNavigate } from "react-router";
import { useAdminUsers } from "../../hooks/queries/admin";
import { hasNextPage, usePageOffset } from "../../hooks/usePageOffset";
import { usePageTitle } from "../../hooks/usePageTitle";
import { ErrorBanner } from "../../components/ErrorBanner/ErrorBanner";
import { Input } from "../../components/Input/Input";
import { Pagination } from "../../components/Pagination/Pagination";
import { RolePill } from "../../components/RolePill/RolePill";
import { formatDate } from "../../utils/time";
import styles from "./AdminUsers.module.css";

const LIMIT = 20;

export function AdminUsers() {
    usePageTitle("Admin - Users");
    const navigate = useNavigate();
    const page = usePageOffset({ limit: LIMIT });
    const [search, setSearch] = useState("");
    const [committed, setCommitted] = useState("");
    const { users, total, loading, error } = useAdminUsers(committed, page.limit, page.offset);

    function handleSearch(e: React.SubmitEvent) {
        e.preventDefault();
        page.reset();
        setCommitted(search);
    }

    return (
        <div className={styles.page}>
            <h1 className={styles.title}>Users</h1>

            <form className={styles.searchRow} onSubmit={handleSearch}>
                <Input
                    placeholder="Search users..."
                    value={search}
                    onChange={e => setSearch(e.target.value)}
                    fullWidth
                />
            </form>

            {loading && <div className={styles.loading}>Loading users...</div>}

            {!loading && error && <ErrorBanner message="Could not load the user list." />}

            {!loading && !error && (
                <>
                    {users.length === 0 ? (
                        <div className={styles.empty}>No users found</div>
                    ) : (
                        <table className={styles.table}>
                            <thead>
                                <tr>
                                    <th>User</th>
                                    <th>Display Name</th>
                                    <th>Role</th>
                                    <th>Status</th>
                                    <th>Joined</th>
                                </tr>
                            </thead>
                            <tbody>
                                {users.map(u => (
                                    <tr
                                        key={u.id}
                                        className={styles.row}
                                        onClick={() => navigate(`/admin/users/${u.id}`)}
                                    >
                                        <td>
                                            <div className={styles.userCell}>
                                                {u.avatar_url ? (
                                                    <img className={styles.avatar} src={u.avatar_url} alt="" />
                                                ) : (
                                                    <span className={styles.avatarPlaceholder}>
                                                        {u.display_name[0]}
                                                    </span>
                                                )}
                                                {u.username}
                                            </div>
                                        </td>
                                        <td dir="auto">{u.display_name}</td>
                                        <td>
                                            <RolePill role={u.role ?? ""} userId={u.id} />
                                        </td>
                                        <td>
                                            {u.banned ? (
                                                <span className={styles.banned}>Banned</span>
                                            ) : (
                                                <span className={styles.notBanned}>Active</span>
                                            )}
                                        </td>
                                        <td>{formatDate(u.created_at)}</td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    )}

                    <Pagination
                        offset={page.offset}
                        limit={page.limit}
                        total={total}
                        hasNext={hasNextPage(page, total)}
                        hasPrev={page.hasPrev}
                        onNext={page.goNext}
                        onPrev={page.goPrev}
                    />
                </>
            )}
        </div>
    );
}
