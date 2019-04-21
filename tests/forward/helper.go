package forward

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
)

type RelayCmd struct {
	cmd    *exec.Cmd
	key    string
	secret string
}

func NewRelayCmd(key string, secret string) *RelayCmd {
	return &RelayCmd{
		key:    key,
		secret: secret,
	}
}

func (rc *RelayCmd) Forward(ctx context.Context, args []string) error {

	cmd := "relayd"
	fullArgs := []string{"forward"}
	fullArgs = append(fullArgs, args...)
	c := exec.CommandContext(ctx, cmd, fullArgs...)
	c.Env = []string{
		"DEBUG=true",
		"RELAY_KEY=" + rc.key,
		"RELAY_SECRET=" + rc.secret,
		"HTTP_PROXY=" + os.Getenv("HTTP_PROXY"),
		"HTTPS_PROXY=" + os.Getenv("HTTPS_PROXY"),
		"RELAY_INSECURE=" + os.Getenv("RELAY_INSECURE"),
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	rc.cmd = c

	err := c.Run()
	if err != nil {
		if strings.Contains(err.Error(), "killed") {
			return nil
		}
	}
	return err
}

func (rc *RelayCmd) Stop() error {
	defer log.Println("relayd stopped")
	return rc.cmd.Process.Kill()
}
