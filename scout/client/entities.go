package client

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	scoutagent "github.com/choria-io/go-choria/scout/agent/scout"
	scoutclient "github.com/choria-io/go-choria/scout/client/scout"
)

// TriggerChecks triggers immediate check of some or all checks, based on the ids, facts and classes filters
func (s *ScoutAPI) TriggerChecks(ctx context.Context, checks []string, ids []string, facts []string, classes []string, cb func(r *scoutclient.TriggerOutput)) (scoutclient.Stats, error) {
	log := s.log.WithFields(logrus.Fields{"action": "trigger"})

	agent, err := scoutclient.New(scoutclient.ConfigFile(s.cfile), scoutclient.Logger(log))
	if err != nil {
		return nil, err
	}

	ci := make([]interface{}, len(checks))
	for i, c := range checks {
		ci[i] = c
	}

	res, err := agent.OptionFactFilter(facts...).OptionClassFilter(classes...).OptionIdentityFilter(ids...).Trigger().Checks(ci).Do(ctx)
	if err != nil {
		return nil, err
	}

	res.EachOutput(func(r *scoutclient.TriggerOutput) {
		cb(r)
	})

	return res.Stats(), nil
}

// PauseChecks sets checks, or all checks, to maintenance mode based on the ids, facts and classes filters
func (s *ScoutAPI) PauseChecks(ctx context.Context, checks []string, ids []string, facts []string, classes []string, cb func(r *scoutclient.MaintenanceOutput)) (scoutclient.Stats, error) {
	log := s.log.WithFields(logrus.Fields{"action": "maintenance"})

	agent, err := scoutclient.New(scoutclient.ConfigFile(s.cfile), scoutclient.Logger(log))
	if err != nil {
		return nil, err
	}

	ci := make([]interface{}, len(checks))
	for i, c := range checks {
		ci[i] = c
	}

	res, err := agent.OptionFactFilter(facts...).OptionClassFilter(classes...).OptionIdentityFilter(ids...).Maintenance().Checks(ci).Do(ctx)
	if err != nil {
		return nil, err
	}

	res.EachOutput(func(r *scoutclient.MaintenanceOutput) {
		cb(r)
	})

	return res.Stats(), nil
}

// ResumeChecks sets checks, or all checks, to resume regular checks based on the ids, facts and classes filters
func (s *ScoutAPI) ResumeChecks(ctx context.Context, checks []string, ids []string, facts []string, classes []string, cb func(r *scoutclient.ResumeOutput)) (scoutclient.Stats, error) {
	log := s.log.WithFields(logrus.Fields{"action": "resume"})

	agent, err := scoutclient.New(scoutclient.ConfigFile(s.cfile), scoutclient.Logger(log))
	if err != nil {
		return nil, err
	}

	ci := make([]interface{}, len(checks))
	for i, c := range checks {
		ci[i] = c
	}

	res, err := agent.OptionFactFilter(facts...).OptionClassFilter(classes...).OptionIdentityFilter(ids...).Resume().Checks(ci).Do(ctx)
	if err != nil {
		return nil, err
	}

	res.EachOutput(func(r *scoutclient.ResumeOutput) {
		cb(r)
	})

	return res.Stats(), nil
}

// EntityChecks retrieves the checks running on a specific entity identified by id
func (s *ScoutAPI) EntityChecks(ctx context.Context, id string) (checks []*scoutagent.CheckState, err error) {
	log := s.log.WithFields(logrus.Fields{"action": "check"})

	agent, err := scoutclient.New(scoutclient.ConfigFile(s.cfile), scoutclient.Logger(log))
	if err != nil {
		return nil, err
	}

	res, err := agent.OptionTargets([]string{id}).Checks().Do(ctx)
	if err != nil {
		return nil, err
	}

	if res.Stats().ResponsesCount() == 0 {
		return nil, fmt.Errorf("no responses received")
	}

	res.EachOutput(func(r *scoutclient.ChecksOutput) {
		if !r.ResultDetails().OK() {
			log.Errorf("Failed response received from %s: %s", r.ResultDetails().Sender(), r.ResultDetails().StatusMessage())
			return
		}

		result := &scoutagent.ChecksResponse{}
		err = r.ParseChecksOutput(result)
		if err != nil {
			log.Errorf("Could not parse response from %s: %s", r.ResultDetails().Sender(), err)
			return
		}

		checks = append(checks, result.Checks...)
	})

	return checks, nil
}
