package authz

import (
	"slices"

	"umineko_city_of_books/internal/role"
)

type (
	Permission string

	PermissionScope string

	PermissionDef struct {
		Permission Permission      `json:"permission"`
		Label      string          `json:"label"`
		Scope      PermissionScope `json:"scope"`
	}
)

const (
	ScopeStaff      PermissionScope = "staff"
	ScopeGeneral    PermissionScope = "general"
	ScopeRestricted PermissionScope = "restricted"
)

const (
	PermAll               Permission = "*"
	PermViewAdminPanel    Permission = "view_admin_panel"
	PermViewStats         Permission = "view_stats"
	PermViewAuditLog      Permission = "view_audit_log"
	PermManageSettings    Permission = "manage_settings"
	PermManageRoles       Permission = "manage_roles"
	PermDeleteAnyTheory   Permission = "delete_any_theory"
	PermDeleteAnyResponse Permission = "delete_any_response"
	PermDeleteAnyUser     Permission = "delete_any_user"
	PermBanUser           Permission = "ban_user"
	PermViewUsers         Permission = "view_users"
	PermDeleteAnyPost     Permission = "delete_any_post"
	PermDeleteAnyComment  Permission = "delete_any_comment"
	PermEditAnyTheory     Permission = "edit_any_theory"
	PermEditAnyPost       Permission = "edit_any_post"
	PermEditAnyComment    Permission = "edit_any_comment"
	PermResolveSuggestion Permission = "resolve_suggestion"
	PermEditMysteryScore  Permission = "edit_mystery_score"
	PermEditAnyJournal    Permission = "edit_any_journal"
	PermDeleteAnyJournal  Permission = "delete_any_journal"
	PermManageVanityRoles Permission = "manage_vanity_roles"
	PermManageBannedWords Permission = "manage_banned_words"
	PermResetPassword     Permission = "reset_password"
	PermManageUserAccount Permission = "manage_user_account"
	PermManageUserEmail   Permission = "manage_user_email"
	PermSetEmailVerified  Permission = "set_email_verified"
	PermUseChatbot        Permission = "use_chatbot"
)

var (
	permissionCatalogue = []PermissionDef{
		{PermViewAdminPanel, "View admin panel", ScopeStaff},
		{PermViewStats, "View site stats", ScopeStaff},
		{PermViewAuditLog, "View audit log", ScopeStaff},
		{PermManageSettings, "Manage site settings", ScopeRestricted},
		{PermManageRoles, "Manage roles and permissions", ScopeRestricted},
		{PermDeleteAnyTheory, "Delete any theory", ScopeStaff},
		{PermDeleteAnyResponse, "Delete any response", ScopeStaff},
		{PermDeleteAnyUser, "Delete any user", ScopeStaff},
		{PermBanUser, "Ban and lock users", ScopeStaff},
		{PermViewUsers, "View user records", ScopeStaff},
		{PermDeleteAnyPost, "Delete any post", ScopeStaff},
		{PermDeleteAnyComment, "Delete any comment", ScopeStaff},
		{PermEditAnyTheory, "Edit any theory", ScopeStaff},
		{PermEditAnyPost, "Edit any post", ScopeStaff},
		{PermEditAnyComment, "Edit any comment", ScopeStaff},
		{PermResolveSuggestion, "Resolve suggestions", ScopeStaff},
		{PermEditMysteryScore, "Edit mystery scores", ScopeStaff},
		{PermEditAnyJournal, "Edit any journal", ScopeStaff},
		{PermDeleteAnyJournal, "Delete any journal", ScopeStaff},
		{PermManageVanityRoles, "Manage vanity roles", ScopeStaff},
		{PermManageBannedWords, "Manage banned words", ScopeStaff},
		{PermResetPassword, "Reset user passwords", ScopeStaff},
		{PermManageUserAccount, "Manage user accounts", ScopeStaff},
		{PermManageUserEmail, "Manage user email addresses", ScopeStaff},
		{PermSetEmailVerified, "Set email verified", ScopeStaff},
		{PermUseChatbot, "Summon chatbots", ScopeGeneral},
	}

	permissionIndex = buildPermissionIndex()

	defaultModeratorPermissions = []Permission{
		PermViewAdminPanel,
		PermViewStats,
		PermViewUsers,
		PermDeleteAnyTheory,
		PermDeleteAnyResponse,
		PermDeleteAnyPost,
		PermDeleteAnyComment,
		PermEditAnyTheory,
		PermEditAnyPost,
		PermEditAnyComment,
		PermBanUser,
		PermEditMysteryScore,
		PermEditAnyJournal,
		PermDeleteAnyJournal,
		PermManageUserAccount,
		PermUseChatbot,
	}
)

func buildPermissionIndex() map[Permission]PermissionDef {
	index := make(map[Permission]PermissionDef, len(permissionCatalogue))
	for _, def := range permissionCatalogue {
		index[def.Permission] = def
	}

	return index
}

func PermissionCatalogue() []PermissionDef {
	return slices.Clone(permissionCatalogue)
}

func LookupPermission(perm Permission) (PermissionDef, bool) {
	def, ok := permissionIndex[perm]

	return def, ok
}

func IsKnownPermission(perm Permission) bool {
	_, ok := permissionIndex[perm]

	return ok
}

func IsVanityAssignable(perm Permission) bool {
	def, ok := permissionIndex[perm]

	return ok && def.Scope == ScopeGeneral
}

func IsRoleAssignable(perm Permission) bool {
	def, ok := permissionIndex[perm]

	return ok && def.Scope != ScopeRestricted
}

func RoleAssignablePermissions() []PermissionDef {
	result := make([]PermissionDef, 0, len(permissionCatalogue))
	for _, def := range permissionCatalogue {
		if def.Scope == ScopeRestricted {
			continue
		}

		result = append(result, def)
	}

	return result
}

func VanityAssignablePermissions() []Permission {
	var result []Permission
	for _, def := range permissionCatalogue {
		if def.Scope == ScopeGeneral {
			result = append(result, def.Permission)
		}
	}

	return result
}

func IsImmutableRole(r role.Role) bool {
	return r == RoleAdmin || r == RoleSuperAdmin
}

func IsEditableSystemRole(r role.Role) bool {
	return r == RoleModerator
}

func EditableSystemRoles() []role.Role {
	return []role.Role{RoleModerator}
}

func DefaultRolePermissions(r role.Role) []Permission {
	if r == RoleModerator {
		return slices.Clone(defaultModeratorPermissions)
	}

	return nil
}
