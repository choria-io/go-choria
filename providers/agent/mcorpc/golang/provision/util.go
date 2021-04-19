package provision

import (
	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

func abort(msg string, reply *mcorpc.Reply) {
	reply.Statuscode = mcorpc.Aborted
	reply.Statusmsg = msg
}

func checkToken(token string, reply *mcorpc.Reply) bool {
	if build.ProvisionToken == "" {
		return true
	}

	if token != build.ProvisionToken {
		log.Errorf("Incorrect Provisioning Token %s given", token)
		abort("Incorrect provision token supplied", reply)
		return false
	}

	return true
}
