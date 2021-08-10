package cmd

import (
	"fmt"
	"os"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/server"
	log "github.com/sirupsen/logrus"
)

func (r *serverRunCommand) Setup() (err error) {
	if broker, ok := cmdWithFullCommand("server"); ok {
		r.cmd = broker.Cmd().Command("run", "Runs a Choria Server").Default()
		r.cmd.Flag("disable-tls", "Disables TLS").Hidden().Default("false").BoolVar(&r.disableTLS)
		r.cmd.Flag("disable-ssl-verification", "Disables SSL Verification").Hidden().Default("false").BoolVar(&r.disableTLSVerify)
		r.cmd.Flag("pid", "Write running PID to a file").StringVar(&r.pidFile)
		r.cmd.Flag("service-host", "Runs as a Service Agent host").BoolVar(&r.serviceHost)
	}

	return
}

func (r *serverRunCommand) Configure() error {
	if debug {
		log.SetOutput(os.Stdout)
		log.SetLevel(log.DebugLevel)
		log.Debug("Logging at debug level due to CLI override")
	}

	if configFile == "" {
		return fmt.Errorf("server run requires a configuration file")
	}

	switch {
	case choria.FileExist(configFile):
		cfg, err = config.NewSystemConfig(configFile, true)
		if err != nil {
			return fmt.Errorf("could not parse configuration: %s", err)
		}

	case bi.ProvisionBrokerURLs() != "" || choria.FileExist(bi.ProvisionJWTFile()):
		cfg, err = config.NewDefaultSystemConfig(true)
		if err != nil {
			return fmt.Errorf("could not create default configuration for provisioning: %s", err)
		}
		cfg.ConfigFile = configFile

	default:
		return fmt.Errorf("configuration file %s was not found and provisioning is disabled", configFile)
	}

	cfg.ApplyBuildSettings(bi)

	cfg.DisableSecurityProviderVerify = true

	if os.Getenv("INSECURE_YES_REALLY") == "true" {
		protocol.Secure = "false"
		cfg.DisableTLS = true
	}

	return nil
}

func (r *serverRunCommand) prepareInstance() (i *server.Instance, err error) {
	if r.disableTLS {
		c.Config.DisableTLS = true
		log.Warn("Running with TLS disabled, not compatible with production use.")
	}

	if r.disableTLSVerify {
		c.Config.DisableTLSVerify = true
		log.Warn("Running with TLS Verification disabled, not compatible with production use.")
	}

	instance, err := server.NewInstance(c)
	if err != nil {
		return nil, fmt.Errorf("could not create Choria Server instance: %s", err)
	}

	log.Infof("Choria Server version %s starting with config %s", bi.Version(), c.Config.ConfigFile)

	if r.pidFile != "" {
		err := os.WriteFile(r.pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
		if err != nil {
			return nil, fmt.Errorf("could not write PID: %s", err)
		}
	}

	return instance, nil
}

func init() {
	cli.commands = append(cli.commands, &serverRunCommand{})
}
