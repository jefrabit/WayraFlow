// Command bench prints the three strategies for every scenario. It is the
// reproducible measurement behind the numbers in the README and the demo.
package main

import (
	"fmt"
	"os"
	"time"

	"wayraflow/backend/internal/catalog"
	"wayraflow/backend/internal/domain"
	"wayraflow/backend/internal/optim"
	"wayraflow/backend/internal/sim"
)

func main() {
	net := catalog.Network()
	fmt.Printf("%-24s %-9s %8s %8s %8s %8s\n", "escenario", "estrategia", "espera_s", "cola_veh", "paradas", "co2_kg_h")
	for _, sc := range catalog.Scenarios() {
		fixed, webster, ga, err := optim.BuildAll(net, sc, 42, time.Now())
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", sc.ID, err)
			os.Exit(1)
		}
		results := map[domain.Strategy]domain.KPI{}
		for strategy, p := range map[domain.Strategy]domain.Plan{
			domain.StrategyFixed:   fixed,
			domain.StrategyWebster: webster,
			domain.StrategyGA:      ga,
		} {
			res, err := sim.Run(net, sc, p, sim.DefaultHorizonS)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s/%s: %v\n", sc.ID, strategy, err)
				os.Exit(1)
			}
			results[strategy] = res.City
			fmt.Printf("%-24s %-9s %8.2f %8.2f %8.3f %8.1f\n",
				sc.ID, strategy, res.City.AvgDelayS, res.City.MaxQueueVeh, res.City.StopsPerVeh, res.City.CO2KgH)
		}
		base := results[domain.StrategyFixed].AvgDelayS
		if base > 0 {
			fmt.Printf("%-24s %-9s %7.1f%%\n\n", sc.ID, "mejora", (base-results[domain.StrategyGA].AvgDelayS)/base*100)
		}
	}
}
