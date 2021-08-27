package provision

import (
	"context"
	"fmt"
	"os"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

type JWTRequest struct {
	Token string `json:"token"`
}

type JWTReply struct {
	JWT        string `json:"jwt"`
	ECDHPublic string `json:"ecdh_public"`
}

func jwtAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	if !agent.Choria.ProvisionMode() {
		abort("Cannot reconfigure a server that is not in provisioning mode", reply)
		return
	}

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

	j, err := os.ReadFile(build.ProvisionJWTFile)
	if err != nil {
		abort(fmt.Sprintf("Could not read Provisioning JWT: %s", err), reply)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	err = updateECDHLocked()
	if err != nil {
		abort(fmt.Sprintf("Could not calculate EDCH keys: %s", err), reply)
		return
	}

	reply.Data = JWTReply{
		JWT:        string(j),
		ECDHPublic: fmt.Sprintf("%x", ecdhPublic),
	}
}
