package middleware

import "github.com/prometheus/client_golang/prometheus"

var droneblChecks = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "dronebl_checks_total",
		Help: "DroneBL verdicts served to requests, by outcome.",
	},
	[]string{"outcome"},
)

func init() {
	prometheus.MustRegister(droneblChecks)
}
