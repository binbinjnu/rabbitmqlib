module go.slotsdev.info/server-group/rabbitmqlib

//module rabbitmqlib

go 1.18

require github.com/streadway/amqp v1.0.0

require go.slotsdev.info/server-group/gamelib v0.0.4

require (
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/gogf/gf/v2 v2.2.5 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/jonboulle/clockwork v0.3.0 // indirect
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible // indirect
	github.com/lestrrat-go/strftime v1.0.6 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	go.opentelemetry.io/otel v1.7.0 // indirect
	go.opentelemetry.io/otel/trace v1.7.0 // indirect
	golang.org/x/sys v0.3.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace go.slotsdev.info/server-group/gamelib v0.0.4 => go.slotsdev.info/server-group/gamelib.git v0.0.4
