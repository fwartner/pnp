package plugin

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

var hookRegistry = make(map[string][]registeredHook)

type registeredHook struct {
	pluginName string
	binary     string
}

// HookData is the data passed to hook binaries on stdin.
type HookData struct {
	Event   string      `json:"event"`
	Project interface{} `json:"project,omitempty"`
	Global  interface{} `json:"global,omitempty"`
}

// RegisterHook adds a hook for the given event.
func RegisterHook(pluginName, event, binary string) {
	hookRegistry[event] = append(hookRegistry[event], registeredHook{
		pluginName: pluginName,
		binary:     binary,
	})
}

// RunHooks executes all hooks for the given event. Stops on first non-zero exit.
func RunHooks(event string, data HookData) error {
	hooks := hookRegistry[event]
	if len(hooks) == 0 {
		return nil
	}

	data.Event = event
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling hook data: %w", err)
	}

	for _, h := range hooks {
		cmd := exec.Command(h.binary)
		cmd.Stdin = strings.NewReader(string(jsonData))

		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("hook %s (%s) failed: %s: %w", h.pluginName, event, strings.TrimSpace(string(out)), err)
		}
	}

	return nil
}

// HasHooks returns true if any hooks are registered for the given event.
func HasHooks(event string) bool {
	return len(hookRegistry[event]) > 0
}
