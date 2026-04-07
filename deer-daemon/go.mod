module github.com/aspectrr/deer.sh/deer-daemon

go 1.24.0

toolchain go1.24.4

require (
	github.com/aspectrr/deer.sh/proto/gen/go v0.1.5
	github.com/aspectrr/deer.sh/shared v0.0.0
	github.com/glebarez/sqlite v1.11.0
	github.com/google/uuid v1.6.0
	github.com/posthog/posthog-go v1.10.0
	google.golang.org/grpc v1.79.1
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/gorm v1.31.1
)

require (
	github.com/anchore/go-lzo v0.1.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/diskfs/go-diskfs v1.7.0 // indirect
	github.com/djherbis/times v1.6.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/elliotwutingfeng/asciiset v0.0.0-20230602022725-51bbb787efab // indirect
	github.com/glebarez/go-sqlite v1.22.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.17 // indirect
	github.com/pkg/xattr v0.4.9 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/segmentio/kafka-go v0.4.47 // indirect
	github.com/sirupsen/logrus v1.9.4-0.20230606125235-dd1b4c2e81af // indirect
	github.com/ulikunitz/xz v0.5.11 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	modernc.org/libc v1.55.3 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
	modernc.org/sqlite v1.33.1 // indirect
)

replace github.com/aspectrr/deer.sh/shared => ../shared

replace github.com/aspectrr/deer.sh/proto/gen/go => ../proto/gen/go
