module github.com/choria-io/go-choria

go 1.17

// fixes a vulnerability, this can be removed once all
// dependencies have also updated themselves
replace gopkg.in/yaml.v2 => gopkg.in/yaml.v3 v3.0.0

require (
	github.com/AlecAivazis/survey/v2 v2.3.5
	github.com/Freman/eventloghook v0.0.0-20191003051739-e4d803b6b48b
	github.com/Masterminds/semver v1.5.0
	github.com/adrg/xdg v0.4.0
	github.com/aelsabbahy/goss v0.3.18
	github.com/antonmedv/expr v1.9.0
	github.com/awesome-gocui/gocui v1.1.0
	github.com/brutella/hc v1.2.5
	github.com/choria-io/appbuilder v0.0.8
	github.com/choria-io/fisk v0.1.3
	github.com/cloudevents/sdk-go/v2 v2.10.1
	github.com/fatih/color v1.13.0
	github.com/ghodss/yaml v1.0.0
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang-jwt/jwt/v4 v4.4.1
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.5.8
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/gosuri/uiprogress v0.0.1
	github.com/guptarohit/asciigraph v0.5.5
	github.com/itchyny/gojq v0.12.8
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/looplab/fsm v0.3.0
	github.com/miekg/pkcs11 v1.1.1
	github.com/mitchellh/mapstructure v1.5.0
	github.com/nats-io/jsm.go v0.0.34-0.20220610113917-5a299917dacd
	github.com/nats-io/nats-server/v2 v2.8.5-0.20220607144903-0794eafa6f0e
	github.com/nats-io/nats.go v1.16.1-0.20220610202224-dcbb65a13ee9
	github.com/nats-io/natscli v0.0.34-0.20220617135711-98725765760e
	github.com/olekukonko/tablewriter v0.0.5
	github.com/onsi/ginkgo/v2 v2.1.4
	github.com/onsi/gomega v1.19.0
	github.com/open-policy-agent/opa v0.41.0
	github.com/prometheus/client_golang v1.12.2
	github.com/prometheus/client_model v0.2.0
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/tidwall/gjson v1.14.1
	github.com/tidwall/pretty v1.2.0
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/xlab/tablewriter v0.0.0-20160610135559-80b567a11ad5
	go.uber.org/atomic v1.9.0
	golang.org/x/crypto v0.0.0-20220525230936-793ad666bf5e
	golang.org/x/sys v0.0.0-20220615213510-4f61da869c0c
	golang.org/x/term v0.0.0-20220526004731-065cf7ba2467
	golang.org/x/text v0.3.7
	golang.org/x/tools v0.1.11
	rsc.io/goversion v1.2.0
)

require (
	github.com/HdrHistogram/hdrhistogram-go v1.1.2 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.2 // indirect
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/achanda/go-sysctl v0.0.0-20160222034550-6be7678c45d2 // indirect
	github.com/aelsabbahy/GOnetstat v0.0.0-20220505220511-31d79a98d9f2 // indirect
	github.com/aelsabbahy/go-ps v0.0.0-20201009164808-61c449472dcf // indirect
	github.com/agnivade/levenshtein v1.0.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/brutella/dnssd v1.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cheekybits/genny v1.0.0 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/emicklei/dot v0.16.0 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/gdamore/tcell/v2 v2.4.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gosuri/uilive v0.0.4 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/itchyny/timefmt-go v0.1.3 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.15.5 // indirect
	github.com/lucasb-eyer/go-colorful v1.0.3 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/miekg/dns v1.1.49 // indirect
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/sys/mountinfo v0.6.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nats-io/jwt/v2 v2.2.1-0.20220330180145-442af02fd36a // indirect
	github.com/nats-io/nkeys v0.3.0 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/oleiade/reflections v1.0.1 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/common v0.34.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.0.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/tadglines/go-pkgs v0.0.0-20140924210655-1f86682992f1 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tylertreat/hdrhistogram-writer v0.0.0-20210816161836-2e440612a39f // indirect
	github.com/vektah/gqlparser/v2 v2.4.4 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xiam/to v0.0.0-20191116183551-8328998fc0ed // indirect
	github.com/yashtewari/glob-intersection v0.1.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.17.0 // indirect
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4 // indirect
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	golang.org/x/time v0.0.0-20220411224347-583f2d630306 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/alessio/shellescape.v1 v1.0.0-20170105083845-52074bc9df61 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
