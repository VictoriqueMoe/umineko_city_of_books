import React, { useState } from "react";
import { useNavigate } from "react-router";
import { useAdminUsers } from "../../api/queries/admin";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Input } from "../../components/Input/Input";
import { Pagination } from "../../components/Pagination/Pagination";
import { RolePill } from "../../components/RolePill/RolePill";
import { formatDate } from "../../utils/time";
import styles from "./AdminUsers.module.css";

const LIMIT = 20;

export function AdminUsers() {
    usePageTitle("Admin - Users");
    const navigate = useNavigate();
    const [offset, setOffset] = useState(0);
    const [search, setSearch] = useState("");
    const [committed, setCommitted] = useState("");
    const { users, total, loading } = useAdminUsers(committed, LIMIT, offset);
    const error = "";

    function handleSearch(e: React.SubmitEvent) {
        e.preventDefault();
        setOffset(0);
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
            {error && <div className={styles.error}>{error}</div>}

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
                                        <td>{u.display_name}</td>
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
                        offset={offset}
                        limit={LIMIT}
                        total={total}
                        hasNext={offset + LIMIT < total}
                        hasPrev={offset > 0}
                        onNext={() => setOffset(prev => prev + LIMIT)}
                        onPrev={() => setOffset(prev => Math.max(0, prev - LIMIT))}
                    />
                </>
            )}
        </div>
    );
}
