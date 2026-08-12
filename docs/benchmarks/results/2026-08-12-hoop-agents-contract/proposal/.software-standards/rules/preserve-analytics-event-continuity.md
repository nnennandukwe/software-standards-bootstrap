# Preserve analytics event continuity

When a route is added, duplicated, or superseded for an already tracked action, preserve its analytics event or document an intentional successor. Use constants from `gateway/analytics/events.go`; never replace them with event-name string literals or silently remove the final emission site.
