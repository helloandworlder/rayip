module github.com/rayip/rayip

go 1.26

require (
	github.com/gofiber/fiber/v3 v3.0.0-rc.3
	github.com/google/uuid v1.6.0
	github.com/nats-io/nats.go v1.47.0
	github.com/pressly/goose/v3 v3.26.0
	github.com/redis/go-redis/v9 v9.17.1
	github.com/spf13/viper v1.21.0
	go.uber.org/fx v1.24.0
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.79.3
	google.golang.org/protobuf v1.36.11
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	github.com/apernet/quic-go v0.59.1-0.20260217092621-db4786c77a22 // indirect
	github.com/cloudflare/circl v1.6.3 // indirect
	github.com/juju/ratelimit v1.0.2 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/miekg/dns v1.1.72 // indirect
	github.com/pires/go-proxyproto v0.11.0 // indirect
	github.com/refraction-networking/utls v1.8.3-0.20260301010127-aa6edf4b11af // indirect
	github.com/sagernet/sing v0.5.1 // indirect
	github.com/xtls/reality v0.0.0-20260322125925-9234c772ba8f // indirect
	golang.org/x/mod v0.33.0 // indirect
	golang.org/x/tools v0.42.0 // indirect
	lukechampine.com/blake3 v1.4.1 // indirect
)

require (
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gofiber/schema v1.6.0 // indirect
	github.com/gofiber/utils/v2 v2.0.0-rc.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.5 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.18.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/nats-io/nkeys v0.4.11 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sethvargo/go-retry v0.3.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tinylib/msgp v1.5.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.68.0 // indirect
	github.com/xtls/xray-core v0.0.0
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
)

replace github.com/xtls/xray-core => ./third_party/xray-core
