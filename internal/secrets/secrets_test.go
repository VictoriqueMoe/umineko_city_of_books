package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookup_Known(t *testing.T) {
	spec, ok := Lookup("witchHunter")
	require.True(t, ok)
	assert.Equal(t, WitchHunter, spec.ID)
	assert.NotEmpty(t, spec.ExpectedHash)
	assert.Equal(t, "system_witch_hunter", spec.VanityRoleID)
	assert.Equal(t, "The Witch's Epitaph", spec.Title)
	assert.Len(t, spec.PieceIDs, 12)
}

func TestLookup_Unknown(t *testing.T) {
	_, ok := Lookup("nonsense")
	assert.False(t, ok)
}

func TestListed_ReturnsOnlyParents(t *testing.T) {
	listed := Listed()
	require.Len(t, listed, 1)
	assert.Equal(t, WitchHunter, listed[0].ID)
	assert.NotEmpty(t, listed[0].Title)
}

func TestAll_IncludesEverything(t *testing.T) {
	all := All()
	assert.Equal(t, 13, len(all))
}

func TestWithVanityRole_OnlyParents(t *testing.T) {
	withRole := WithVanityRole()
	require.Len(t, withRole, 1)
	assert.Equal(t, WitchHunter, withRole[0].ID)
	assert.Equal(t, "system_witch_hunter", withRole[0].VanityRoleID)
}

func TestParentOf_Piece(t *testing.T) {
	parent, ok := ParentOf(Piece04)
	require.True(t, ok)
	assert.Equal(t, WitchHunter, parent.ID)
}

func TestParentOf_Parent(t *testing.T) {
	parent, ok := ParentOf(WitchHunter)
	require.True(t, ok)
	assert.Equal(t, WitchHunter, parent.ID)
}

func TestParentOf_Unknown(t *testing.T) {
	_, ok := ParentOf(ID("nonsense"))
	assert.False(t, ok)
}

func TestPieceIDStrings_OrderPreserved(t *testing.T) {
	spec, _ := Lookup(string(WitchHunter))
	strs := PieceIDStrings(spec)
	require.Len(t, strs, 12)
	assert.Equal(t, string(Piece01), strs[0])
	assert.Equal(t, string(Piece12), strs[11])
}
