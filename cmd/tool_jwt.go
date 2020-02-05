package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/provtarget/builddefaults"
	"github.com/dgrijalva/jwt-go"
)

type tJWTCommand struct {
	file         string
	insecure     bool
	validateCert string
	token        string
	srvDomain    string
	regData      string
	facts        string
	provDefault  bool
	urls         []string

	command
}

func (j *tJWTCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		j.cmd = tool.Cmd().Command("jwt", "Create or validate JWT files")
		j.cmd.Arg("file", "The JWT file to act on").Required().StringVar(&j.file)
		j.cmd.Arg("certificate", "Path to a certificate used to validate or sign the JWT").Required().ExistingFileVar(&j.validateCert)
		j.cmd.Flag("insecure", "Disable TLS security during provisioning").BoolVar(&j.insecure)
		j.cmd.Flag("token", "Token used to secure access to the provisioning agent").StringVar(&j.token)
		j.cmd.Flag("urls", "URLs to connect to for provisioning").StringsVar(&j.urls)
		j.cmd.Flag("srv", "Domain to query for SRV records to find provisioning urls").StringVar(&j.srvDomain)
		j.cmd.Flag("default", "Enables provisioning by default").BoolVar(&j.provDefault)
		j.cmd.Flag("registration", "File to publish as registration data during provisioning").StringVar(&j.regData)
		j.cmd.Flag("facts", "File to use for facts during registration").StringVar(&j.facts)
	}

	return nil
}

func (j *tJWTCommand) Configure() error {
	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return fmt.Errorf("could not create default configuration: %s", err)
	}

	cfg.DisableSecurityProviderVerify = true
	cfg.Choria.SecurityProvider = "file"

	return nil
}

func (j *tJWTCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	if choria.FileExist(j.file) {
		err = j.validateJWT()
		if err != nil {
			return fmt.Errorf("token validation failed: %s", err)
		}
	} else {
		err = j.createJWT()
		if err != nil {
			return fmt.Errorf("could not create token: %s", err)
		}
	}

	return nil
}

func (j *tJWTCommand) validateJWT() error {
	certdat, err := ioutil.ReadFile(j.validateCert)
	if err != nil {
		return fmt.Errorf("could not read validation certificate: %s", err)
	}

	cert, err := jwt.ParseRSAPublicKeyFromPEM(certdat)
	if err != nil {
		return fmt.Errorf("could not parse validation certificate: %s", err)
	}

	token, err := ioutil.ReadFile(j.file)
	if err != nil {
		return fmt.Errorf("could not read token: %s", err)
	}

	claims := &builddefaults.ProvClaims{}
	_, err = jwt.ParseWithClaims(string(token), claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unsupported signing method in token")
		}

		return cert, nil
	})
	if err != nil {
		return fmt.Errorf("could not parse token: %s", err)
	}

	if claims.Token != "" {
		claims.Token = "set"
	}

	fmt.Printf("JWT Token %s\n\n", j.file)
	fmt.Printf("                         Token: %s\n", claims.Token)
	fmt.Printf("                        Secure: %t\n", claims.Secure)
	if claims.SRVDomain != "" {
		fmt.Printf("                    SRV Domain: %s\n", claims.SRVDomain)
	}
	if claims.URLs != "" {
		fmt.Printf("                          URLS: %s\n", claims.URLs)
	}
	fmt.Printf("       Provisioning by default: %t\n", claims.ProvDefault)
	if claims.ProvRegData != "" {
		fmt.Printf("Provisioning Registration Data: %s\n", claims.ProvRegData)
	}
	if claims.ProvFacts != "" {
		fmt.Printf("            Provisioning Facts: %s\n", claims.ProvFacts)
	}

	stdc, err := json.MarshalIndent(claims.StandardClaims, "                               ", "  ")
	if err != nil {
		return nil
	}

	fmt.Printf("              Standard Claims: %s\n", string(stdc))

	return nil
}

func (j *tJWTCommand) createJWT() error {
	if j.token == "" {
		survey.AskOne(&survey.Password{Message: "Provisioning Token"}, &j.token, survey.WithValidator(survey.Required))

	}

	if j.srvDomain == "" && len(j.urls) == 0 {
		return fmt.Errorf("URLs or a SRV Domain is required")
	}

	claims := &builddefaults.ProvClaims{
		Secure:      true,
		ProvDefault: false,
		Token:       j.token,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  time.Now().UTC().Unix(),
			Issuer:    "choria cli",
			NotBefore: time.Now().UTC().Unix(),
			Subject:   "choria_provisioning",
		},
	}

	if j.insecure {
		claims.Secure = false

	}

	if len(j.urls) > 0 {
		claims.URLs = strings.Join(j.urls, ",")
	}

	if j.srvDomain != "" {
		claims.SRVDomain = j.srvDomain
	}

	if j.provDefault {
		claims.ProvDefault = true
	}

	if j.regData != "" {
		claims.ProvRegData = j.regData
	}

	if j.facts != "" {
		claims.ProvFacts = j.facts
	}

	keydat, err := ioutil.ReadFile(j.validateCert)
	if err != nil {
		return fmt.Errorf("could not read signing key: %s", err)
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(keydat)
	if err != nil {
		return fmt.Errorf("could not parse signing key: %s", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	stoken, err := token.SignedString(key)
	if err != nil {
		return fmt.Errorf("could not sign token using key: %s", err)
	}

	err = ioutil.WriteFile(j.file, []byte(stoken), 0644)
	if err != nil {
		return fmt.Errorf("could not write token: %s", err)
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tJWTCommand{})
}
