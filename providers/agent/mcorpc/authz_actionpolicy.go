// Copyright (c) 2020-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package mcorpc

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/filter"
	"github.com/choria-io/go-choria/filter/classes"
	"github.com/choria-io/go-choria/filter/facts"
	"github.com/choria-io/go-choria/internal/util"

	"github.com/sirupsen/logrus"
)

func actionPolicyAuthorize(req *Request, cfg *config.Config, log *logrus.Entry) bool {
	logger := log.WithFields(logrus.Fields{
		"authorizer": "actionpolicy",
		"agent":      req.Agent,
		"request":    req.RequestID,
	})

	authz := &actionPolicy{
		cfg:     cfg,
		req:     req,
		matcher: &actionPolicyPolicy{log: logger},
		groups:  make(map[string][]string),
		log:     logger,
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
	log     *logrus.Entry
	matcher *actionPolicyPolicy
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
	a.matcher.SetFile(f)

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

	factsMatched, err := pol.MatchesFacts(a.cfg, a.log)
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
	unconfigured, err := util.StrToBool(a.cfg.Option("plugin.actionpolicy.allow_unconfigured", "n"))
	if err != nil {
		return false
	}

	return unconfigured
}

func (a *actionPolicy) shouldUseDefault() bool {
	enabled, err := util.StrToBool(a.cfg.Option("plugin.actionpolicy.enable_default", "n"))
	if err != nil {
		return false
	}

	return enabled
}

func (a *actionPolicy) defaultPolicyFileName() string {
	return a.cfg.Option("plugin.actionpolicy.default_name", "default")
}

func (a *actionPolicy) lookupPolicyFile() (string, error) {
	agentPolicy := filepath.Join(filepath.Dir(a.cfg.ConfigFile), "policies", a.req.Agent+".policy")

	a.log.Debugf("Looking up agent policy in %s", agentPolicy)
	if util.FileExist(agentPolicy) {
		return agentPolicy, nil
	}

	if a.shouldUseDefault() {
		defaultPolicy := filepath.Join(filepath.Dir(a.cfg.ConfigFile), "policies", a.defaultPolicyFileName()+".policy")
		if util.FileExist(defaultPolicy) {
			return defaultPolicy, nil
		}
	}

	return "", fmt.Errorf("no policy found for %s", a.req.Agent)
}

func (a *actionPolicy) parseGroupFile(gfile string) error {
	if gfile == "" {
		gfile = filepath.Join(filepath.Dir(a.cfg.ConfigFile), "policies", "groups")
	}

	if !util.FileExist(gfile) {
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
	log     *logrus.Entry
	file    string
}

func (p *actionPolicyPolicy) Set(caller string, actions string, facts string, classes string, groups map[string][]string) {
	p.caller = caller
	p.actions = actions
	p.facts = facts
	p.classes = classes
	p.groups = groups
}

func (p *actionPolicyPolicy) MatchesFacts(cfg *config.Config, log *logrus.Entry) (bool, error) {
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
		filter, err := filter.ParseFactFilterString(f)
		if err != nil {
			return false, fmt.Errorf("invalid fact matcher: %s", err)
		}

		matches = append(matches, [3]string{filter.Fact, filter.Operator, filter.Value})
	}

	if facts.MatchFile(matches, cfg.FactSourceFile, log) {
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

	return classes.MatchFile(strings.Split(p.classes, " "), classesFile, log), nil
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

	regexIdsMatcher := regexp.MustCompile("^/(.+)/$")

	for _, c := range strings.Split(p.caller, " ") {
		if c == id {
			return true
		}

		if strings.HasPrefix(c, "/") {
			if !regexIdsMatcher.MatchString(c) {
				p.log.Errorf("Invalid CallerID matcher '%s' found in policy file %s", c, p.file)
				return false
			}

			matched := regexIdsMatcher.FindStringSubmatch(c)

			re, err := regexp.Compile(matched[1])
			if err != nil {
				p.log.Errorf("Could not compile regex found in CallerID '%s' in policy file %s: %s", c, p.file, err)
				return false
			}

			if re.MatchString(id) {
				return true
			}
		}
	}

	return false
}

// SetFile sets the file being parsed for errors and logging purposes
func (p *actionPolicyPolicy) SetFile(f string) {
	p.file = f
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
