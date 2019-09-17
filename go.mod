module github.com/choria-io/go-choria

go 1.12

replace golang.org/x/sys v0.0.0-20190726091711-fde4db37ae7a => golang.org/x/sys v0.0.0-20190813064441-fde4db37ae7a

require (
	github.com/AlecAivazis/survey/v2 v2.0.2
	github.com/choria-io/go-client v0.5.0
	github.com/choria-io/go-config v0.0.4-0.20190917140504-89b909578aaf
	github.com/choria-io/go-lifecycle v1.0.3
	github.com/choria-io/go-network-broker v1.3.1
	github.com/choria-io/go-protocol v1.3.2
	github.com/choria-io/go-puppet v0.0.1
	github.com/choria-io/go-security v0.5.0
	github.com/choria-io/go-srvcache v0.0.6
	github.com/choria-io/mcorpc-agent-provider v0.7.2-0.20190917141331-5d57b132d25b
	github.com/choria-io/provisioning-agent v0.5.0
	github.com/fatih/color v1.7.0
	github.com/ghodss/yaml v1.0.0
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang/mock v1.3.1
	github.com/google/shlex v0.0.0-20181106134648-c34317bd91bf
	github.com/gosuri/uiprogress v0.0.1
	github.com/guptarohit/asciigraph v0.4.1
	github.com/looplab/fsm v0.1.0
	github.com/nats-io/nats-server/v2 v2.0.4
	github.com/nats-io/nats-streaming-server v0.16.2 // indirect
	github.com/nats-io/nats.go v1.8.1
	github.com/nats-io/stan.go v0.5.0
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.4.2
	github.com/tidwall/gjson v1.3.2
	github.com/tidwall/pretty v1.0.0
	github.com/xeipuuv/gojsonschema v1.1.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	rsc.io/goversion v1.2.0
)
