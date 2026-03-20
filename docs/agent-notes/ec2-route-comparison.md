# EC2 Route Comparison

## CompareRoutes (`lib/ec2.go`)

Compares two `types.Route` structs field by field. Returns `true` if they match.

### Blackhole Ignore Logic

When the `State` fields differ, `CompareRoutes` checks whether either route is an "ignored blackhole" via the `isIgnoredBlackhole` helper. A route qualifies if:
1. Its state is `RouteStateBlackhole`
2. It has a non-nil `VpcPeeringConnectionId`
3. That peering connection ID appears in the `blackholeIgnore` slice

This check is symmetric — it applies to both route arguments. (Fixed in T-410; previously only checked route1.)

### Call Site

Called from `cmd/drift.go` during drift detection:
- `route1` = actual AWS route (from `GetRouteTable`)
- `route2` = expected CloudFormation route (from `FilterRoutesByLogicalId`)
- `blackholeIgnore` = `settings.GetStringSlice("drift.ignore-blackholes")`

### Related Functions

- `GetRouteDestination` — returns the destination CIDR/prefix of a route
- `GetRouteTarget` — returns the target resource of a route
- `FilterRoutesByLogicalId` — extracts routes from a CFN template for a given logical resource
