module istio.io/bots

go 1.18

// Old version had no license
replace github.com/chzyer/logex => github.com/chzyer/logex v1.1.11-0.20170329064859-445be9e134b2

require (
	cloud.google.com/go/bigquery v1.8.0
	cloud.google.com/go/spanner v1.9.0
	cloud.google.com/go/storage v1.10.0
	github.com/eapache/channels v1.1.0
	github.com/google/go-cmp v0.5.1
	github.com/google/go-github/v26 v26.0.9
	github.com/gorilla/mux v1.7.1
	github.com/gorilla/websocket v1.4.1
	github.com/hashicorp/go-multierror v1.0.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/sendgrid/sendgrid-go v3.4.1+incompatible
	github.com/spf13/cobra v0.0.4
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/tools v0.0.0-20200814230902-9882f1d1823d
	google.golang.org/api v0.30.0
	google.golang.org/grpc v1.33.0-dev.0.20200828165940-d8ef479ab79a
	gotest.tools v2.2.0+incompatible
	istio.io/pkg v0.0.0-20190710182420-c26792dead42
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.0
	sigs.k8s.io/yaml v1.2.0
)

require (
	cloud.google.com/go v0.63.0 // indirect
	github.com/beorn7/perks v1.0.0 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-logr/logr v0.2.0 // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/googleapis/gax-go/v2 v2.0.5 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/howeyc/fsnotify v0.9.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v0.9.3 // indirect
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4 // indirect
	github.com/prometheus/common v0.4.1 // indirect
	github.com/prometheus/procfs v0.0.1 // indirect
	github.com/prometheus/prom2json v1.1.0 // indirect
	github.com/sendgrid/rest v2.4.1+incompatible // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.opencensus.io v0.22.4 // indirect
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.10.0 // indirect
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9 // indirect
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/mod v0.3.0 // indirect
	golang.org/x/net v0.0.0-20200707034311-ab3426394381 // indirect
	golang.org/x/sys v0.0.0-20200803210538-64077c9b5642 // indirect
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/genproto v0.0.0-20200815001618-f69a88009b70 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.2.8 // indirect
	k8s.io/klog/v2 v2.2.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.0.1 // indirect
)
