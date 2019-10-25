package builddefaults

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"github.com/choria-io/go-choria/backoff"
	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-config"
	"github.com/choria-io/go-srvcache"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
)

type provClaims struct {
	Secure      bool   `json:"chs"`
	URLs        string `json:"chu"`
	Token       string `json:"cht"`
	SRVDomain   string `json:"chsrv"`
	ProvDefault bool   `json:"chpd"`

	jwt.StandardClaims
}

// Provider creates an instance of the provider
func Provider() *Resolver {
	return &Resolver{}
}

// Resolver resolve names against the compile time build properties
type Resolver struct {
	identity string
}

// Name is te name of the resolver
func (b *Resolver) Name() string {
	return "Default"
}

// Configure overrides build settings using the contents of the JWT
func (b *Resolver) Configure(cfg *config.Config, log *logrus.Entry) {
	if build.ProvisionJWTFile == "" {
		return
	}

	_, err := os.Stat(build.ProvisionJWTFile)
	if os.IsNotExist(err) {
		return
	}

	log.Infof("Setting build defaults to those found in %s", build.ProvisionJWTFile)

	b.identity = cfg.Identity

	_, err = b.setBuildBasedOnJWT()
	if err != nil {
		log.Errorf("Configuration of the provisioner settings based on JWT file %s failed: %s", build.ProvisionJWTFile, err)
	}
}

// Targets are the build time configured provisioners
func (b *Resolver) Targets(ctx context.Context, log *logrus.Entry) []string {
	if build.ProvisionBrokerURLs != "" {
		return strings.Split(build.ProvisionBrokerURLs, ",")
	}

	domain := build.ProvisionBrokerSRVDomain
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

		for _, q := range []string{"_mcollective-provisioner._tcp", "_choria-provisioner.tcp"} {
			if ctx.Err() != nil {
				return []string{}
			}

			record := q + "." + domain
			log.Infof("Attempting SRV lookup on %s", record)

			servers, err = cache.LookupSrvServers("", "", record, "")
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

		backoff.TwentySec.InterruptableSleep(ctx, try)
	}

	return servers.Strings()
}

// setBuildBasedOnJWT sets build settings based on contents of a JWT file
func (b *Resolver) setBuildBasedOnJWT() (*provClaims, error) {
	_, err := os.Stat(build.ProvisionJWTFile)
	if os.IsNotExist(err) {
		return &provClaims{}, nil
	}

	j, err := ioutil.ReadFile(build.ProvisionJWTFile)
	if err != nil {
		return nil, err
	}

	claims := &provClaims{}
	_, _, err = new(jwt.Parser).ParseUnverified(string(j), claims)
	if err != nil {
		return nil, fmt.Errorf("jwt parse error: %s", err)
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

	build.ProvisionBrokerURLs = claims.URLs
	build.ProvisionToken = claims.Token
	build.ProvisionBrokerSRVDomain = claims.SRVDomain

	if claims.ProvDefault {
		build.ProvisionModeDefault = "true"
	} else {
		build.ProvisionModeDefault = "false"
	}

	if claims.Secure {
		build.ProvisionSecure = "true"
	} else {
		build.ProvisionSecure = "false"
	}

	return claims, nil
}
