package qtypes_messages


import (
	"fmt"
	"strings"
	"github.com/docker/docker/api/types"
)

type ContainerMessage struct {
	Message
	Container types.ContainerJSON
}


func NewContainerMessage(base Base, cnt types.ContainerJSON, msg string) ContainerMessage {
	m := NewMessage(base, msg)
	c := ContainerMessage{
		m,
		cnt,
	}
	c.ID = c.GenContainerMsgID()
	return c
}

// GenContainerMsgID uses "<container_id>-<time.UnixNano()>-<MSG>" and does a sha1 hash.
func (c *ContainerMessage) GenContainerMsgID() string {
	s := fmt.Sprintf("%s-%d-%s", c.Container.ID, c.Time.UnixNano(), c.Message)
	return Sha1HashString(s)
}

func (c *ContainerMessage) GetContainerName() string {
	if c.Container.Name != "" {
		return strings.Trim(c.Container.Name, "/")
	} else {
		return "<none>"
	}
}
