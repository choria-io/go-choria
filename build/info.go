package build

type Info struct{}

func (i *Info) Version() string                   { return Version }
func (i *Info) SHA() string                       { return SHA }
func (i *Info) BuildDate() string                 { return BuildDate }
func (i *Info) License() string                   { return License }
func (i *Info) HasTLS() bool                      { return HasTLS() }
func (i *Info) MaxBrokerClients() int             { return MaxBrokerClients() }
func (i *Info) ProvisionSecurity() bool           { return ProvisionSecurity() }
func (i *Info) ProvisionDefault() bool            { return ProvisionDefault() }
func (i *Info) ProvisionBrokerURLs() string       { return ProvisionBrokerURLs }
func (i *Info) ProvisionAgent() bool              { return ProvisionAgent == "true" }
func (i *Info) ProvisionRegistrationData() string { return ProvisionRegistrationData }
func (i *Info) ProvisionFacts() string            { return ProvisionFacts }
func (i *Info) ProvisionToken() string            { return ProvisionToken }
func (i *Info) ProvisionStatusFile() string       { return ProvisionStatusFile }
func (i *Info) AgentProviders() []string          { return AgentProviders }
