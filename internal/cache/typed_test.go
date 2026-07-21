package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sample struct {
	Name string `json:"name"`
	N    int    `json:"n"`
}

func roundTrip[T any](t *testing.T, value T) {
	t.Helper()

	data, err := encode(value)
	require.NoError(t, err)

	got, err := decode[T](data)
	require.NoError(t, err)

	assert.Equal(t, value, got)
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		roundTrip(t, "hello featherine")
	})
	t.Run("bytes", func(t *testing.T) {
		roundTrip(t, []byte{0x00, 0x01, 0xff, 0x10, 0x7f})
	})
	t.Run("struct", func(t *testing.T) {
		roundTrip(t, sample{Name: "beatrice", N: 7})
	})
	t.Run("pointer", func(t *testing.T) {
		roundTrip(t, &sample{Name: "battler", N: 3})
	})
	t.Run("int", func(t *testing.T) {
		roundTrip(t, 1998)
	})
}

func TestEncodeStoresBytesAndStringsRaw(t *testing.T) {
	raw := []byte{0x00, 0x10, 0xff}

	encodedBytes, err := encode(raw)
	require.NoError(t, err)
	assert.Equal(t, raw, encodedBytes)

	encodedString, err := encode("hi")
	require.NoError(t, err)
	assert.Equal(t, []byte("hi"), encodedString)
}

func TestEncodeStructUsesJSON(t *testing.T) {
	encoded, err := encode(sample{Name: "ange", N: 12})
	require.NoError(t, err)

	assert.Equal(t, `{"name":"ange","n":12}`, string(encoded))
}

func TestEncodeDecodeNilPointer(t *testing.T) {
	var original *sample

	encoded, err := encode(original)
	require.NoError(t, err)
	assert.Equal(t, "null", string(encoded))

	got, err := decode[*sample](encoded)
	require.NoError(t, err)
	assert.Nil(t, got)
}
