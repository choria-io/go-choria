// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package builddefaults

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/srvcache"
	"github.com/choria-io/go-choria/tokens"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/backoff"
	"github.com/choria-io/go-choria/build"
)

// Provider creates an instance of the provider
func Provider() *Resolver {
	return &Resolver{
		bi: &build.Info{},
	}
}

// Resolver resolve names against the compile time build properties
type Resolver struct {
	identity string
	bi       *build.Info
}

// Name is te name of the resolver
func (b *Resolver) Name() string {
	return "Choria JWT Resolver"
}

// Configure overrides build settings using the contents of the JWT
func (b *Resolver) Configure(cfg *config.Config, log *logrus.Entry) {
	jwtf := b.bi.ProvisionJWTFile()
	if jwtf == "" {
		return
	}

	if !util.FileExist(jwtf) {
		return
	}

	log.Infof("Setting build defaults to those found in %s", jwtf)

	b.identity = cfg.Identity

	_, err := b.setBuildBasedOnJWT()
	if err != nil {
		log.Errorf("Configuration of the provisioner settings based on JWT file %s failed: %s", jwtf, err)
	}
}

// Targets are the build time configured provisioners
func (b *Resolver) Targets(ctx context.Context, log *logrus.Entry) []string {
	if b.bi.ProvisionBrokerURLs() != "" {
		return strings.Split(b.bi.ProvisionBrokerURLs(), ",")
	}

	domain := b.bi.ProvisionBrokerSRVDomain()
	if domain == "" {
		log.Warnf("Neither provisioning broker url or provisioning SRV domain is set, cannot continue")
		return []string{}
	}

	log.Infof("Performing provisioning broker resolution via SRV using domain %s", domain)

	servers := srvcache.NewServers()
	cache := srvcache.New(b.identity, 5*time.Second, net.LookupSRV, log)
	var err error
	try := 0

	for {
		try++

		for _, q := range []string{"_choria-provisioner._tcp"} {
			if ctx.Err() != nil {
				return []string{}
			}

			record := q + "." + domain
			log.Infof("Attempting SRV lookup on %s", record)

			servers, err = cache.LookupSrvServers("", "", record, "nats")
			if err != nil {
				log.Warnf("Failed to resolve %s: %s", record, err)
				continue
			}

			log.Infof("Found %d SRV records for %s", servers.Count(), record)
			break
		}

		if servers.Count() > 0 {
			break
		}

		log.Warnf("Resolving provisioning brokers via SRV lookups in domain %s failed on try %d, will keep trying", domain, try)

		backoff.TwentySec.TrySleep(ctx, try)
	}

	return servers.Strings()
}

// setBuildBasedOnJWT sets build settings based on contents of a JWT file
func (b *Resolver) setBuildBasedOnJWT() (*tokens.ProvisioningClaims, error) {
	bi := b.bi

	d, err := os.ReadFile(bi.ProvisionJWTFile())
	if err != nil {
		return nil, err
	}

	claims, err := tokens.ParseProvisionTokenUnverified(string(d))
	if err != nil {
		return nil, err
	}

	if claims.Token == "" {
		return nil, fmt.Errorf("no auth token")
	}

	if claims.SRVDomain == "" && claims.URLs == "" {
		return nil, fmt.Errorf("no srv domain or urls")
	}

	if claims.SRVDomain != "" && claims.URLs != "" {
		return nil, fmt.Errorf("both srv domain and URLs supplied")
	}

	bi.SetProvisionBrokerURLs(claims.URLs)
	bi.SetProvisionToken(claims.Token)
	bi.SetProvisionBrokerSRVDomain(claims.SRVDomain)
	bi.SetProvisionUsingVersion2(claims.ProtoV2)

	if claims.ProvDefault {
		bi.EnableProvisionModeAsDefault()
	} else {
		bi.DisableProvisionModeAsDefault()
	}

	if claims.Secure {
		bi.EnableProvisionModeSecurity()
	} else {
		bi.DisableProvisionModeSecurity()
	}

	if claims.ProvFacts != "" {
		bi.SetProvisionFacts(claims.ProvFacts)
	}

	if claims.ProvRegData != "" {
		bi.SetProvisionRegistrationData(claims.ProvRegData)
	}

	if claims.ProvNatsUser != "" {
		bi.SetProvisioningBrokerUsername(claims.ProvNatsUser)
	}

	if claims.ProvNatsPass != "" {
		bi.SetProvisioningBrokerPassword(claims.ProvNatsPass)
	}

	return claims, nil
}
