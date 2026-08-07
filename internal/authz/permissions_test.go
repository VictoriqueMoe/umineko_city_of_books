package authz

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"umineko_city_of_books/internal/role"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func declaredPermissionConstants(t *testing.T) map[string]string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "permissions.go", nil, 0)
	require.NoError(t, err)

	declared := make(map[string]string)
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}

		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			ident, ok := value.Type.(*ast.Ident)
			if !ok || ident.Name != "Permission" {
				continue
			}

			for i, name := range value.Names {
				lit, ok := value.Values[i].(*ast.BasicLit)
				if !ok {
					continue
				}

				declared[name.Name] = lit.Value
			}
		}
	}

	return declared
}

func TestPermissionCatalogue_ClassifiesEveryDeclaredPermission(t *testing.T) {
	// given
	declared := declaredPermissionConstants(t)
	require.NotEmpty(t, declared)

	classified := make(map[string]struct{}, len(permissionCatalogue))
	for _, def := range permissionCatalogue {
		classified[`"`+string(def.Permission)+`"`] = struct{}{}
	}

	// when
	var unclassified []string
	for name, literal := range declared {
		if name == "PermAll" {
			continue
		}

		if _, ok := classified[literal]; !ok {
			unclassified = append(unclassified, name)
		}
	}

	// then
	assert.Empty(t, unclassified, "every Permission constant must be classified in permissionCatalogue with a staff/general scope")
}

func TestPermissionCatalogue_HasNoUnknownEntries(t *testing.T) {
	// given
	declared := declaredPermissionConstants(t)
	literals := make(map[string]struct{}, len(declared))
	for _, literal := range declared {
		literals[literal] = struct{}{}
	}

	// when
	var orphans []Permission
	for _, def := range permissionCatalogue {
		if _, ok := literals[`"`+string(def.Permission)+`"`]; !ok {
			orphans = append(orphans, def.Permission)
		}
	}

	// then
	assert.Empty(t, orphans, "permissionCatalogue must not contain permissions with no declared constant")
}

func TestPermissionCatalogue_ScopeIsAlwaysValid(t *testing.T) {
	for _, def := range permissionCatalogue {
		t.Run(string(def.Permission), func(t *testing.T) {
			// given
			scope := def.Scope

			// when
			valid := scope == ScopeStaff || scope == ScopeGeneral || scope == ScopeRestricted

			// then
			assert.True(t, valid, "scope must be staff, general or restricted")
			assert.NotEmpty(t, def.Label, "every permission needs a human label")
		})
	}
}

func TestPermissionCatalogue_ExcludesPermAll(t *testing.T) {
	// given
	catalogue := PermissionCatalogue()

	// when
	var found bool
	for _, def := range catalogue {
		if def.Permission == PermAll {
			found = true
		}
	}

	// then
	assert.False(t, found, "PermAll is a code-only wildcard and must never be storable or assignable")
}

func TestIsVanityAssignable(t *testing.T) {
	cases := []struct {
		name string
		perm Permission
		want bool
	}{
		{"use chatbot is assignable", PermUseChatbot, true},
		{"manage vanity roles is staff only", PermManageVanityRoles, false},
		{"manage roles is staff only", PermManageRoles, false},
		{"manage settings is staff only", PermManageSettings, false},
		{"ban user is staff only", PermBanUser, false},
		{"wildcard is never assignable", PermAll, false},
		{"unknown permission is never assignable", Permission("teleport"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			perm := tc.perm

			// when
			got := IsVanityAssignable(perm)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestVanityAssignablePermissions_OnlyGeneralScope(t *testing.T) {
	// given
	assignable := VanityAssignablePermissions()

	// when
	require.NotEmpty(t, assignable)

	// then
	for _, perm := range assignable {
		def, ok := LookupPermission(perm)
		require.True(t, ok)
		assert.Equal(t, ScopeGeneral, def.Scope)
	}
}

func TestImmutableAndEditableRoles(t *testing.T) {
	cases := []struct {
		name          string
		role          role.Role
		wantImmutable bool
		wantEditable  bool
	}{
		{"super admin is immutable", RoleSuperAdmin, true, false},
		{"admin is immutable", RoleAdmin, true, false},
		{"moderator is editable", RoleModerator, false, true},
		{"no role is neither", "", false, false},
		{"unknown role is neither", "gardener", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			r := tc.role

			// when
			immutable := IsImmutableRole(r)
			editable := IsEditableSystemRole(r)

			// then
			assert.Equal(t, tc.wantImmutable, immutable)
			assert.Equal(t, tc.wantEditable, editable)
		})
	}
}

func TestEditableSystemRoles_ExcludesImmutableRoles(t *testing.T) {
	// given
	editable := EditableSystemRoles()

	// when
	require.Len(t, editable, 1)

	// then
	for _, r := range editable {
		assert.False(t, IsImmutableRole(r), "an immutable role must never be exposed as editable")
	}
}

func TestDefaultRolePermissions_MatchesSeededModeratorSet(t *testing.T) {
	// given
	want := []Permission{
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

	// when
	got := DefaultRolePermissions(RoleModerator)

	// then
	assert.Equal(t, want, got)
	assert.Nil(t, DefaultRolePermissions(RoleAdmin))
	assert.Nil(t, DefaultRolePermissions(RoleSuperAdmin))
}
