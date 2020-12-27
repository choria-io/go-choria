package puppetdb

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/protocol"
)

type PuppetDB struct {
	fw      ChoriaFramework
	timeout time.Duration
	log     *logrus.Entry
}

type ChoriaFramework interface {
	Logger(string) *logrus.Entry
	Configuration() *config.Config
	PQLQueryCertNames(query string) ([]string, error)
}

var (
	stringIsRegex = regexp.MustCompile(`^/(.+)/$`)
	stringIsAlpha = regexp.MustCompile(`[[:alpha:]]`)
	stringIsPQL   = regexp.MustCompile(`^pql:\s*(.+)$`)
)

// New creates a new puppetdb discovery client
func New(fw ChoriaFramework) *PuppetDB {
	b := &PuppetDB{
		fw:      fw,
		timeout: time.Second * time.Duration(fw.Configuration().DiscoveryTimeout),
		log:     fw.Logger("puppetdb_discovery"),
	}

	return b
}

// Discover performs a broadcast discovery using the supplied filter
func (p *PuppetDB) Discover(ctx context.Context, opts ...DiscoverOption) (n []string, err error) {
	dopts := &dOpts{
		collective: p.fw.Configuration().MainCollective,
		discovered: []string{},
		filter:     protocol.NewFilter(),
		mu:         &sync.Mutex{},
		timeout:    p.timeout,
	}

	for _, opt := range opts {
		opt(dopts)
	}

	search, err := p.searchString(dopts.collective, dopts.filter)
	if err != nil {
		return nil, err
	}

	return p.fw.PQLQueryCertNames(search)
}

func (p *PuppetDB) searchString(collective string, filter *protocol.Filter) (string, error) {
	var queries []string

	queries = append(queries, p.discoverCollective(collective))
	queries = append(queries, p.discoverNodes(filter.Identity))
	queries = append(queries, p.discoverClasses(filter.Class))
	queries = append(queries, p.discoverAgents(filter.Agent))

	fq, err := p.discoverFacts(filter.Fact)
	if err != nil {
		return "", err
	}

	queries = append(queries, fq)

	var pqlParts []string
	for _, q := range queries {
		if q != "" {
			pqlParts = append(pqlParts, fmt.Sprintf("(%s)", q))
		}
	}

	pql := strings.Join(pqlParts, " and ")
	return fmt.Sprintf(`nodes[certname, deactivated] { %s }`, pql), nil
}

func (p *PuppetDB) discoverAgents(agents []string) string {
	if len(agents) == 0 {
		return ""
	}

	var pql []string

	for _, a := range agents {
		switch {
		case a == "scout" || a == "rpcutil":
			pql = append(pql, fmt.Sprintf("(%s or %s)", p.discoverClasses([]string{"choria::service"}), p.discoverClasses([]string{"mcollective::service"})))
		case stringIsRegex.MatchString(a):
			matches := stringIsRegex.FindStringSubmatch(a)
			pql = append(pql, fmt.Sprintf(`resources {type = "File" and tag ~ "mcollective_agent_.*?%s.*?_server"}`, p.stringRegex(matches[1])))
		default:
			pql = append(pql, fmt.Sprintf(`resources {type = "File" and tag = "mcollective_agent_%s_server"}`, a))
		}
	}

	return strings.Join(pql, " and ")
}

func (p *PuppetDB) stringRegex(s string) string {
	derived := s
	if stringIsRegex.MatchString(s) {
		parts := stringIsRegex.FindStringSubmatch(s)
		derived = parts[1]
	}

	re := ""
	for _, c := range []byte(derived) {
		if stringIsAlpha.MatchString(string(c)) {
			re += fmt.Sprintf("[%s%s]", strings.ToLower(string(c)), strings.ToUpper(string(c)))
		} else {
			re += string(c)
		}
	}

	return re
}

func (p *PuppetDB) capitalizePuppetResource(r string) string {
	parts := strings.Split(r, "::")
	var res []string

	for _, p := range parts {
		res = append(res, strings.Title(p))
	}

	return strings.Join(res, "::")
}

func (p *PuppetDB) discoverClasses(classes []string) string {
	if len(classes) == 0 {
		return ""
	}

	var pql []string

	for _, class := range classes {
		if stringIsRegex.MatchString(class) {
			parts := stringIsRegex.FindStringSubmatch(class)
			pql = append(pql, fmt.Sprintf(`resources {type = "Class" and title ~ "%s"}`, p.stringRegex(parts[1])))
		} else {
			pql = append(pql, fmt.Sprintf(`resources {type = "Class" and title = "%s"}`, p.capitalizePuppetResource(class)))
		}
	}

	return strings.Join(pql, " and ")
}

func (p *PuppetDB) discoverNodes(nodes []string) string {
	if len(nodes) == 0 {
		return ""
	}

	var pql []string

	for _, node := range nodes {
		switch {
		case stringIsPQL.MatchString(node):
			parts := stringIsPQL.FindStringSubmatch(node)
			pql = append(pql, fmt.Sprintf("certname in %s", parts[1]))

		case stringIsRegex.MatchString(node):
			parts := stringIsRegex.FindStringSubmatch(node)
			pql = append(pql, fmt.Sprintf(`certname ~ "%s"`, p.stringRegex(parts[1])))

		default:
			pql = append(pql, fmt.Sprintf(`certname = "%s"`, node))

		}
	}

	return strings.Join(pql, " or ")
}

func (p *PuppetDB) discoverCollective(f string) string {
	if f == "" {
		return ""
	}

	return fmt.Sprintf(`certname in inventory[certname] { facts.mcollective.server.collectives.match("\d+") = "%s" }`, f)
}

func (p *PuppetDB) isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func (p *PuppetDB) discoverFacts(facts []protocol.FactFilter) (string, error) {
	if len(facts) == 0 {
		return "", nil
	}

	var pql []string

	for _, f := range facts {
		switch f.Operator {
		case "=~":
			pql = append(pql, fmt.Sprintf(`inventory {facts.%s ~ "%s"}`, f.Fact, p.stringRegex(f.Value)))

		case "==":
			if f.Value == "true" || f.Value == "false" || p.isNumeric(f.Value) {
				pql = append(pql, fmt.Sprintf(`inventory {facts.%s = %s or facts.%s = "%s"}`, f.Fact, f.Value, f.Fact, f.Value))
			} else {
				pql = append(pql, fmt.Sprintf(`inventory {facts.%s = "%s"}`, f.Fact, f.Value))
			}

		case "!=":
			if f.Value == "true" || f.Value == "false" || p.isNumeric(f.Value) {
				pql = append(pql, fmt.Sprintf(`inventory {!(facts.%s = %s or facts.%s = "%s")}`, f.Fact, f.Value, f.Fact, f.Value))
			} else {
				pql = append(pql, fmt.Sprintf(`inventory {!(facts.%s = "%s")}`, f.Fact, f.Value))
			}

		case ">=", ">", "<=", "<":
			if !p.isNumeric(f.Value) {
				return "", fmt.Errorf("'%s' operator supports only numeric values", f.Operator)
			}

			pql = append(pql, fmt.Sprintf("inventory {facts.%s %s %s}", f.Fact, f.Operator, f.Value))

		default:
			return "", fmt.Errorf("do not know how to do fact comparisons using the '%s' operator with PuppetDB", f.Operator)

		}
	}

	return strings.Join(pql, " and "), nil
}
