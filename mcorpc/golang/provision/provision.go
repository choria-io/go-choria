package provision

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	mrand "math/rand"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc"
	"github.com/sirupsen/logrus"
)

type ConfigureRequest struct {
	Configuration string `json:"config"`
	Certificate   string `json:"certificate"`
	CA            string `json:"ca"`
	SSLDir        string `json:"ssldir"`
}

type RestartRequest struct {
	Splay int `json:"splay"`
}

type CSRRequest struct {
	CN string `json:"cn"`
	C  string `json:"C"`
	L  string `json:"L"`
	O  string `json:"O"`
	OU string `json:"OU"`
	ST string `json:"ST"`
}

type CSRReply struct {
	CSR    string `json:"csr"`
	SSLDir string `json:"ssldir"`
}

type Reply struct {
	Message string `json:"message"`
}

var mu = &sync.Mutex{}
var allowRestart = true

func New(mgr server.AgentManager) (*mcorpc.Agent, error) {
	metadata := &agents.Metadata{
		Name:        "choria_provision",
		Description: "Choria Provisioner",
		Author:      "R.I.Pienaar <rip@devco.net>",
		Version:     build.Version,
		License:     build.License,
		Timeout:     2,
		URL:         "http://choria.io",
	}

	agent := mcorpc.New("choria_provision", metadata, mgr.Choria(), mgr.Logger())

	agent.MustRegisterAction("gencsr", csrAction)
	agent.MustRegisterAction("configure", configureAction)
	agent.MustRegisterAction("restart", restartAction)
	agent.MustRegisterAction("reprovision", reprovisionAction)

	return agent, nil
}

func csrAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	mu.Lock()
	defer mu.Unlock()

	if !agent.Choria.ProvisionMode() {
		abort("Cannot reconfigure a server that is not in provisioning mode", reply)
		return
	}

	if agent.Config.ConfigFile == "" && agent.Config.Choria.SSLDir == "" {
		abort("Cannot determine where to store SSL data, no configure file given and no SSL directory configured", reply)
		return
	}

	ssldir := filepath.Join(filepath.Dir(agent.Config.ConfigFile), "ssl")
	if agent.Config.Choria.SSLDir != "" {
		ssldir = agent.Config.Choria.SSLDir
	}

	keyfile := filepath.Join(ssldir, "private.pem")
	csrfile := filepath.Join(ssldir, "csr.pem")

	agent.Log.Infof("Creating a new CSR in %s", ssldir)

	err := os.MkdirAll(ssldir, 0700)
	if err != nil {
		abort(fmt.Sprintf("Could not create SSL Directory %s: %s", ssldir, err), reply)
		return
	}

	args := CSRRequest{}
	if !mcorpc.ParseRequestData(&args, req, reply) {
		return
	}

	if args.CN == "" {
		args.CN = agent.Choria.Certname()
	}

	subj := pkix.Name{
		CommonName: args.CN,
	}

	if args.C != "" {
		subj.Country = []string{args.C}
	}

	if args.L != "" {
		subj.Locality = []string{args.L}
	}

	if args.O != "" {
		subj.Organization = []string{args.O}
	}

	if args.OU != "" {
		subj.OrganizationalUnit = []string{args.OU}
	}

	rawSubj := subj.ToRDNSequence()

	asn1Subj, err := asn1.Marshal(rawSubj)
	if err != nil {
		abort(fmt.Sprintf("Could not create CSR: %s", err), reply)
		return
	}

	template := x509.CertificateRequest{
		RawSubject:         asn1Subj,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	keyBytes, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		abort(fmt.Sprintf("Could not create private key: %s", err), reply)
		return
	}

	keyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(keyBytes),
		},
	)

	err = ioutil.WriteFile(keyfile, keyPem, 0700)
	if err != nil {
		abort(fmt.Sprintf("Could not store private key: %s", err), reply)
		return
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, keyBytes)
	if err != nil {
		abort(fmt.Sprintf("Could not create CSR bytes: %s", err), reply)
		return
	}

	pb := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})

	err = ioutil.WriteFile(csrfile, pb, 0700)
	if err != nil {
		abort(fmt.Sprintf("Could not store CSR: %s", err), reply)
		return
	}

	reply.Data = &CSRReply{
		CSR:    string(pb),
		SSLDir: ssldir,
	}
}

func reprovisionAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	mu.Lock()
	defer mu.Unlock()

	if agent.Choria.ProvisionMode() {
		abort("Server is already in provisioning mode, cannot enable provisioning mode again", reply)
		return
	}

	if agent.Config.ConfigFile == "" {
		abort("Cannot determine the configuration file to manage", reply)
		return
	}

	cfg := make(map[string]string)

	cfg["plugin.choria.server.provision"] = "1"
	cfg["loglevel"] = "debug"

	if agent.Config.LogFile != "" {
		cfg["logfile"] = agent.Config.LogFile
	}

	if agent.Config.Choria.FileContentRegistrationData != "" {
		cfg["registration"] = "file_content"
		cfg["plugin.choria.registration.file_content.data"] = agent.Config.Choria.FileContentRegistrationData
	}

	_, err := writeConfig(cfg, req, agent.Config, agent.Log)
	if err != nil {
		abort(fmt.Sprintf("Could not write config: %s", err), reply)
		return
	}

	splay := time.Duration(mrand.Intn(10) + 2)

	if allowRestart {
		go restart(splay, agent.Log)
	}

	reply.Data = Reply{fmt.Sprintf("Restarting after %ds", splay)}
}

func configureAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	mu.Lock()
	defer mu.Unlock()

	if !agent.Choria.ProvisionMode() {
		abort("Cannot reconfigure a server that is not in provisioning mode", reply)
		return
	}

	if agent.Config.ConfigFile == "" {
		abort("Cannot determine the configuration file to manage", reply)
		return
	}

	args := &ConfigureRequest{}
	if !mcorpc.ParseRequestData(args, req, reply) {
		return
	}

	if len(args.Configuration) == 0 {
		abort("Did not receive any configuration to write, cannot write a empty configuration file", reply)
		return
	}

	settings := make(map[string]string)
	err := json.Unmarshal([]byte(args.Configuration), &settings)
	if err != nil {
		abort(fmt.Sprintf("Could not decode configuration data: %s", err), reply)
		return
	}

	lines, err := writeConfig(settings, req, agent.Config, agent.Log)
	if err != nil {
		abort(fmt.Sprintf("Could not write config: %s", err), reply)
		return
	}

	if args.Certificate != "" && args.SSLDir != "" && args.CA != "" {
		target := filepath.Join(args.SSLDir, "certificate.pem")
		err = ioutil.WriteFile(target, []byte(args.Certificate), 0700)
		if err != nil {
			abort(fmt.Sprintf("Could not write Certificate to %s: %s", target, err), reply)
			return
		}

		target = filepath.Join(args.SSLDir, "ca.pem")
		err = ioutil.WriteFile(target, []byte(args.CA), 0700)
		if err != nil {
			abort(fmt.Sprintf("Could not write CA to %s: %s", target, err), reply)
			return
		}

	}

	reply.Data = Reply{fmt.Sprintf("Wrote %d lines to %s", lines, agent.Config.ConfigFile)}
}

func restartAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	mu.Lock()
	defer mu.Unlock()

	if !agent.Choria.ProvisionMode() {
		abort("Cannot restart a server that is not in provisioning mode", reply)
		return
	}

	args := RestartRequest{}
	err := json.Unmarshal(req.Data, &args)
	if err != nil {
		abort(fmt.Sprintf("Could not parse request arguments: %s", err), reply)
		return
	}

	cfg, err := config.NewConfig(agent.Config.ConfigFile)
	if err != nil {
		abort(fmt.Sprintf("Configuration %s could not be parsed, restart cannot continue: %s", agent.Config.ConfigFile, err), reply)
		return
	}

	if cfg.Choria.Provision {
		abort(fmt.Sprintf("Configuration %s enables provisioning, restart cannot continue", agent.Config.ConfigFile), reply)
		return
	}

	if args.Splay == 0 {
		args.Splay = 10
	}

	splay := time.Duration(mrand.Intn(args.Splay) + 2)

	agent.Log.Warnf("Restarting server via request %s from %s (%s) with splay %d", req.RequestID, req.CallerID, req.SenderID, splay)

	if allowRestart {
		go restart(splay, agent.Log)
	}

	reply.Data = Reply{fmt.Sprintf("Restarting Choria Server after %ds", splay)}
}

func abort(msg string, reply *mcorpc.Reply) {
	reply.Statuscode = mcorpc.Aborted
	reply.Statusmsg = msg
}

func writeConfig(settings map[string]string, req *mcorpc.Request, cfg *config.Config, log *logrus.Entry) (int, error) {
	cfile := cfg.ConfigFile

	_, err := os.Stat(cfile)
	if err == nil {
		cfile, err = filepath.EvalSymlinks(cfg.ConfigFile)
		if err != nil {
			return 0, fmt.Errorf("cannot determine full path to config file %s: %s", cfile, err)
		}
	}

	log.Warnf("Rewriting configuration file %s in request %s from %s (%s)", cfile, req.RequestID, req.CallerID, req.SenderID)

	cdir := filepath.Dir(cfile)

	tmpfile, err := ioutil.TempFile(cdir, "provision")
	if err != nil {
		return 0, fmt.Errorf("cannot create a temp file in %s: %s", cdir, err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	_, err = fmt.Fprintf(tmpfile, "# configuration file writen in request %s from %s (%s) at %s\n", req.RequestID, req.CallerID, req.SenderID, time.Now())
	if err != nil {
		return 0, fmt.Errorf("could not write to temp file %s: %s", tmpfile.Name(), err)
	}

	written := 1

	for k, v := range settings {
		log.Infof("Adding configuration: %s = %s", k, v)

		_, err := fmt.Fprintf(tmpfile, "%s=%s\n", k, v)
		if err != nil {
			return 0, fmt.Errorf("could not write to temp file %s: %s", tmpfile.Name(), err)
		}

		written++
	}

	err = tmpfile.Close()
	if err != nil {
		return 0, fmt.Errorf("could not close temp file %s: %s", tmpfile.Name(), err)
	}

	_, err = config.NewConfig(tmpfile.Name())
	if err != nil {
		return 0, fmt.Errorf("generated configuration could not be parsed: %s", err)
	}

	err = os.Rename(tmpfile.Name(), cfile)
	if err != nil {
		return 0, fmt.Errorf("could not rename temp file %s to %s: %s", tmpfile.Name(), cfile, err)
	}

	return written, nil
}

func restart(splay time.Duration, log *logrus.Entry) {
	mu.Lock()
	defer mu.Unlock()

	log.Warnf("Restarting Choria Server after %ds splay time", splay)
	time.Sleep(splay * time.Second)

	err := syscall.Exec(os.Args[0], os.Args, os.Environ())
	if err != nil {
		log.Errorf("Could not restart server: %s", err)
	}
}
