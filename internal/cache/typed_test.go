package cache

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/logger"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
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
		values  map[Key]string
	}{
		{name: "nil manager", manager: nil, values: map[Key]string{untimed.Key("k"): "v"}},
		{name: "nil values", manager: NewManager(), values: nil},
		{name: "empty values", manager: NewManager(), values: map[Key]string{}},
		{name: "cache disabled", manager: NewManager(), values: map[Key]string{untimed.Key("k"): "v"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manager.SetMany(context.Background(), tt.values)

			require.NoError(t, err)
		})
	}
}

func TestSetManyPropagatesEncodeError(t *testing.T) {
	values := map[Key]chan int{untimed.Key("unserialisable"): make(chan int)}

	err := NewManager().SetMany(context.Background(), values)

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

func newLoadManager() (*Manager, Key) {
	return NewManager(engines.NewInMemory(0)), timed.Key("gold")
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	buf := new(bytes.Buffer)
	previousLogger := logger.Log
	previousLevel := zerolog.GlobalLevel()

	logger.Log = zerolog.New(buf)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	t.Cleanup(func() {
		logger.Log = previousLogger
		zerolog.SetGlobalLevel(previousLevel)
	})

	return buf
}

func TestLoadReturnsCachedValueWithoutCallingLoader(t *testing.T) {
	m, key := newLoadManager()
	ctx := t.Context()

	require.NoError(t, m.Set(ctx, key, sample{Name: "beatrice", N: 1}))

	calls := 0
	load := func(context.Context) (sample, error) {
		calls++

		return sample{Name: "battler", N: 2}, nil
	}

	got, err := m.Load(ctx, key, load)

	require.NoError(t, err)
	assert.Equal(t, sample{Name: "beatrice", N: 1}, got)
	assert.Zero(t, calls)
}

func TestLoadStoresLoadedValueOnMiss(t *testing.T) {
	m, key := newLoadManager()
	ctx := t.Context()

	want := sample{Name: "beatrice", N: 7}
	load := func(context.Context) (sample, error) {
		return want, nil
	}

	got, err := m.Load(ctx, key, load)
	require.NoError(t, err)
	assert.Equal(t, want, got)

	cached, err := m.Get[sample](ctx, key)
	require.NoError(t, err)
	assert.Equal(t, want, cached)
}

func TestLoadStoresWithTheNamespaceTTL(t *testing.T) {
	// given an engine that misses and then expects the write
	stub := newEngineMock(t, "stub", true)
	stub.EXPECT().Get(testifymock.Anything, "timed:gold").Return(nil, ErrMiss).Once()
	stub.EXPECT().Set(testifymock.Anything, "timed:gold", []byte(`{"name":"beatrice","n":7}`), time.Minute).Return(nil).Once()

	load := func(context.Context) (sample, error) {
		return sample{Name: "beatrice", N: 7}, nil
	}

	// when
	got, err := NewManager(stub).Load(t.Context(), timed.Key("gold"), load)

	// then the write carried the key's own ttl, which the mock enforces
	require.NoError(t, err)
	assert.Equal(t, sample{Name: "beatrice", N: 7}, got)
}

func TestLoadLogsAFailedWriteAndStillReturnsTheLoadedValue(t *testing.T) {
	// given an engine that misses and then fails the write
	logs := captureLogs(t)
	writeErr := errors.New("valkey went away")

	stub := newEngineMock(t, "stub", true)
	stub.EXPECT().Get(testifymock.Anything, "timed:gold").Return(nil, ErrMiss).Once()
	stub.EXPECT().Set(testifymock.Anything, "timed:gold", testifymock.Anything, time.Minute).Return(writeErr).Once()

	want := sample{Name: "beatrice", N: 7}
	load := func(context.Context) (sample, error) {
		return want, nil
	}

	// when
	got, err := NewManager(stub).Load(t.Context(), timed.Key("gold"), load)

	// then the caller still gets its value, and the failed write is reported rather than swallowed
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Contains(t, logs.String(), `"level":"error"`)
	assert.Contains(t, logs.String(), writeErr.Error())
	assert.Contains(t, logs.String(), `"key":"timed:gold"`)
}

func TestLoadPropagatesLoaderErrorAndCachesNothing(t *testing.T) {
	m, key := newLoadManager()
	ctx := t.Context()

	wantErr := errors.New("the golden land is closed")
	load := func(context.Context) (sample, error) {
		return sample{Name: "battler", N: 2}, wantErr
	}

	got, err := m.Load(ctx, key, load)

	require.ErrorIs(t, err, wantErr)
	assert.Zero(t, got)

	_, err = m.Get[sample](ctx, key)
	assert.ErrorIs(t, err, ErrMiss)
}

func TestLoadFallsBackToLoaderWithoutManager(t *testing.T) {
	var m *Manager

	want := sample{Name: "beatrice", N: 3}
	load := func(context.Context) (sample, error) {
		return want, nil
	}

	got, err := m.Load(t.Context(), untimed.Key("gold"), load)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}
