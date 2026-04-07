package arch

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vibe-menu/internal/manifest"
)

// buildAllInfo constructs detail lines for every node AND edge, stored in one map.
func buildAllInfo(mf *manifest.Manifest, nodes []archNode, edges []archEdge) map[string][]string {
	outEdges := map[string][]archEdge{}
	inEdges := map[string][]archEdge{}
	for _, e := range edges {
		outEdges[e.fromID] = append(outEdges[e.fromID], e)
		inEdges[e.toID] = append(inEdges[e.toID], e)
	}

	info := map[string][]string{}
	for _, n := range nodes {
		info[n.id] = buildSingleNodeInfo(mf, n, outEdges[n.id], inEdges[n.id])
	}
	for _, e := range edges {
		info[e.id] = buildEdgeInfo(mf, e)
	}
	return info
}

// epRow holds one endpoint's display columns for aligned formatting.
type epRow struct {
	method string
	path   string
	reqDTO string
	resDTO string
}

// formatEpRows formats a slice of endpoint rows as aligned table lines.
// If any row has DTOs, all rows include the DTO column (using "—" for absent values).
// Columns: method (dynamic width)  path (dynamic width, capped)  reqDTO → resDTO
func formatEpRows(rows []epRow) []string {
	if len(rows) == 0 {
		return nil
	}
	methodW := 0
	pathW := 0
	hasDTOs := false
	for _, r := range rows {
		if len(r.method) > methodW {
			methodW = len(r.method)
		}
		if len(r.path) > pathW {
			pathW = len(r.path)
		}
		if r.reqDTO != "" || r.resDTO != "" {
			hasDTOs = true
		}
	}
	const maxPathW = 40
	if pathW > maxPathW {
		pathW = maxPathW
	}

	var out []string
	for _, r := range rows {
		if !hasDTOs {
			out = append(out, fmt.Sprintf("    %-*s  %s", methodW, r.method, r.path))
			continue
		}
		req := r.reqDTO
		if req == "" {
			req = "—"
		}
		res := r.resDTO
		if res == "" {
			res = "—"
		}
		out = append(out, fmt.Sprintf("    %-*s  %-*s  %s → %s", methodW, r.method, pathW, r.path, req, res))
	}
	return out
}

// buildEdgeInfo builds the detail lines for a selected communication path.
func buildEdgeInfo(mf *manifest.Manifest, e archEdge) []string {
	var lines []string

	if e.fromID == "frontend" && e.toID == "gateway" {
		gw := mf.Backend.APIGateway
		if gw != nil && gw.Endpoints != "" {
			var rows []epRow
			for _, epName := range splitTrimComma(gw.Endpoints) {
				epDef := findEndpoint(mf, epName)
				if epDef == nil {
					continue
				}
				method := epDef.HTTPMethod
				if method == "" {
					method = "[" + epDef.Protocol + "]"
				}
				rows = append(rows, epRow{method: method, path: epDef.NamePath,
					reqDTO: epDef.RequestDTO, resDTO: epDef.ResponseDTO})
			}
			if len(rows) > 0 {
				lines = append(lines, "  Gateway-routed endpoints:")
				lines = append(lines, formatEpRows(rows)...)
			}
		} else {
			lines = append(lines, "  All service endpoints are routed through the gateway.")
		}
	} else if e.fromID == "frontend" && strings.HasPrefix(e.toID, "svc.") {
		svcName := strings.TrimPrefix(e.toID, "svc.")
		var rows []epRow
		for _, ep := range mf.Contracts.Endpoints {
			if ep.ServiceUnit != svcName {
				continue
			}
			method := ep.HTTPMethod
			if method == "" {
				method = "[" + ep.Protocol + "]"
			}
			rows = append(rows, epRow{
				method: method,
				path:   ep.NamePath,
				reqDTO: ep.RequestDTO,
				resDTO: ep.ResponseDTO,
			})
		}
		if len(rows) > 0 {
			lines = append(lines, "  Direct endpoints (bypasses gateway):")
			lines = append(lines, formatEpRows(rows)...)
		} else {
			lines = append(lines, "  No endpoints defined for this service.")
		}
	} else if strings.HasPrefix(e.fromID, "svc.") && e.toID == "broker" {
		svcName := strings.TrimPrefix(e.fromID, "svc.")
		var events []manifest.EventDef
		for _, evt := range mf.Backend.Events {
			if evt.PublisherService == svcName {
				events = append(events, evt)
			}
		}
		if len(events) > 0 {
			lines = append(lines, fmt.Sprintf("  Published events: (%d)", len(events)))
			for _, evt := range events {
				l := "    → " + evt.Name
				if evt.ConsumerService != "" {
					l += "  (consumed by: " + evt.ConsumerService + ")"
				}
				if evt.DTO != "" {
					l += "  [" + evt.DTO + "]"
				}
				lines = append(lines, l)
				if evt.Description != "" {
					lines = append(lines, "        "+evt.Description)
				}
			}
		} else {
			lines = append(lines, "  No events published by this service.")
		}
	} else if e.fromID == "broker" && strings.HasPrefix(e.toID, "svc.") {
		svcName := strings.TrimPrefix(e.toID, "svc.")
		var events []manifest.EventDef
		for _, evt := range mf.Backend.Events {
			if evt.ConsumerService == svcName {
				events = append(events, evt)
			}
		}
		if len(events) > 0 {
			lines = append(lines, fmt.Sprintf("  Consumed events: (%d)", len(events)))
			for _, evt := range events {
				l := "    ← " + evt.Name
				if evt.PublisherService != "" {
					l += "  (published by: " + evt.PublisherService + ")"
				}
				if evt.DTO != "" {
					l += "  [" + evt.DTO + "]"
				}
				lines = append(lines, l)
				if evt.Description != "" {
					lines = append(lines, "        "+evt.Description)
				}
			}
		} else {
			lines = append(lines, "  No events consumed by this service.")
		}
	} else if strings.HasPrefix(e.fromID, "svc.") && strings.HasPrefix(e.toID, "svc.") {
		fromSvc := strings.TrimPrefix(e.fromID, "svc.")
		toSvc := strings.TrimPrefix(e.toID, "svc.")
		found := false
		for _, link := range mf.Backend.CommLinks {
			if link.From != fromSvc || link.To != toSvc {
				continue
			}
			found = true
			if link.Protocol != "" {
				lines = append(lines, "  Protocol:    "+link.Protocol)
			}
			if link.Direction != "" {
				lines = append(lines, "  Direction:   "+link.Direction)
			}
			if link.Trigger != "" {
				lines = append(lines, "  Trigger:     "+link.Trigger)
			}
			if string(link.SyncAsync) != "" {
				lines = append(lines, "  Sync/Async:  "+string(link.SyncAsync))
			}
			if len(link.DTOs) > 0 {
				lines = append(lines, "  Request DTOs:")
				for _, dto := range link.DTOs {
					lines = append(lines, "    → "+dto)
				}
			}
			if len(link.ResponseDTOs) > 0 {
				lines = append(lines, "  Response DTOs:")
				for _, dto := range link.ResponseDTOs {
					lines = append(lines, "    ← "+dto)
				}
			}
			if len(link.ResiliencePatterns) > 0 {
				lines = append(lines, "  Resilience:  "+strings.Join(link.ResiliencePatterns, ", "))
			}
		}
		if !found {
			lines = append(lines, "  No communication details available.")
		}
	} else if strings.HasPrefix(e.fromID, "svc.") && strings.HasPrefix(e.toID, "fs.") {
		idxStr := strings.TrimPrefix(e.toID, "fs.")
		idx, err := strconv.Atoi(idxStr)
		if err == nil && idx >= 0 && idx < len(mf.Data.FileStorages) {
			fs := mf.Data.FileStorages[idx]
			if fs.Purpose != "" {
				lines = append(lines, "  Purpose:       "+fs.Purpose)
			}
			if fs.Access != "" {
				lines = append(lines, "  Access:        "+fs.Access)
			}
			if fs.MaxSize != "" {
				lines = append(lines, "  Max size:      "+fs.MaxSize)
			}
			if fs.AllowedTypes != "" {
				lines = append(lines, "  Allowed types: "+fs.AllowedTypes)
			}
			if fs.TTLMinutes != "" {
				lines = append(lines, "  TTL:           "+fs.TTLMinutes+" min")
			}
			if fs.Domains != "" {
				lines = append(lines, "  Domains:       "+fs.Domains)
			}
		}
	} else if strings.HasPrefix(e.fromID, "svc.") && strings.HasPrefix(e.toID, "ext.") {
		provider := strings.TrimPrefix(e.toID, "ext.")
		for _, api := range mf.Contracts.ExternalAPIs {
			if api.Provider != provider {
				continue
			}
			if api.Protocol != "" {
				lines = append(lines, "  Protocol:     "+api.Protocol)
			}
			if api.AuthMechanism != "" {
				lines = append(lines, "  Auth:         "+api.AuthMechanism)
			}
			if api.FailureStrategy != "" {
				lines = append(lines, "  On failure:   "+api.FailureStrategy)
			}
			if api.BaseURL != "" {
				lines = append(lines, "  Base URL:     "+api.BaseURL)
			}
			if api.RateLimit != "" {
				lines = append(lines, "  Rate limit:   "+api.RateLimit)
			}
			if cnt := len(api.Interactions); cnt > 0 {
				lines = append(lines, fmt.Sprintf("  Interactions: (%d)", cnt))
				for i, ia := range api.Interactions {
					if i >= 5 {
						lines = append(lines, fmt.Sprintf("    … +%d more", cnt-5))
						break
					}
					l := "    → " + ia.Name
					if ia.HTTPMethod != "" {
						l += "  " + ia.HTTPMethod
					}
					if ia.Path != "" {
						l += "  " + ia.Path
					}
					lines = append(lines, l)
				}
			}
			break
		}
	} else if strings.HasPrefix(e.fromID, "svc.") && strings.HasPrefix(e.toID, "db.") {
		svcName := strings.TrimPrefix(e.fromID, "svc.")
		dbAlias := strings.TrimPrefix(e.toID, "db.")
		var repos []manifest.RepositoryDef
		for _, svc := range mf.Backend.Services {
			if svc.Name != svcName {
				continue
			}
			for _, repo := range svc.Repositories {
				if repo.TargetDB == dbAlias {
					repos = append(repos, repo)
				}
			}
			break
		}
		if len(repos) == 0 {
			lines = append(lines, "  Implied connection (no repositories defined).")
		} else {
			lines = append(lines, fmt.Sprintf("  Repositories: (%d)", len(repos)))
			for _, repo := range repos {
				repoHeader := "    · " + repo.Name
				if repo.EntityRef != "" {
					repoHeader += "  [" + repo.EntityRef + "]"
				}
				lines = append(lines, repoHeader)
				for _, op := range repo.Operations {
					opLine := "        " + op.OpType + ": " + op.Name
					if op.ResultShape != "" {
						opLine += " → " + op.ResultShape
					}
					lines = append(lines, opLine)
				}
			}
		}
	}

	if len(lines) == 0 {
		lines = []string{"  No contract details available."}
	}
	return lines
}

func buildSingleNodeInfo(mf *manifest.Manifest, n archNode, outs, ins []archEdge) []string {
	var lines []string

	switch n.kind {
	case archFrontend:
		pages := mf.Frontend.Pages
		if len(pages) == 0 {
			lines = append(lines, "  No pages configured.")
		} else {
			// Compute max route width for aligned columns.
			maxRouteW := 0
			for _, page := range pages {
				if len(page.Route) > maxRouteW {
					maxRouteW = len(page.Route)
				}
			}
			const maxRouteCap = 35
			if maxRouteW > maxRouteCap {
				maxRouteW = maxRouteCap
			}

			lines = append(lines, fmt.Sprintf("  Pages: (%d)", len(pages)))
			for _, page := range pages {
				var pageLabel string
				if page.Name != "" && page.Name != page.Route {
					pageLabel = fmt.Sprintf("%-*s  %s", maxRouteW, page.Route, page.Name)
				} else {
					pageLabel = page.Route
				}
				if page.AuthRequired == "true" || page.AuthRequired == "yes" {
					pageLabel += "  [auth]"
				}
				lines = append(lines, "    "+pageLabel)
				if page.ComponentRefs != "" {
					for _, ref := range splitTrimComma(page.ComponentRefs) {
						lines = append(lines, "      · "+ref)
					}
				}
				if page.LinkedPages != "" {
					for _, link := range splitTrimComma(page.LinkedPages) {
						lines = append(lines, "      → "+link)
					}
				}
			}
		}

	case archAPIGateway:
		gw := mf.Backend.APIGateway
		if gw != nil {
			if gw.Technology != "" {
				lines = append(lines, "  Technology:  "+gw.Technology)
			}
			if gw.Routing != "" {
				lines = append(lines, "  Routing:     "+gw.Routing)
			}
			if gw.Features != "" {
				lines = append(lines, "  Features:    "+gw.Features)
			}
			if gw.Endpoints != "" {
				lines = append(lines, "  Endpoints:   "+gw.Endpoints)
			}
			if gw.Environment != "" {
				lines = append(lines, "  Environment: "+gw.Environment)
			}
			// List all services that route through this gateway
			if len(mf.Backend.Services) > 0 {
				lines = append(lines, "")
				lines = append(lines, fmt.Sprintf("  Routes to: (%d services)", len(mf.Backend.Services)))
				for _, svc := range mf.Backend.Services {
					if svc.Name != "" {
						lines = append(lines, "    → "+svc.Name)
					}
				}
			}
		}

	case archService:
		svcName := strings.TrimPrefix(n.id, "svc.")
		// Collect rows first to compute dynamic column width.
		var rows []epRow
		for _, ep := range mf.Contracts.Endpoints {
			if ep.ServiceUnit != svcName {
				continue
			}
			method := ep.HTTPMethod
			if method == "" {
				method = "[" + ep.Protocol + "]"
			}
			rows = append(rows, epRow{method: method, path: ep.NamePath})
		}
		if len(rows) > 0 {
			lines = append(lines, fmt.Sprintf("  Endpoints: (%d)", len(rows)))
			lines = append(lines, formatEpRows(rows)...)
		} else {
			lines = append(lines, "  No endpoints configured.")
		}

		// Job queues assigned to this service
		var jobQueues []manifest.JobQueueDef
		for _, jq := range mf.Backend.JobQueues {
			if jq.WorkerService == svcName {
				jobQueues = append(jobQueues, jq)
			}
		}
		if len(jobQueues) > 0 {
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("  Jobs: (%d)", len(jobQueues)))
			for _, jq := range jobQueues {
				jobLine := "    · " + jq.Name
				if jq.Technology != "" {
					jobLine += "  [" + jq.Technology + "]"
				}
				lines = append(lines, jobLine)
				if jq.Concurrency != "" {
					lines = append(lines, "      concurrency: "+jq.Concurrency)
				}
				if jq.RetryPolicy != "" {
					lines = append(lines, "      retry: "+jq.RetryPolicy)
				}
				for _, cj := range jq.CronJobs {
					schedule := cj.Schedule
					if schedule == "" {
						schedule = "—"
					}
					lines = append(lines, "      ⏰ "+cj.Name+"  ("+schedule+")")
				}
			}
		}

	case archBroker:
		msg := mf.Backend.Messaging
		if msg != nil {
			if msg.BrokerTech != "" {
				lines = append(lines, "  Technology:    "+msg.BrokerTech)
			}
			if msg.Deployment != "" {
				lines = append(lines, "  Deployment:    "+msg.Deployment)
			}
			if msg.Serialization != "" {
				lines = append(lines, "  Serialization: "+msg.Serialization)
			}
			if msg.Delivery != "" {
				lines = append(lines, "  Delivery:      "+msg.Delivery)
			}
		}
		if cnt := len(mf.Backend.Events); cnt > 0 {
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("  Events: (%d)", cnt))
			for i, evt := range mf.Backend.Events {
				if i >= 8 {
					lines = append(lines, fmt.Sprintf("    … +%d more", cnt-8))
					break
				}
				l := "    · " + evt.Name
				if evt.PublisherService != "" && evt.ConsumerService != "" {
					l += "  (" + evt.PublisherService + " → " + evt.ConsumerService + ")"
				} else if evt.PublisherService != "" {
					l += "  (pub: " + evt.PublisherService + ")"
				} else if evt.ConsumerService != "" {
					l += "  (sub: " + evt.ConsumerService + ")"
				}
				lines = append(lines, l)
			}
		}

	case archFileStorage:
		idxStr := strings.TrimPrefix(n.id, "fs.")
		idx, err := strconv.Atoi(idxStr)
		if err == nil && idx >= 0 && idx < len(mf.Data.FileStorages) {
			fs := mf.Data.FileStorages[idx]
			if fs.Purpose != "" {
				lines = append(lines, "  Purpose:      "+fs.Purpose)
			}
			if fs.Access != "" {
				lines = append(lines, "  Access:       "+fs.Access)
			}
			if fs.MaxSize != "" {
				lines = append(lines, "  Max size:     "+fs.MaxSize)
			}
			if fs.AllowedTypes != "" {
				lines = append(lines, "  Types:        "+fs.AllowedTypes)
			}
			if fs.TTLMinutes != "" {
				lines = append(lines, "  TTL (min):    "+fs.TTLMinutes)
			}
			if fs.Domains != "" {
				lines = append(lines, "  Domains:      "+fs.Domains)
			}
			if fs.UsedByService != "" {
				lines = append(lines, "  Used by:      "+fs.UsedByService)
			}
		}

	case archDatabase:
		alias := strings.TrimPrefix(n.id, "db.")
		var domainNames []string
		for _, domain := range mf.Data.Domains {
			if domain.Databases == "" {
				continue
			}
			for _, dbName := range splitTrimComma(domain.Databases) {
				if dbName == alias {
					domainNames = append(domainNames, domain.Name)
					break
				}
			}
		}
		if len(domainNames) > 0 {
			lines = append(lines, fmt.Sprintf("  Domains: (%d)", len(domainNames)))
			for _, name := range domainNames {
				lines = append(lines, "    · "+name)
			}
		} else {
			lines = append(lines, "  No domains assigned to this database.")
		}

	case archExternalAPI:
		provider := strings.TrimPrefix(n.id, "ext.")
		for _, api := range mf.Contracts.ExternalAPIs {
			if api.Provider != provider {
				continue
			}
			if api.Protocol != "" {
				lines = append(lines, "  Protocol:     "+api.Protocol)
			}
			if api.AuthMechanism != "" {
				lines = append(lines, "  Auth:         "+api.AuthMechanism)
			}
			if api.BaseURL != "" {
				lines = append(lines, "  Base URL:     "+api.BaseURL)
			}
			if api.RateLimit != "" {
				lines = append(lines, "  Rate limit:   "+api.RateLimit)
			}
			if api.FailureStrategy != "" {
				lines = append(lines, "  On failure:   "+api.FailureStrategy)
			}
			if cnt := len(api.Interactions); cnt > 0 {
				lines = append(lines, fmt.Sprintf("  Interactions: %d configured", cnt))
				for i, ia := range api.Interactions {
					if i >= 4 {
						lines = append(lines, fmt.Sprintf("    … +%d more", cnt-4))
						break
					}
					lines = append(lines, "    "+ia.Name)
				}
			}
			break
		}
	}

	// Connection summary — only for external API nodes.
	// Frontend and service connections are navigated directly via the edge selection.
	if n.kind == archExternalAPI && len(outs)+len(ins) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  Connections:")
		for _, e := range outs {
			arrow := "──▶"
			if e.direction == dirBidirectional {
				arrow = "◀──▶"
			}
			target := strings.TrimPrefix(e.toID, "svc.")
			target = strings.TrimPrefix(target, "db.")
			target = strings.TrimPrefix(target, "ext.")
			l := "    " + arrow + " " + target
			if e.label != "" {
				l += "  [" + e.label + "]"
			}
			lines = append(lines, l)
		}
		for _, e := range ins {
			if e.direction == dirBidirectional {
				continue
			}
			src := strings.TrimPrefix(e.fromID, "svc.")
			src = strings.TrimPrefix(src, "frontend")
			if src == "" {
				src = "frontend"
			}
			lines = append(lines, "    ◀── "+src)
		}
	}

	if len(lines) == 0 {
		lines = []string{"  No details available."}
	}
	return lines
}
