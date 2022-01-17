module github.com/choria-io/go-choria

go 1.16

// shuts up vulnerability alerts that did not affect this project
replace github.com/opencontainers/runc v0.0.0-20161107232042-8779fa57eb4a => github.com/opencontainers/runc v1.0.3

require (
	github.com/AlecAivazis/survey/v2 v2.3.2
	github.com/Freman/eventloghook v0.0.0-20191003051739-e4d803b6b48b
	github.com/aelsabbahy/goss v0.3.16
	github.com/antonmedv/expr v1.9.0
	github.com/awesome-gocui/gocui v1.1.0
	github.com/brutella/hc v1.2.4
	github.com/choria-io/go-updater v0.0.4-0.20211231123842-da243cbc508c
	github.com/cloudevents/sdk-go/v2 v2.8.0
	github.com/fatih/color v1.13.0
	github.com/ghodss/yaml v1.0.0
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/golang-jwt/jwt/v4 v4.2.0
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.5.6
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/gosuri/uiprogress v0.0.1
	github.com/guptarohit/asciigraph v0.5.2
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/looplab/fsm v0.3.0
	github.com/miekg/pkcs11 v1.1.1
	github.com/mitchellh/mapstructure v1.4.3
	github.com/nats-io/jsm.go v0.0.28-0.20220114120941-6b58848ff92e
	github.com/nats-io/nats-server/v2 v2.7.1-0.20220114031104-297b44394f92
	github.com/nats-io/nats.go v1.13.1-0.20220112203823-4db0fb0b7033
	github.com/nats-io/natscli v0.0.28
	github.com/olekukonko/tablewriter v0.0.5
	github.com/onsi/ginkgo/v2 v2.0.0
	github.com/onsi/gomega v1.17.0
	github.com/open-policy-agent/opa v0.36.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/tidwall/gjson v1.13.0
	github.com/tidwall/pretty v1.2.0
	github.com/xeipuuv/gojsonschema v1.2.0
	go.uber.org/atomic v1.9.0
	golang.org/x/crypto v0.0.0-20220112180741-5e0467b6c7ce
	golang.org/x/net v0.0.0-20220114011407-0dd24b26b47d // indirect
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	golang.org/x/tools v0.1.8
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	rsc.io/goversion v1.2.0
)
