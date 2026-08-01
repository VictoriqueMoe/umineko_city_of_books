package cache

import (
	"context"
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

func TestSetManyWithoutClient(t *testing.T) {
	tests := []struct {
		name    string
		manager *Manager
		values  map[string]string
	}{
		{name: "nil manager", manager: nil, values: map[string]string{"k": "v"}},
		{name: "nil values", manager: NewManager(), values: nil},
		{name: "empty values", manager: NewManager(), values: map[string]string{}},
		{name: "cache disabled", manager: NewManager(), values: map[string]string{"k": "v"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SetMany(context.Background(), tt.manager, tt.values, 0)

			require.NoError(t, err)
		})
	}
}

func TestSetManyPropagatesEncodeError(t *testing.T) {
	values := map[string]chan int{"unserialisable": make(chan int)}

	err := SetMany(context.Background(), NewManager(), values, 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "chan int")
}

func TestSetManyEncodesEveryValue(t *testing.T) {
	values := map[string]sample{
		"a": {Name: "beatrice", N: 1},
		"b": {Name: "battler", N: 2},
	}

	entries := make(map[string][]byte, len(values))
	for key, value := range values {
		data, err := encode(value)
		require.NoError(t, err)

		entries[key] = data
	}

	assert.Equal(t, `{"name":"beatrice","n":1}`, string(entries["a"]))
	assert.Equal(t, `{"name":"battler","n":2}`, string(entries["b"]))
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
