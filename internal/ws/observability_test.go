package ws

import (
	"testing"

	"github.com/google/uuid"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRecordInbound_CountsByType(t *testing.T) {
	// given
	wsInboundMessages.Reset()

	// when
	recordInbound("typing", 42)
	recordInbound("typing", 41)
	recordInbound("join_room", 40)

	// then
	assert.Equal(t, 2.0, testutil.ToFloat64(wsInboundMessages.WithLabelValues("typing")))
	assert.Equal(t, 1.0, testutil.ToFloat64(wsInboundMessages.WithLabelValues("join_room")))
}

func TestRecordDropped_SeparatesAuthedFromAnon(t *testing.T) {
	// given
	wsInboundDropped.Reset()

	// when
	recordDropped(true)
	recordDropped(false)
	recordDropped(false)

	// then
	assert.Equal(t, 1.0, testutil.ToFloat64(wsInboundDropped.WithLabelValues("true")))
	assert.Equal(t, 2.0, testutil.ToFloat64(wsInboundDropped.WithLabelValues("false")))
}

func TestRecordConnection_GaugeRisesAndFalls(t *testing.T) {
	// given
	wsConnections.Reset()
	wsConnectionsTotal.Reset()

	// when
	recordConnectionOpened("test", true)
	recordConnectionOpened("test", true)
	recordConnectionClosed("test", true)

	// then
	assert.Equal(t, 1.0, testutil.ToFloat64(wsConnections.WithLabelValues("test", "true")))
	assert.Equal(t, 2.0, testutil.ToFloat64(wsConnectionsTotal.WithLabelValues("test", "true")))
}

func TestHubRegister_TracksConnectionsPerHub(t *testing.T) {
	// given
	wsConnections.Reset()
	main := NewHub("main")
	overlay := NewHub("overlay")

	// when
	main.Register(NewClient(uuid.New(), nil))
	overlay.Register(NewClient(uuid.New(), nil))
	overlay.Register(NewClient(uuid.New(), nil))

	// then
	assert.Equal(t, 1.0, testutil.ToFloat64(wsConnections.WithLabelValues("main", "true")))
	assert.Equal(t, 2.0, testutil.ToFloat64(wsConnections.WithLabelValues("overlay", "true")))
}

func TestHubUnregister_DoubleCallDoesNotCorruptGauge(t *testing.T) {
	// given
	wsConnections.Reset()
	hub := NewHub("main")
	client := NewClient(uuid.New(), nil)
	hub.Register(client)

	// when
	hub.Unregister(client)
	hub.Unregister(client)

	// then
	assert.Equal(t, 0.0, testutil.ToFloat64(wsConnections.WithLabelValues("main", "true")))
}

func TestNewHub_DefaultsToMain(t *testing.T) {
	// given / when
	hub := NewHub()

	// then
	assert.Equal(t, "main", hub.name)
}
