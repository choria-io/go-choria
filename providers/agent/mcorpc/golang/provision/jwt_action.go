package provision

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

type JWTRequest struct {
	Token string `json:"token"`
}

type JWTReply struct {
	JWT string `json:"jwt"`
}

func jwtAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	args := &JWTRequest{}
	if !mcorpc.ParseRequestData(args, req, reply) {
		return
	}

	if !checkToken(args.Token, reply) {
		return
	}

	if build.ProvisionJWTFile == "" {
		abort("No Provisioning JWT file has been configured", reply)
		return
	}

	if !choria.FileExist(build.ProvisionJWTFile) {
		abort("Provisioning JWT file does not exist", reply)
		return
	}

	j, err := ioutil.ReadFile(build.ProvisionJWTFile)
	if err != nil {
		abort(fmt.Sprintf("Could not read Provisioning JWT: %s", err), reply)
		return
	}

	reply.Data = JWTReply{
		JWT: string(j),
	}
}
