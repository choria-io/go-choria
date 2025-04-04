// Copyright (c) 2023-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/choria-io/fisk"
	"github.com/choria-io/go-choria/config"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/tokens"
	"github.com/expr-lang/expr"
	"github.com/nats-io/jsm.go/monitor"
)

type jwtCheckCommand struct {
	file             string
	kind             string
	validityMin      time.Duration
	validityMax      time.Duration
	chainIssuerSet   bool
	chainIssuer      bool
	issuer           string
	opa              bool
	pub              []string
	sub              []string
	identity         string
	queries          []string
	renderFormatText string
	renderFormat     monitor.RenderFormat

	command
}

func (c *jwtCheckCommand) Setup() error {
	if jwt, ok := cmdWithFullCommand("jwt"); ok {
		c.cmd = jwt.Cmd().Command("check", "Checks validity of JWT tokens")
		c.cmd.Arg("file", "The JWT file to act on").Required().StringVar(&c.file)
		c.cmd.Flag("purpose", "Checks if a JWT is of a specific purpose (client, server, provisioner)").EnumVar(&c.kind, "client", "server", "provisioning")
		c.cmd.Flag("validity-min", "Checks the validity of a token is more than DURATION").PlaceHolder("DURATION").DurationVar(&c.validityMin)
		c.cmd.Flag("validity-max", "Checks the validity of a token is not more than DURATION").PlaceHolder("DURATION").DurationVar(&c.validityMax)
		c.cmd.Flag("chain-issuer", "Checks if a token is a chain issuer or not").IsSetByUser(&c.chainIssuerSet).BoolVar(&c.chainIssuer)
		c.cmd.Flag("issuer", "Checks if a specific issuer signed it, directly or via a chain").PlaceHolder("PUBK").StringVar(&c.issuer)
		c.cmd.Flag("client-opa", "Checks if a client has a OPA policy").UnNegatableBoolVar(&c.opa)
		c.cmd.Flag("sub", "Additional subjects that should be subscribable").PlaceHolder("SUBJECT").StringsVar(&c.sub)
		c.cmd.Flag("pub", "Additional subjects that should be publishable").PlaceHolder("SUBJECT").StringsVar(&c.pub)
		c.cmd.Flag("identity", "Checks the identity or caller id").StringVar(&c.identity)
		c.cmd.Flag("query", "Performs a boolean expr query against the token").StringsVar(&c.queries)
		c.cmd.Flag("format", "Render the check in a specific format (nagios, json, prometheus, text)").Default("nagios").EnumVar(&c.renderFormatText, "nagios", "json", "prometheus", "text")

		c.cmd.PreAction(c.parseRenderFormat)
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &jwtCheckCommand{})
}

func (c *jwtCheckCommand) parseRenderFormat(_ *fisk.ParseContext) error {
	switch c.renderFormatText {
	case "prometheus":
		c.renderFormat = monitor.PrometheusFormat
	case "text":
		c.renderFormat = monitor.TextFormat
	case "json":
		c.renderFormat = monitor.JSONFormat
	}

	return nil
}

func (c *jwtCheckCommand) Configure() error {
	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return fmt.Errorf("could not create default configuration: %s", err)
	}

	cfg.DisableSecurityProviderVerify = true
	cfg.Choria.SecurityProvider = "file"

	return nil
}

func (c *jwtCheckCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	result := &monitor.Result{Name: c.file, Check: "jwt", NameSpace: "choria", RenderFormat: c.renderFormat}
	defer result.GenericExit()

	c.check(result)

	return nil
}

func (c *jwtCheckCommand) check(result *monitor.Result) {
	if c.file == "" {
		result.Critical("token file is required")
		return
	}

	if !iu.FileExist(c.file) {
		result.Critical("token not found")
		return
	}

	token, err := os.ReadFile(c.file)
	if result.CriticalIfErr(err, "token cannot be read: %v", err) {
		return
	}

	var issuer ed25519.PublicKey
	if c.issuer != "" {
		issuer, err = hex.DecodeString(c.issuer)
		if result.CriticalIfErr(err, "invalid issuer") {
			return
		}
	}

	purpose := tokens.TokenPurpose(string(token))

	c.checkPurpose(purpose, result)

	switch purpose {
	case tokens.ClientIDPurpose:
		c.checkClientToken(string(token), issuer, result)
	case tokens.ServerPurpose:
		c.checkServerToken(string(token), issuer, result)
	case tokens.ProvisioningPurpose:
		c.checkProvToken(string(token), issuer, result)
	case tokens.UnknownPurpose:
		result.Critical("unknown token purpose")
	}
}

func (c *jwtCheckCommand) checkClientToken(token string, issuer ed25519.PublicKey, result *monitor.Result) {
	var err error
	var jwt *tokens.ClientIDClaims

	if issuer == nil {
		jwt, err = tokens.ParseClientIDTokenUnverified(token)
	} else {
		jwt, err = tokens.ParseClientIDToken(token, issuer, false)
	}
	if result.CriticalIfErr(err, "%s", err) {
		return
	}

	if c.identity != "" && jwt.CallerID != c.identity {
		result.Critical("identity %s", jwt.CallerID)
	}

	if jwt.PublicKey == "" {
		result.Critical("no public key")
	}

	if c.chainIssuerSet {
		isIssuer := jwt.IsChainedIssuer(true)

		if isIssuer != c.chainIssuer {
			result.Critical("chain issuer: %t", isIssuer)
		}
	}

	if c.opa && jwt.OPAPolicy == "" {
		result.Critical("no OPA policy")
	}

	for _, sub := range c.sub {
		if !iu.StringInList(jwt.AdditionalSubscribeSubjects, sub) {
			result.Critical("subscribe %v", sub)
		}
	}

	for _, pub := range c.pub {
		if !iu.StringInList(jwt.AdditionalPublishSubjects, pub) {
			result.Critical("subscribe %v", pub)
		}
	}

	c.commonChecks(jwt.StandardClaims, result)
	c.checkQueries(jwt, result)

	result.Ok("client token")
}

func (c *jwtCheckCommand) checkServerToken(token string, issuer ed25519.PublicKey, result *monitor.Result) {
	var err error
	var jwt *tokens.ServerClaims

	if issuer == nil {
		jwt, err = tokens.ParseServerTokenUnverified(token)
	} else {
		jwt, err = tokens.ParseServerToken(token, issuer)
	}
	if result.CriticalIfErr(err, "%s", err) {
		return
	}

	if c.identity != "" && jwt.ChoriaIdentity != c.identity {
		result.Critical("identity %s", jwt.ChoriaIdentity)
	}

	if jwt.PublicKey == "" {
		result.Critical("no public key")
	}

	for _, pub := range c.pub {
		if !iu.StringInList(jwt.AdditionalPublishSubjects, pub) {
			result.Critical("publish %v", pub)
		}
	}

	c.commonChecks(jwt.StandardClaims, result)
	c.checkQueries(jwt, result)
	result.Ok("server token")
}

func (c *jwtCheckCommand) checkProvToken(token string, issuer ed25519.PublicKey, result *monitor.Result) {
	var err error
	var jwt *tokens.ProvisioningClaims

	if issuer == nil {
		jwt, err = tokens.ParseProvisionTokenUnverified(token)
	} else {
		jwt, err = tokens.ParseProvisioningToken(token, issuer)
	}
	if result.CriticalIfErr(err, "%s", err) {
		return
	}

	c.commonChecks(jwt.StandardClaims, result)
	c.checkQueries(jwt, result)
	result.Ok("provisioning token")
}

func (c *jwtCheckCommand) commonChecks(claims tokens.StandardClaims, result *monitor.Result) {
	c.checkValidity(claims, result)

	if claims.ID == "" {
		result.Critical("no id")
	}
	if claims.ExpireTime().IsZero() {
		result.Critical("no expiry")
	}
	if claims.NotBefore == nil || claims.NotBefore.IsZero() {
		result.Critical("no not before")
	}
}

func (c *jwtCheckCommand) checkValidity(claims tokens.StandardClaims, result *monitor.Result) {
	exp := claims.ExpireTime()
	untilExp := time.Until(exp)
	if exp.IsZero() {
		result.Critical("no expires time")
		return
	}

	result.Pd(&monitor.PerfDataItem{Name: "expires", Value: untilExp.Seconds(), Unit: "s", Help: "Seconds until expiry"})

	if claims.IsExpired() {
		result.Critical("expired")
		return
	}

	if c.validityMin > 0 {
		if untilExp < c.validityMin {
			result.Critical("expires in %v", iu.RenderDuration(untilExp))
		}
	}

	if c.validityMax > 0 {
		if untilExp > c.validityMax {
			result.Critical("expires in %v", iu.RenderDuration(untilExp))
		}
	}
}

func (c *jwtCheckCommand) checkPurpose(purpose tokens.Purpose, result *monitor.Result) {
	if c.kind != "" {
		var should tokens.Purpose
		switch c.kind {
		case "client":
			should = tokens.ClientIDPurpose
		case "server":
			should = tokens.ServerPurpose
		case "provisioning":
			should = tokens.ProvisioningPurpose
		}
		if purpose != should {
			result.Critical("%s purpose", purpose)
		}
	}
}

func (c *jwtCheckCommand) checkQueries(token any, result *monitor.Result) {
	for _, query := range c.queries {
		var claims map[string]any

		// not checking errors, query would fail anyway
		dat, _ := json.Marshal(token)
		json.Unmarshal(dat, &claims)

		prog, err := expr.Compile(query, expr.AsBool())
		if result.CriticalIfErr(err, "invalid query: %s: %v", query, err) {
			return
		}

		res, err := expr.Run(prog, claims)
		if result.CriticalIfErr(err, "invalid query: %s: %v", query, err) {
			return
		}

		b, ok := res.(bool)
		if !ok {
			result.Critical("query not boolean")
			return
		}

		if !b {
			result.Critical(query)
		}
	}
}
