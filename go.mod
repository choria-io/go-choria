module github.com/choria-io/go-choria

go 1.16

// shuts up vulnerability alerts that did not affect this project
replace github.com/opencontainers/runc v0.0.0-20161107232042-8779fa57eb4a => github.com/opencontainers/runc v1.0.2

require (
	github.com/AlecAivazis/survey/v2 v2.3.2
	github.com/Freman/eventloghook v0.0.0-20191003051739-e4d803b6b48b
	github.com/aelsabbahy/goss v0.3.16
	github.com/antonmedv/expr v1.9.0
	github.com/awesome-gocui/gocui v1.0.1
	github.com/brutella/hc v1.2.4
	github.com/choria-io/go-updater v0.0.3
	github.com/cloudevents/sdk-go/v2 v2.6.1
	github.com/fatih/color v1.13.0
	github.com/ghodss/yaml v1.0.0
	github.com/gofrs/uuid v4.1.0+incompatible
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.5.6
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/gosuri/uilive v0.0.4 // indirect
	github.com/gosuri/uiprogress v0.0.1
	github.com/guptarohit/asciigraph v0.5.2
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/looplab/fsm v0.3.0
	github.com/miekg/pkcs11 v1.0.3
	github.com/mitchellh/mapstructure v1.4.2
	github.com/nats-io/jsm.go v0.0.27-0.20211006163108-9aae04fb57e9
	github.com/nats-io/nats-server/v2 v2.6.3-0.20211026165144-1441d2597749
	github.com/nats-io/nats-streaming-server v0.23.0 // indirect
	github.com/nats-io/nats.go v1.13.1-0.20211018182449-f2416a8b1483
	github.com/nats-io/stan.go v0.10.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.16.0
	github.com/open-policy-agent/opa v0.33.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/tidwall/gjson v1.10.2
	github.com/tidwall/pretty v1.2.0
	github.com/xeipuuv/gojsonschema v1.2.0
	go.uber.org/atomic v1.9.0
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/net v0.0.0-20211020060615-d418f374d309 // indirect
	golang.org/x/sys v0.0.0-20211025201205-69cdffdb9359
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	golang.org/x/tools v0.1.7
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	rsc.io/goversion v1.2.0
)
