SHELL := /bin/bash

.PHONY: proto ent infra-up infra-down go-test go-build frontend-install frontend-build release-dry-run dev-api dev-agent

proto:
	mkdir -p packages/proto/gen/go
	protoc -I packages/proto/proto \
		--go_out=packages/proto/gen/go --go_opt=paths=source_relative \
		--go-grpc_out=packages/proto/gen/go --go-grpc_opt=paths=source_relative \
		packages/proto/proto/rayip/control/v1/control.proto \
		packages/proto/proto/rayip/runtime/v1/runtime.proto

ent:
	go run entgo.io/ent/cmd/ent generate --feature sql/upsert ./services/api/ent/schema

infra-up:
	docker compose up -d postgres redis nats

infra-down:
	docker compose down

go-test:
	GOPROXY=https://goproxy.cn,direct go test ./...

go-build:
	GOPROXY=https://goproxy.cn,direct go build ./services/api/cmd/api ./services/node-agent/cmd/node-agent

frontend-install:
	pnpm install

frontend-build:
	pnpm typecheck && pnpm build

release-dry-run:
	bash scripts/build-runtime-release.sh

dev-api:
	GOPROXY=https://goproxy.cn,direct go run ./services/api/cmd/api

dev-agent:
	RAYIP_AGENT_NODE_CODE=local-home-001 GOPROXY=https://goproxy.cn,direct go run ./services/node-agent/cmd/node-agent
