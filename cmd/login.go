// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/choria-io/go-choria/choria"
)

type loginCommand struct {
	command
}

func (p *loginCommand) Setup() error {
	p.cmd = cli.app.Command("login", "Authenticate against the Choria AAA System")
	p.cmd.Flag("config", "Config file to use").PlaceHolder("FILE").StringVar(&configFile)

	return nil
}

func (p *loginCommand) Configure() error {
	return commonConfigure()
}

func (p *loginCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	ext, _ := exec.LookPath("choria-login")
	if ext != "" {
		return p.runExternal(ext)
	}

	return p.login()
}

func (p *loginCommand) sign(user string, pass string, timeStamp string) (signature string, pub string, err error) {
	seed, err := c.SignerSeedFile()
	if err != nil {
		return "", "", err
	}

	pubK, priK, err := choria.Ed25519KeyPairToFile(seed)
	if err != nil {
		return "", "", fmt.Errorf("could not generate keypair: %s", err)
	}

	sig, err := choria.Ed25519Sign(priK, []byte(fmt.Sprintf("%s:%s:%s", timeStamp, user, pass)))
	if err != nil {
		return "", "", fmt.Errorf("could not sign request: %s", err)
	}

	return hex.EncodeToString(sig), hex.EncodeToString(pubK), nil
}

func (p *loginCommand) login() error {
	loginURLs := cfg.Choria.AAAServiceLoginURLs
	if len(loginURLs) == 0 {
		return fmt.Errorf("please configure a login server URL using plugin.login.aaasvc.login.url")
	}

	token, err := c.SignerTokenFile()
	if err != nil {
		return fmt.Errorf("no token path found: %v", err)
	}

	user := ""
	pass := ""

	err = survey.AskOne(&survey.Input{Message: "Username: ", Default: os.Getenv("USER")}, &user, survey.WithValidator(survey.Required))
	if err != nil {
		return err
	}

	err = survey.AskOne(&survey.Password{Message: "Password: "}, &pass, survey.WithValidator(survey.Required))
	if err != nil {
		return err
	}

	abs, err := filepath.Abs(token)
	if err != nil {
		return fmt.Errorf("cannot determine parent directory for token file: %s", err)
	}

	err = os.MkdirAll(filepath.Dir(abs), 0700)
	if err != nil {
		return err
	}

	nowString := strconv.Itoa(int(time.Now().Unix()))
	sig, pub, err := p.sign(user, pass, nowString)
	if err != nil {
		return err
	}

	request := map[string]any{
		"username":   user,
		"password":   pass,
		"signature":  sig,
		"public_key": pub,
		"timestamp":  nowString,
	}
	jr, err := json.Marshal(&request)
	if err != nil {
		return err
	}

	rand.Shuffle(len(loginURLs), func(i, j int) { loginURLs[i], loginURLs[j] = loginURLs[j], loginURLs[i] })

	uri, err := url.Parse(loginURLs[0])
	if err != nil {
		return err
	}

	client := &http.Client{}
	if uri.Scheme == "https" {
		tlsc, err := c.ClientTLSConfig()
		if err != nil {
			return err
		}
		client.Transport = &http.Transport{TLSClientConfig: tlsc}
	}

	resp, err := client.Post(uri.String(), "application/json", bytes.NewBuffer(jr))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("invalid response code %d: %s", resp.StatusCode, string(body))
	}

	login := map[string]string{}
	err = json.Unmarshal(body, &login)
	if err != nil {
		return err
	}

	if login["error"] != "" {
		return fmt.Errorf(login["error"])
	}

	if login["token"] == "" {
		return fmt.Errorf("no token received")
	}

	err = os.WriteFile(abs, []byte(login["token"]), 0600)
	if err != nil {
		return err
	}
	fmt.Printf("Token saved to %s\n", abs)

	return nil
}

func (p *loginCommand) runExternal(ext string) error {
	cmd := exec.Command(ext, os.Args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func init() {
	cli.commands = append(cli.commands, &loginCommand{})
}
