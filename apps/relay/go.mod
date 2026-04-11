module github.com/isaacwallace123/cortex/apps/relay

go 1.25.0

require (
	github.com/isaacwallace123/cortex/services/courier v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.64.0
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
)

replace github.com/isaacwallace123/cortex/services/courier => ../../services/courier
