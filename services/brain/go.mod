module github.com/isaacwallace123/cortex/services/brain

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/isaacwallace123/cortex/pkg/observe v0.0.0-00010101000000-000000000000
	github.com/isaacwallace123/cortex/services/beacon v0.0.0-00010101000000-000000000000
	github.com/isaacwallace123/cortex/services/arsenal v0.0.0-00010101000000-000000000000
	github.com/isaacwallace123/cortex/services/atlas v0.0.0-00010101000000-000000000000
	github.com/isaacwallace123/cortex/services/compass v0.0.0-00010101000000-000000000000
	github.com/isaacwallace123/cortex/services/courier v0.0.0-00010101000000-000000000000
	github.com/isaacwallace123/cortex/services/crucible v0.0.0-00010101000000-000000000000
	github.com/isaacwallace123/cortex/services/forge v0.0.0-00010101000000-000000000000
	github.com/isaacwallace123/cortex/services/inference v0.0.0
	github.com/isaacwallace123/cortex/services/memory v0.0.0
	github.com/isaacwallace123/cortex/services/nerva v0.0.0-00010101000000-000000000000
	github.com/isaacwallace123/cortex/services/policy v0.0.0-00010101000000-000000000000
	github.com/isaacwallace123/cortex/services/shell v0.0.0-00010101000000-000000000000
	github.com/isaacwallace123/cortex/services/vault v0.0.0-00010101000000-000000000000
	github.com/prometheus/client_golang v1.23.2
	google.golang.org/grpc v1.64.0
	google.golang.org/protobuf v1.36.8
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
)

replace github.com/isaacwallace123/cortex/pkg/observe => ../../pkg/observe

replace github.com/isaacwallace123/cortex/services/beacon => ../beacon

replace github.com/isaacwallace123/cortex/services/arsenal => ../arsenal

replace github.com/isaacwallace123/cortex/services/compass => ../compass

replace github.com/isaacwallace123/cortex/services/atlas => ../atlas

replace github.com/isaacwallace123/cortex/services/courier => ../courier

replace github.com/isaacwallace123/cortex/services/crucible => ../crucible

replace github.com/isaacwallace123/cortex/services/inference => ../inference

replace github.com/isaacwallace123/cortex/services/memory => ../memory

replace github.com/isaacwallace123/cortex/services/forge => ../forge

replace github.com/isaacwallace123/cortex/services/nerva => ../nerva

replace github.com/isaacwallace123/cortex/services/policy => ../policy

replace github.com/isaacwallace123/cortex/services/shell => ../shell

replace github.com/isaacwallace123/cortex/services/vault => ../vault
