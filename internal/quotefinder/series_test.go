package quotefinder

import (
	"testing"
)

func TestSeriesValid(t *testing.T) {
	cases := []struct {
		name string
		s    Series
		want bool
	}{
		{"umineko", SeriesUmineko, true},
		{"higurashi", SeriesHigurashi, true},
		{"ciconia", SeriesCiconia, true},
		{"empty", Series(""), false},
		{"unknown", Series("roseguns"), false},
		{"case sensitive", Series("Ciconia"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.Valid(); got != tc.want {
				t.Fatalf("%q.Valid() = %v, want %v", tc.s, got, tc.want)
			}
		})
	}
}

func TestParseSeries(t *testing.T) {
	ok := []struct {
		in   string
		want Series
	}{
		{"umineko", SeriesUmineko},
		{"UMINEKO", SeriesUmineko},
		{"  higurashi  ", SeriesHigurashi},
		{"Ciconia", SeriesCiconia},
		{"ciconia", SeriesCiconia},
	}

	for _, tc := range ok {
		t.Run("ok_"+tc.in, func(t *testing.T) {
			got, err := ParseSeries(tc.in)
			if err != nil {
				t.Fatalf("ParseSeries(%q) unexpected error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("ParseSeries(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}

	bad := []string{"", "roseguns", "umin", "higu "}
	for _, in := range bad {
		t.Run("bad_"+in, func(t *testing.T) {
			if _, err := ParseSeries(in); err == nil {
				t.Fatalf("ParseSeries(%q) expected error, got nil", in)
			}
		})
	}
}
