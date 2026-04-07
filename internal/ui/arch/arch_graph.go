package arch

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vibe-menu/internal/manifest"
)

// ── Graph building ────────────────────────────────────────────────────────────

func buildEnvInfoMap(mf *manifest.Manifest) map[string]string {
	m := map[string]string{}
	for _, env := range mf.Infra.Environments {
		if env.Name == "" {
			continue
		}
		var parts []string
		if env.ComputeEnv != "" {
			parts = append(parts, env.ComputeEnv)
		}
		if env.CloudProvider != "" {
			parts = append(parts, env.CloudProvider)
		}
		if env.Orchestrator != "" && env.Orchestrator != env.ComputeEnv {
			parts = append(parts, env.Orchestrator)
		}
		m[env.Name] = strings.Join(parts, " · ")
	}
	return m
}

// buildConfigLabelMap maps each StackConfig name to its "Language · Framework" label.
func buildConfigLabelMap(mf *manifest.Manifest) map[string]string {
	m := map[string]string{}
	for _, sc := range mf.Backend.StackConfigs {
		if sc.Name == "" {
			continue
		}
		label := sc.Language
		if sc.Framework != "" {
			label += " · " + sc.Framework
		}
		if sc.LanguageVersion != "" {
			label += " " + sc.LanguageVersion
		}
		m[sc.Name] = label
	}
	return m
}

func buildArchGraph(mf *manifest.Manifest) ([]archNode, []archEdge) {
	var nodes []archNode
	var edges []archEdge

	// Frontend — label only
	if mf.Frontend.Tech != nil && mf.Frontend.Tech.Language != "" {
		tech := mf.Frontend.Tech
		label := tech.Language
		if tech.Framework != "" {
			label += " / " + tech.Framework
		}
		nodes = append(nodes, archNode{id: "frontend", kind: archFrontend, label: label})
	}

	// API Gateway — sits between frontend and backend services.
	// gwServices tracks which service names are routed through the gateway.
	// If Endpoints is empty all services are routed; otherwise only the listed ones.
	hasGateway := mf.Backend.APIGateway != nil && mf.Backend.APIGateway.Technology != ""
	gwServices := map[string]bool{}
	if hasGateway {
		gw := mf.Backend.APIGateway
		label := gw.Technology
		if label == "" {
			label = "API Gateway"
		}
		nodes = append(nodes, archNode{
			id:          "gateway",
			kind:        archAPIGateway,
			label:       label,
			environment: gw.Environment,
		})
		if gw.Endpoints == "" {
			// No restriction: every service routes through the gateway.
			for _, svc := range mf.Backend.Services {
				if svc.Name != "" {
					gwServices[svc.Name] = true
				}
			}
		} else {
			// Only services with a listed endpoint route through the gateway;
			// the rest get a direct frontend→service edge.
			for _, epName := range splitTrimComma(gw.Endpoints) {
				ep := findEndpoint(mf, epName)
				if ep != nil && ep.ServiceUnit != "" {
					gwServices[ep.ServiceUnit] = true
				}
			}
		}
	}

	// Message broker — shown between services and databases when configured.
	hasBroker := mf.Backend.Messaging != nil && mf.Backend.Messaging.BrokerTech != ""
	if hasBroker {
		msg := mf.Backend.Messaging
		nodes = append(nodes, archNode{
			id:          "broker",
			kind:        archBroker,
			label:       msg.BrokerTech,
			environment: msg.Environment,
		})
	}

	// Monolith arch: all services share the pillar-level MonolithEnvironment when
	// the individual service has no Environment set.
	isMonolith := mf.Backend.ArchPattern == manifest.ArchMonolith ||
		mf.Backend.ArchPattern == manifest.ArchModularMonolith

	// Services — label + configRef for grouping
	for _, svc := range mf.Backend.Services {
		if svc.Name == "" {
			continue
		}
		env := svc.Environment
		if env == "" && isMonolith {
			env = mf.Backend.MonolithEnvironment
		}
		nodes = append(nodes, archNode{
			id:          "svc." + svc.Name,
			kind:        archService,
			label:       svc.Name,
			environment: env,
			configRef:   svc.ConfigRef,
		})
	}

	// Databases — label only
	for _, db := range mf.Data.Databases {
		if db.Alias == "" {
			continue
		}
		nodes = append(nodes, archNode{
			id:          "db." + db.Alias,
			kind:        archDatabase,
			label:       db.Alias,
			environment: db.Environment,
		})
	}

	// File storages — appended before External APIs to match the visual column 4 layout:
	// DATA SOURCES → OBJECT STORAGE → EXTERNAL APIS (top to bottom).
	for i, fs := range mf.Data.FileStorages {
		if fs.Technology == "" {
			continue
		}
		fsID := fmt.Sprintf("fs.%d", i)
		label := fs.Technology
		if fs.Purpose != "" {
			label = fs.Technology + " · " + fs.Purpose
		}
		nodes = append(nodes, archNode{
			id:          fsID,
			kind:        archFileStorage,
			label:       label,
			environment: fs.Environment,
		})
	}

	// External APIs — label + edge from the calling service when configured
	for _, api := range mf.Contracts.ExternalAPIs {
		if api.Provider == "" {
			continue
		}
		nodes = append(nodes, archNode{
			id:    "ext." + api.Provider,
			kind:  archExternalAPI,
			label: api.Provider,
		})
	}

	// frontendEdgeTarget returns the correct hop for a frontend→service edge:
	// the gateway when that service is gateway-routed, or the service directly.
	frontendEdgeTarget := func(svcName string) string {
		if gwServices[svcName] {
			return "gateway"
		}
		return "svc." + svcName
	}

	// Edges: Frontend → [Gateway or Services] via component actions
	seen := map[string]bool{}
	for _, comp := range mf.Frontend.Components {
		for _, action := range comp.Actions {
			if action.Endpoint == "" {
				continue
			}
			ep := findEndpoint(mf, action.Endpoint)
			if ep == nil || ep.ServiceUnit == "" {
				continue
			}
			target := frontendEdgeTarget(ep.ServiceUnit)
			key := "frontend|" + target
			if seen[key] {
				continue
			}
			seen[key] = true
			eID := "edge." + strconv.Itoa(len(edges))
			edges = append(edges, archEdge{
				id:        eID,
				fromID:    "frontend",
				toID:      target,
				direction: dirUnidirectional,
			})
		}
	}
	// Implied edges when frontend is configured but has no explicit actions
	if len(edges) == 0 && mf.Frontend.Tech != nil && mf.Frontend.Tech.Language != "" {
		for _, ep := range mf.Contracts.Endpoints {
			if ep.ServiceUnit == "" {
				continue
			}
			target := frontendEdgeTarget(ep.ServiceUnit)
			key := "frontend|" + target
			if seen[key] {
				continue
			}
			seen[key] = true
			eID := "edge." + strconv.Itoa(len(edges))
			edges = append(edges, archEdge{
				id:        eID,
				fromID:    "frontend",
				toID:      target,
				direction: dirUnidirectional,
			})
		}
	}

	// Gateway → Services: one edge per gateway-routed service.
	if hasGateway {
		gwSeen := map[string]bool{}
		for _, svc := range mf.Backend.Services {
			if svc.Name == "" || gwSeen[svc.Name] || !gwServices[svc.Name] {
				continue
			}
			gwSeen[svc.Name] = true
			eID := "edge." + strconv.Itoa(len(edges))
			edges = append(edges, archEdge{
				id:        eID,
				fromID:    "gateway",
				toID:      "svc." + svc.Name,
				direction: dirUnidirectional,
			})
		}
	}

	// Broker edges: one edge per unique producer service (svc → broker)
	// and one per unique consumer service (broker → svc).
	if hasBroker {
		pubSeen := map[string]bool{}
		conSeen := map[string]bool{}
		for _, evt := range mf.Backend.Events {
			if evt.PublisherService != "" && !pubSeen[evt.PublisherService] {
				pubSeen[evt.PublisherService] = true
				eID := "edge." + strconv.Itoa(len(edges))
				edges = append(edges, archEdge{
					id:        eID,
					fromID:    "svc." + evt.PublisherService,
					toID:      "broker",
					label:     "produces",
					direction: dirUnidirectional,
				})
			}
			if evt.ConsumerService != "" && !conSeen[evt.ConsumerService] {
				conSeen[evt.ConsumerService] = true
				eID := "edge." + strconv.Itoa(len(edges))
				edges = append(edges, archEdge{
					id:        eID,
					fromID:    "broker",
					toID:      "svc." + evt.ConsumerService,
					label:     "consumes",
					direction: dirUnidirectional,
				})
			}
		}
	}

	// Edges: Service → Service (direction-aware)
	for _, link := range mf.Backend.CommLinks {
		if link.From == "" || link.To == "" {
			continue
		}
		dir := dirUnidirectional
		if strings.EqualFold(link.Direction, "bidirectional") {
			dir = dirBidirectional
		}
		eID := "edge." + strconv.Itoa(len(edges))
		edges = append(edges, archEdge{
			id:        eID,
			fromID:    "svc." + link.From,
			toID:      "svc." + link.To,
			label:     link.Protocol,
			direction: dir,
		})
	}

	// Edges: Service → Database (via RepositoryDef.TargetDB)
	dbEdgeSeen := map[string]bool{}
	for _, svc := range mf.Backend.Services {
		if svc.Name == "" {
			continue
		}
		for _, repo := range svc.Repositories {
			if repo.TargetDB == "" {
				continue
			}
			key := "svc." + svc.Name + "|db." + repo.TargetDB
			if dbEdgeSeen[key] {
				continue
			}
			dbEdgeSeen[key] = true
			eID := "edge." + strconv.Itoa(len(edges))
			edges = append(edges, archEdge{
				id:        eID,
				fromID:    "svc." + svc.Name,
				toID:      "db." + repo.TargetDB,
				direction: dirUnidirectional,
			})
		}
	}
	// Edges: Service → File Storage (via FileStorageDef.UsedByService)
	for i, fs := range mf.Data.FileStorages {
		if fs.Technology == "" || fs.UsedByService == "" {
			continue
		}
		fsID := fmt.Sprintf("fs.%d", i)
		eID := "edge." + strconv.Itoa(len(edges))
		edges = append(edges, archEdge{
			id:        eID,
			fromID:    "svc." + fs.UsedByService,
			toID:      fsID,
			direction: dirUnidirectional,
		})
	}

	// Edges: Service → External API (via ExternalAPIDef.CalledByService)
	extEdgeSeen := map[string]bool{}
	for _, api := range mf.Contracts.ExternalAPIs {
		if api.Provider == "" || api.CalledByService == "" {
			continue
		}
		key := "svc." + api.CalledByService + "|ext." + api.Provider
		if extEdgeSeen[key] {
			continue
		}
		extEdgeSeen[key] = true
		eID := "edge." + strconv.Itoa(len(edges))
		edges = append(edges, archEdge{
			id:        eID,
			fromID:    "svc." + api.CalledByService,
			toID:      "ext." + api.Provider,
			label:     api.Protocol,
			direction: dirUnidirectional,
		})
	}

	return nodes, edges
}

func findEndpoint(mf *manifest.Manifest, namePath string) *manifest.EndpointDef {
	for i := range mf.Contracts.Endpoints {
		if mf.Contracts.Endpoints[i].NamePath == namePath {
			return &mf.Contracts.Endpoints[i]
		}
	}
	return nil
}

// splitTrimComma splits a comma-separated string and trims whitespace from each part.
func splitTrimComma(s string) []string {
	var result []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
