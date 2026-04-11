module github.com/isaacwallace123/cortex/services/api

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/isaacwallace123/cortex/services/brain v0.0.0
	github.com/isaacwallace123/cortex/services/chat v0.0.0-00010101000000-000000000000
	github.com/isaacwallace123/cortex/services/compass v0.0.0-00010101000000-000000000000
	github.com/isaacwallace123/cortex/services/memory v0.0.0
	github.com/isaacwallace123/cortex/services/policy v0.0.0-00010101000000-000000000000
	github.com/isaacwallace123/cortex/services/sovereign v0.0.0-00010101000000-000000000000
	github.com/isaacwallace123/cortex/services/workspace v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.64.0
)

require (
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
)

replace github.com/isaacwallace123/cortex/services/brain => ../brain

replace github.com/isaacwallace123/cortex/services/chat => ../chat

replace github.com/isaacwallace123/cortex/services/compass => ../compass

replace github.com/isaacwallace123/cortex/services/memory => ../memory

replace github.com/isaacwallace123/cortex/services/policy => ../policy

replace github.com/isaacwallace123/cortex/services/sovereign => ../sovereign

replace github.com/isaacwallace123/cortex/services/workspace => ../workspace
