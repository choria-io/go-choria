package mcorpc

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server/discovery/classes"
	"github.com/choria-io/go-choria/server/discovery/facts"
	"github.com/choria-io/go-client/client"
	"github.com/choria-io/go-config"
	"github.com/sirupsen/logrus"
)

type policyMatcher interface {
	Set(caller string, actions string, facts string, classes string, groups map[string][]string)
	MatchesFacts(fw *choria.Framework, log *logrus.Entry) (bool, error)
	MatchesClasses(classesFile string, log *logrus.Entry) (bool, error)
	MatchesAction(act string) bool
	MatchesCallerID(id string) bool
	IsCompound(line string) bool
}

func actionPolicyAuthorize(req *Request, agent *Agent, log *logrus.Entry) bool {
	authz := &actionPolicy{
		cfg:     agent.Config,
		req:     req,
		agent:   agent,
		matcher: &actionPolicyPolicy{},
		groups:  make(map[string][]string),
		log: log.WithFields(logrus.Fields{
			"authorizer": "actionpolicy",
			"agent":      agent.Name(),
			"request":    req.RequestID,
		}),
	}

	err := authz.parseGroupFile("")
	if err != nil {
		authz.log.Errorf("failed to parse groups file: %s", err)
	}

	return authz.authorize()
}

type actionPolicy struct {
	cfg     *config.Config
	req     *Request
	agent   *Agent
	log     *logrus.Entry
	matcher policyMatcher
	groups  map[string][]string
}

func (a *actionPolicy) authorize() bool {
	policyFile, err := a.lookupPolicyFile()
	if err != nil {
		a.log.Errorf("Could not lookup policy files: %s", err)
		return false
	}

	if policyFile == "" {
		if a.allowUnconfigured() {
			a.log.Infof("Allowing unconfigured agent request after failing to find any suitable policy file")
			return true
		}

		a.log.Infof("Denying unconfigured agent request after failing to find any suitable policy file")
		return false
	}

	allowed, reason, err := a.evaluatePolicy(policyFile)
	if err != nil {
		a.log.Errorf("Authorizing request %s failed: %s", a.req.RequestID, err)
		return false
	}

	if !allowed {
		a.log.Infof("Denying request %s: %s", a.req.RequestID, reason)
		return false
	}

	return true
}

func (a *actionPolicy) evaluatePolicy(f string) (allowed bool, denyreason string, err error) {
	a.log.Debugf("Parsing policy %s", f)

	pf, err := os.Open(f)
	if err != nil {
		return false, "", err
	}
	defer pf.Close()

	commentRe := regexp.MustCompile(`^(#.*|\s*)$`)
	defaultRe := regexp.MustCompile(`^policy\s+default\s+(\w+)`)
	policyRe := regexp.MustCompile(`^(allow|deny)\t+(.+?)\t+(.+?)\t+(.+?)(\t+(.+?))*$`)
	allowed = a.allowUnconfigured()

	scanner := bufio.NewScanner(pf)
	for scanner.Scan() {
		line := scanner.Text()

		if commentRe.MatchString(line) {
			continue
		}

		if defaultRe.MatchString(line) {
			matched := defaultRe.FindStringSubmatch(line)
			if matched[1] == "allow" {
				a.log.Debugf("found default allow line: %s", line)
				allowed = true
			} else {
				a.log.Debugf("found default deny line: %s", line)
				allowed = false
			}

		} else if policyRe.MatchString(line) {
			matched := policyRe.FindStringSubmatch(line)
			if a.matcher.IsCompound(matched[4]) || a.matcher.IsCompound(matched[6]) {
				a.log.Warnf("Compound policy statements are not supported, skipping line: %s", line)
				continue
			}

			a.matcher.Set(matched[2], matched[3], matched[4], matched[6], a.groups)
			pmatch, err := a.checkRequestAgainstPolicy()
			if err != nil {
				return false, "", err
			}

			if pmatch {
				if matched[1] == "allow" {
					return true, "", nil
				}

				return false, fmt.Sprintf("Denying based on explicit 'deny' policy in %s", filepath.Base(f)), nil
			}

		} else {
			a.log.Warnf("invalid policy line: %s", line)
			continue
		}
	}

	err = scanner.Err()
	if err != nil {
		return false, "", err
	}

	if allowed {
		return allowed, "", nil
	}

	return allowed, fmt.Sprintf("Denying based on default policy in %s", filepath.Base(f)), nil
}

func (a *actionPolicy) checkRequestAgainstPolicy() (bool, error) {
	pol := a.matcher

	if !pol.MatchesCallerID(a.req.CallerID) {
		return false, nil
	}

	if !pol.MatchesAction(a.req.Action) {
		return false, nil
	}

	fw, ok := a.agent.Choria.(*choria.Framework)
	if !ok {
		return false, fmt.Errorf("could not obtain a choria framework instance")
	}

	factsMatched, err := pol.MatchesFacts(fw, a.log)
	if err != nil {
		return false, err
	}

	classesMatched, err := pol.MatchesClasses(a.cfg.ClassesFile, a.log)
	if err != nil {
		return false, err
	}

	return classesMatched && factsMatched, nil
}

func (a *actionPolicy) allowUnconfigured() bool {
	unconfigured, err := choria.StrToBool(a.cfg.Option("plugin.actionpolicy.allow_unconfiguredt", "n"))
	if err != nil {
		return false
	}

	return unconfigured
}

func (a *actionPolicy) shouldUseDefault() bool {
	enabled, err := choria.StrToBool(a.cfg.Option("plugin.actionpolicy.enable_default", "n"))
	if err != nil {
		return false
	}

	return enabled
}

func (a *actionPolicy) defaultPolicyFileName() string {
	return a.cfg.Option("plugin.actionpolicy.default_name", "default")
}

func (a *actionPolicy) lookupPolicyFile() (string, error) {
	agentPolicy := filepath.Join(filepath.Dir(a.cfg.ConfigFile), "policies", a.agent.Name()+".policy")

	a.log.Debugf("Looking up agent policy in %s", agentPolicy)
	if choria.FileExist(agentPolicy) {
		return agentPolicy, nil
	}

	if a.shouldUseDefault() {
		defaultPolicy := filepath.Join(filepath.Dir(a.cfg.ConfigFile), "policies", a.defaultPolicyFileName()+".policy")
		if choria.FileExist(defaultPolicy) {
			return defaultPolicy, nil
		}
	}

	return "", fmt.Errorf("no policy found for %s", a.agent.Name())
}

func (a *actionPolicy) parseGroupFile(gfile string) error {
	if gfile == "" {
		gfile = filepath.Join(filepath.Dir(a.cfg.ConfigFile), "policies", "groups")
	}

	if !choria.FileExist(gfile) {
		return nil
	}

	gf, err := os.Open(gfile)
	if err != nil {
		return err
	}
	defer gf.Close()

	commentRe := regexp.MustCompile(`^(#.*|\s*)$`)
	groupRe := regexp.MustCompile(`^([\w\.\-]+)$`)

	scanner := bufio.NewScanner(gf)
	for scanner.Scan() {
		line := scanner.Text()

		if commentRe.MatchString(line) {
			continue
		}

		parts := strings.Split(line, " ")
		if len(parts) < 2 {
			a.log.Errorf("invalid group line in %s: %s", gfile, line)
			continue
		}

		if !groupRe.MatchString(parts[0]) {
			a.log.Errorf("invalid group name in %s: %s", gfile, parts[0])
			continue
		}

		a.groups[parts[0]] = parts[1:]
	}

	err = scanner.Err()
	if err != nil {
		return err
	}

	return nil
}

type actionPolicyPolicy struct {
	caller  string
	actions string
	facts   string
	classes string
	groups  map[string][]string
}

func (p *actionPolicyPolicy) Set(caller string, actions string, facts string, classes string, groups map[string][]string) {
	p.caller = caller
	p.actions = actions
	p.facts = facts
	p.classes = classes
	p.groups = groups
}

func (p *actionPolicyPolicy) MatchesFacts(fw *choria.Framework, log *logrus.Entry) (bool, error) {
	if p.facts == "" {
		return false, fmt.Errorf("empty fact policy found")
	}

	if p.facts == "*" {
		return true, nil
	}

	if p.IsCompound(p.facts) {
		return false, fmt.Errorf("compound statements are not supported")
	}

	matches := [][3]string{}

	for _, f := range strings.Split(p.facts, " ") {
		filter, err := client.ParseFactFilterString(f)
		if err != nil {
			return false, fmt.Errorf("invlid fact matcher: %s", err)
		}

		matches = append(matches, [3]string{filter.Fact, filter.Operator, filter.Value})
	}

	if facts.Match(matches, fw, log) {
		return true, nil
	}

	return false, nil
}

func (p *actionPolicyPolicy) MatchesClasses(classesFile string, log *logrus.Entry) (bool, error) {
	if p.classes == "*" {
		return true, nil
	}

	if p.classes == "" {
		return false, fmt.Errorf("empty classes policy found")
	}

	if classesFile == "" {
		return false, fmt.Errorf("do not know how to resolve classes")
	}

	if p.IsCompound(p.classes) {
		return false, fmt.Errorf("compound statements are not supported")
	}

	factMatcher := regexp.MustCompile(`(.+)(<|>|=|<=|>=)(.+)`)
	for _, c := range strings.Split(p.classes, " ") {
		if factMatcher.MatchString(c) {
			return false, fmt.Errorf("fact found where class was expected")
		}
	}

	return classes.Match(strings.Split(p.classes, " "), classesFile, log), nil
}

func (p *actionPolicyPolicy) MatchesAction(act string) bool {
	if p.actions == "" {
		return false
	}

	if p.actions == "*" {
		return true
	}

	for _, a := range strings.Split(p.actions, " ") {
		if act == a {
			return true
		}
	}

	return false
}

func (p *actionPolicyPolicy) MatchesCallerID(id string) bool {
	if p.caller == "" {
		return false
	}

	if p.caller == "*" {
		return true
	}

	if p.isCallerInGroups(id) {
		return true
	}

	for _, c := range strings.Split(p.caller, " ") {
		if c == id {
			return true
		}
	}

	return false
}

// IsCompound checks if the string is a compound statement
func (p *actionPolicyPolicy) IsCompound(line string) bool {
	matcher := regexp.MustCompile(`^!|^not$|^or$|^and$|\(.+\)`)

	for _, l := range strings.Split(line, " ") {
		if matcher.MatchString(l) {
			return true
		}
	}

	return false
}

func (p *actionPolicyPolicy) isCallerInGroups(id string) bool {
	for _, g := range strings.Split(p.caller, " ") {
		group, ok := p.groups[g]
		if !ok {
			continue
		}

		for _, member := range group {
			if member == id {
				return true
			}
		}
	}

	return false
}
