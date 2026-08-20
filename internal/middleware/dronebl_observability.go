package middleware

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	droneblChecks = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dronebl_checks_total",
			Help: "DroneBL verdicts served to requests, by outcome.",
		},
		[]string{"outcome"},
	)
	droneblBlockedClasses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dronebl_blocked_classes_total",
			Help: "DroneBL listing classes seen on blocked requests.",
		},
		[]string{"class"},
	)
)

func init() {
	prometheus.MustRegister(droneblChecks, droneblBlockedClasses)
}

func recordBlockedClasses(classes []int) {
	for _, class := range classes {
		droneblBlockedClasses.WithLabelValues(strconv.Itoa(class)).Inc()
	}
}
