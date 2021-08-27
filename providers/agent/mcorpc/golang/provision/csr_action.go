package provision

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

type CSRRequest struct {
	Token string `json:"token"`
	CN    string `json:"cn"`
	C     string `json:"C"`
	L     string `json:"L"`
	O     string `json:"O"`
	OU    string `json:"OU"`
	ST    string `json:"ST"`
}

type CSRReply struct {
	CSR       string `json:"csr"`
	PublicKey string `json:"public_key"`
	SSLDir    string `json:"ssldir"`
}

func csrAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
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

	args := CSRRequest{}
	if !mcorpc.ParseRequestData(&args, req, reply) {
		return
	}

	if !checkToken(args.Token, reply) {
		return
	}

	ssldir := filepath.Join(filepath.Dir(agent.Config.ConfigFile), "ssl")
	if agent.Config.Choria.SSLDir != "" {
		ssldir = agent.Config.Choria.SSLDir
	}

	keyfile := filepath.Join(ssldir, "private.pem")
	csrfile := filepath.Join(ssldir, "csr.pem")

	agent.Log.Infof("Creating a new CSR in %s", ssldir)

	err := os.MkdirAll(ssldir, 0771)
	if err != nil {
		abort(fmt.Sprintf("Could not create SSL Directory %s: %s", ssldir, err), reply)
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

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		abort(fmt.Sprintf("Could not create private key: %s", err), reply)
		return
	}
	keyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privKey),
		},
	)

	err = os.WriteFile(keyfile, keyPem, 0600)
	if err != nil {
		abort(fmt.Sprintf("Could not store private key: %s", err), reply)
		return
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, privKey)
	if err != nil {
		abort(fmt.Sprintf("Could not create CSR bytes: %s", err), reply)
		return
	}

	pb := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})

	err = os.WriteFile(csrfile, pb, 0644)
	if err != nil {
		abort(fmt.Sprintf("Could not store CSR: %s", err), reply)
		return
	}

	publicKeyDer := x509.MarshalPKCS1PublicKey(&privKey.PublicKey)
	pubKeyBlock := pem.Block{
		Type:    "PUBLIC KEY",
		Headers: nil,
		Bytes:   publicKeyDer,
	}
	pubKeyPem := string(pem.EncodeToMemory(&pubKeyBlock))

	reply.Data = &CSRReply{
		CSR:       string(pb),
		PublicKey: pubKeyPem,
		SSLDir:    ssldir,
	}
}
