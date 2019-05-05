module istio.io/bots

go 1.12

require (
	cloud.google.com/go v0.38.0
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-playground/webhooks v5.9.0+incompatible
	github.com/google/go-github/v25 v25.0.2
	github.com/gorilla/mux v1.7.1 // indirect
	github.com/howeyc/fsnotify v0.9.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/prometheus/client_golang v0.9.2 // indirect
	github.com/prometheus/prom2json v1.1.0 // indirect
	github.com/spf13/cobra v0.0.3
	github.com/spf13/viper v1.3.2
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.10.0 // indirect
	golang.org/x/oauth2 v0.0.0-20190402181905-9f3314589c9a
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4
	google.golang.org/api v0.4.0
	google.golang.org/grpc v1.20.1
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	istio.io/istio v0.0.0-20190508153008-fe4e23cd49a7
)

replace github.com/golang/glog => github.com/istio/glog v0.0.0-20190424172949-d7cfb6fa2ccd
