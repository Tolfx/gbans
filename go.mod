module github.com/leighmacdonald/gbans

go 1.16

require (
	github.com/Depado/ginprom v1.7.1
	github.com/Masterminds/squirrel v1.5.0
	github.com/bwmarrin/discordgo v0.23.3-0.20210410202908-577e7dd4f6cc
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gin-gonic/gin v1.7.4
	github.com/go-playground/validator/v10 v10.9.0 // indirect
	github.com/golang-migrate/migrate/v4 v4.15.0-beta.3
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jackc/pgconn v1.10.0
	github.com/jackc/pgerrcode v0.0.0-20201024163028-a0d42d470451
	github.com/jackc/pgx/v4 v4.13.0
	github.com/jedib0t/go-pretty/v6 v6.2.4
	github.com/leighmacdonald/golib v1.1.0
	github.com/leighmacdonald/rcon v1.0.6
	github.com/leighmacdonald/steamid/v2 v2.2.0
	github.com/leighmacdonald/steamweb v0.0.0-20210803010711-64b0e363d418
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/microcosm-cc/bluemonday v1.0.15 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.4.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.30.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/rumblefrog/go-a2s v1.0.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/ugorji/go v1.2.6 // indirect
	github.com/yohcop/openid-go v1.0.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/net v0.0.0-20210825183410-e898025ed96a // indirect
	golang.org/x/sys v0.0.0-20210902050250-f475640dd07b // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/grpc v1.40.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/olahol/melody.v1 v1.0.0-20170518105555-d52139073376
)

replace (
	// Partial support for command permissions added
	github.com/bwmarrin/discordgo => github.com/leighmacdonald/discordgo v0.23.3-0.20210501231400-4a24b4e9205c
	// Supports iofs
	github.com/golang-migrate/migrate/v4 => github.com/leighmacdonald/migrate/v4 v4.14.2-0.20210504172520-d53881cff5a4
)
