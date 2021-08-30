package choria

import (
	"strings"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/internal/util"
)

// UserConfig determines what is the active config file for a user
func UserConfig() string {
	return util.UserConfig()
}

// NewRequestID Creates a new RequestID
func NewRequestID() (string, error) {
	return strings.Replace(util.UniqueID(), "-", "", -1), nil
}

// BuildInfo retrieves build information
func BuildInfo() *build.Info {
	return util.BuildInfo()
}

// FileExist checks if a file exist
func FileExist(path string) bool {
	return util.FileExist(path)
}
