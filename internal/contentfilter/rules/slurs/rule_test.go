package slurs

import (
	"context"
	"testing"

	"umineko_city_of_books/internal/contentfilter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRule_LegitimateWordsDoNotTrigger(t *testing.T) {
	// given
	rule := New()
	// These MUST all pass the filter - they contain no actual slurs, only
	// substrings that earlier naive filters would have tripped on.
	// Note: "chink in the armour", "tranny" (car transmission) and "dike"
	// (sea wall) are known tradeoffs - we accept blocking those idioms in
	// exchange for catching the primary slur meaning. Homo sapiens remained
	// a hard-not-to-allow case, hence the pattern was removed entirely.
	safeInputs := []string{
		"She is not niggardly with her praise.",
		"Scunthorpe is a town in Lincolnshire.",
		"Homogeneous mixture of isotopes.",
		"Homo sapiens is the genus and species.",
		"Specific, spectacular, species.",
		"Pakistan is a country in South Asia.",
		"Classical assassination plot.",
		"He has a niggling doubt.",
		"Raccoons and other wildlife.",
		"Bombs, bombardier, bombshell.",
		"Analyse this paragraph carefully.",
		// Beatrice gets called a slut in roleplay and general vulgarity
		// is allowed on the site; ensure the slurs rule never starts
		// catching "slut" or its variants.
		"You absolute slut.",
		"slutty behaviour",
		// "queer" is widely reclaimed and acceptable on the site; ensure
		// the rule never starts catching it.
		"queer community",
		"queerness",
		"genderqueer",
		"fag",
		// "retard" / "retarded" are no longer in the pattern list;
		// lock in that they pass.
		"retard",
		"retarded behaviour",
		// The anti-East-Asian, anti-Semitic, anti-Indigenous and anti-Roma
		// categories were intentionally dropped from the filter. Lock in
		// that a representative term from each now passes so no future edit
		// silently reintroduces them.
		"chink",
		"gook",
		"jap",
		"kike",
		"yid",
		"heeb",
		"redskin",
		"injun",
		"gyp",
		"gypped",
	}

	for _, text := range safeInputs {
		// when
		rej, err := rule.Check(context.Background(), []string{text})

		// then
		require.NoError(t, err, "input: %q", text)
		assert.Nil(t, rej, "should not flag %q but did", text)
	}
}

func TestRule_SlurVariantsTrigger(t *testing.T) {
	// given
	rule := New()
	// Exhaustive matrix: for every remaining pattern, cover literal,
	// pluralised, mixed-case, leetspeak and in-context variants.
	// If you add a new pattern, add at least 3 variants here.
	badInputs := []string{
		// anti-Black: n-word
		"nigger",
		"niggers",
		"NIGGER",
		"NiGgEr",
		"n1gger",
		"n!gger",
		"n|gger",
		"n1gg3r",
		"n1gg4",
		"nigga",
		"niggas",
		"N1GG@",
		"niggaaaa you",
		"hey n1gg3rs",
		"go away nigger",

		// anti-Black: coon
		"coon",
		"coons",
		"COON",
		"call me a Coon",

		// anti-Black: porch monkey / jungle bunny
		"porch monkey",
		"porchmonkey",
		"porch-monkey",
		"Porch Monkeys",
		"jungle bunny",
		"junglebunny",
		"jungle-bunnies",

		// anti-Latino
		"spic",
		"spics",
		"sp1c",
		"SP!C",
		"beaner",
		"beaners",
		"be@ner",
		"BE4NERS",
		"wetback",
		"wet-back",
		"wet back",
		"wetbacks",

		// anti-Arab / Middle Eastern
		"raghead",
		"rag-head",
		"rag head",
		"r@gheads",
		"R4GHEAD",
		"towelhead",
		"towlhead",
		"towel-head",
		"towel head",
		"sandnigger",
		"sand-nigger",
		"s@ndn1gg3r",

		// anti-South-Asian (British)
		"paki",
		"pakis",
		"p@k1",
		"P4KI",

		// anti-LGBT
		"faggot",
		"faggots",
		"fagot",
		"f@gg0t",
		"F4GGOT",
		"fagg0ts",
		"you are a f@ggot",
		"tranny",
		"trannies",
		"tr@nny",
		"tr4nny joke",
		"TR4NN!ES",
		"troon",
		"troons",
		"TROON",
		"tr00n",
		"troooons",
		"dyke",
		"dykes",
		"d1ke",
		"d!ke",
		"DYKES",
	}

	for _, text := range badInputs {
		// when
		rej, err := rule.Check(context.Background(), []string{text})

		// then
		require.NoError(t, err, "input: %q", text)
		require.NotNil(t, rej, "should flag %q but did not", text)
		assert.Equal(t, contentfilter.RuleSlurs, rej.Rule)
	}
}

func TestRule_EmptyInput(t *testing.T) {
	// given
	rule := New()

	// when
	rej, err := rule.Check(context.Background(), []string{""})

	// then
	require.NoError(t, err)
	assert.Nil(t, rej)
}
