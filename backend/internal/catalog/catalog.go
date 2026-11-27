// Package catalog is the in-memory copy of the network and the scenarios. It
// lets the API run without a database (demos, tests and the first boot) while
// the PostGIS store remains the source of truth in production.
package catalog

import "wayraflow/backend/internal/domain"

// Network returns the eight critical nodes and the three corridors from README
// sections 3.3 and 3.4, with the same coordinates as migration 0002.
func Network() domain.Network {
	return domain.Network{
		City: domain.City,
		Nodes: []domain.Node{
			{ID: "N1", Name: "Av. Circunvalación Norte × Av. Independencia (PE-3S)", OSMNodeID: 257687118, Signalized: true, Lon: -70.13584, Lat: -15.48129},
			{ID: "N2", Name: "Av. Circunvalación Norte (PE-34A) × Av. Ferrocarril", OSMNodeID: 1282294742, Signalized: true, Lon: -70.14279, Lat: -15.48646},
			{ID: "N3", Name: "Av. Circunvalación Norte × Av. Circunvalación Este × Av. Triunfo", OSMNodeID: 1281999136, Signalized: true, Lon: -70.12695, Lat: -15.48068},
			{ID: "N4", Name: "Av. Circunvalación Este (PE-3S) × Av. Huancané (PE-34H)", OSMNodeID: 257687482, Signalized: true, Lon: -70.12446, Lat: -15.48482},
			{ID: "N5", Name: "Av. Circunvalación Este × Jr. Huancané", OSMNodeID: 443135956, Signalized: false, Lon: -70.12342, Lat: -15.48803},
			{ID: "N6", Name: "Autopista Mártires del 4 de Noviembre (PE-3S) × Av. Tacna", OSMNodeID: 2420987313, Signalized: true, Lon: -70.12799, Lat: -15.50306},
			{ID: "N7", Name: "Autopista Mártires del 4 de Noviembre × Av. Américas", OSMNodeID: 3226623702, Signalized: true, Lon: -70.12502, Lat: -15.51141},
			{ID: "N8", Name: "Av. Circunvalación Oeste (PE-34A) × Héroes del Pacífico", OSMNodeID: 255459726, Signalized: false, Lon: -70.14661, Lat: -15.49811},
		},
		Corridors: []domain.Corridor{
			{ID: "C1", Name: "Circunvalación Norte", Nodes: []domain.CorridorNode{
				{NodeID: "N2", Position: 0},
				{NodeID: "N1", Position: 1, DistanceM: 941},
				{NodeID: "N3", Position: 2, DistanceM: 955},
			}},
			{ID: "C2", Name: "Circunvalación Este", Nodes: []domain.CorridorNode{
				{NodeID: "N3", Position: 0},
				{NodeID: "N4", Position: 1, DistanceM: 532},
				{NodeID: "N5", Position: 2, DistanceM: 374},
			}},
			{ID: "C3", Name: "Eje sur PE-3S", Nodes: []domain.CorridorNode{
				{NodeID: "N6", Position: 0},
				{NodeID: "N7", Position: 1, DistanceM: 984},
			}},
		},
	}
}

// Scenarios returns the three predefined scenarios of README section 6.
func Scenarios() []domain.Scenario {
	return []domain.Scenario{
		{
			ID:          "base",
			Name:        "Tráfico base",
			Description: "Hora de análisis con la demanda observada como supuesto.",
			Demand: domain.Demand{
				Corridors: map[string]float64{"C1": 700, "C2": 600, "C3": 650, "N8": 400},
				Mix:       domain.Mix{L5: 0.35, M1: 0.35, M2: 0.20, N: 0.10},
			},
		},
		{
			ID:          "hora-punta-regional",
			Name:        "Hora punta entrada/salida regional",
			Description: "Pico de acceso a la ciudad por los tres corredores.",
			Demand: domain.Demand{
				Corridors: map[string]float64{"C1": 1000, "C2": 750, "C3": 1000, "N8": 650},
				Mix:       domain.Mix{L5: 0.25, M1: 0.30, M2: 0.30, N: 0.15},
			},
		},
		{
			ID:          "dia-de-feria",
			Name:        "Día de feria (domingo)",
			Description: "Mayor peso de mototaxis y flujo hacia el mercado.",
			Demand: domain.Demand{
				Corridors: map[string]float64{"C1": 850, "C2": 1050, "C3": 800, "N8": 450},
				Mix:       domain.Mix{L5: 0.45, M1: 0.25, M2: 0.20, N: 0.10},
			},
		},
	}
}

// Scenario returns a scenario by id.
func Scenario(id string) (domain.Scenario, bool) {
	for _, s := range Scenarios() {
		if s.ID == id {
			return s, true
		}
	}
	return domain.Scenario{}, false
}
