package qcollector_docker_logs

import (
	"fmt"
	"strings"
	"github.com/docker/docker/api/types"
)

func SkipContainer(cjson *types.ContainerJSON, logEnv string) (skip bool, err error) {
	for _, v := range cjson.Config.Env {
		s := strings.Split(v,"=")
		if len(s) != 2 {
			err = fmt.Errorf("Could not parse environment variable '%s'", v)
			continue
		}
		if s[0] == logEnv && s[1] == "true" {
			skip = true
			break
		}
	}
	return
}
