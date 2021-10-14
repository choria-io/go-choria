package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/AlecAivazis/survey/v2"
	iu "github.com/choria-io/go-choria/internal/util"
)

type loginCommand struct {
	command
}

func (p *loginCommand) Setup() error {
	p.cmd = cli.app.Command("login", "Logs into the Choria AAA System")

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

func (p *loginCommand) login() error {
	loginURL := cfg.Option("plugin.login.aaasvc.login.url", "")
	if loginURL == "" {
		return fmt.Errorf("please configure a login server URL using plugin.login.aaasvc.login.url")
	}

	if cfg.Choria.RemoteSignerTokenFile == "" && cfg.Choria.RemoteSignerTokenEnvironment == "" {
		return fmt.Errorf("no token configuration set")
	}

	user := ""
	pass := ""

	err := survey.AskOne(&survey.Input{Message: "Username: ", Default: os.Getenv("USER")}, &user, survey.WithValidator(survey.Required))
	if err != nil {
		return err
	}

	err = survey.AskOne(&survey.Password{Message: "Password: "}, &pass, survey.WithValidator(survey.Required))
	if err != nil {
		return err
	}

	request := map[string]string{"username": user, "password": pass}
	jr, err := json.Marshal(&request)
	if err != nil {
		return err
	}

	uri, err := url.Parse(loginURL)
	if err != nil {
		return err
	}

	client := &http.Client{}
	if uri.Scheme == "https" {
		tlsc, err := c.ClientTLSConfig()
		if err != nil {
			return err
		}
		tlsc.InsecureSkipVerify = true
		tlsc.VerifyConnection = nil // legacy san checks are here
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

	switch {
	case cfg.Choria.RemoteSignerTokenFile != "":
		abs, err := filepath.Abs(cfg.Choria.RemoteSignerTokenFile)
		if err != nil {
			return fmt.Errorf("cannot determine parent directory for token file: %s", err)
		}
		parent := filepath.Dir(abs)
		if !iu.FileIsDir(parent) {
			err = os.MkdirAll(parent, 0700)
			if err != nil {
				return err
			}
		}

		err = os.WriteFile(abs, []byte(login["token"]), 0600)
		if err != nil {
			return err
		}
		fmt.Printf("Token saved to %s\n", cfg.Choria.RemoteSignerTokenFile)
	case cfg.Choria.RemoteSignerTokenEnvironment != "":
		if os.Getenv("SHELL") == "" {
			return fmt.Errorf("cannot start new shell, please set SHELL environment")
		}

		err = os.Setenv(cfg.Choria.RemoteSignerTokenEnvironment, login["token"])
		if err != nil {
			return err
		}

		cmd := exec.Command(os.Getenv("SHELL"))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}

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
