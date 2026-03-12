package plugin

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/types"
)

// ExternalProjectType implements types.ProjectType by executing a plugin binary.
type ExternalProjectType struct {
	TypeName string
	Binary   string
	info     *externalTypeInfo
}

type externalTypeInfo struct {
	Name        string        `json:"name"`
	DisplayName string        `json:"displayName"`
	ChartPath   string        `json:"chartPath"`
	HasDB       bool          `json:"hasDatabase"`
	Laravel     bool          `json:"isLaravel"`
	Defaults    types.TypeDefaults `json:"defaults"`
}

func (e *ExternalProjectType) loadInfo() error {
	if e.info != nil {
		return nil
	}
	out, err := exec.Command(e.Binary, "info").Output()
	if err != nil {
		return fmt.Errorf("plugin %s info: %w", e.TypeName, err)
	}
	var info externalTypeInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return fmt.Errorf("parsing plugin %s info: %w", e.TypeName, err)
	}
	e.info = &info
	return nil
}

func (e *ExternalProjectType) Name() string {
	return e.TypeName
}

func (e *ExternalProjectType) DisplayName() string {
	if err := e.loadInfo(); err != nil {
		return e.TypeName
	}
	return e.info.DisplayName
}

func (e *ExternalProjectType) Detect(dir string) string {
	out, err := exec.Command(e.Binary, "detect", dir).Output()
	if err != nil {
		return ""
	}
	var result struct {
		Confidence string `json:"confidence"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return ""
	}
	return result.Confidence
}

func (e *ExternalProjectType) DefaultConfig() types.TypeDefaults {
	if err := e.loadInfo(); err != nil {
		return types.TypeDefaults{CPU: "200m", Memory: "512Mi"}
	}
	return e.info.Defaults
}

func (e *ExternalProjectType) ChartPath() string {
	if err := e.loadInfo(); err != nil {
		return "charts/" + e.TypeName
	}
	return e.info.ChartPath
}

func (e *ExternalProjectType) HasDatabase() bool {
	if err := e.loadInfo(); err != nil {
		return false
	}
	return e.info.HasDB
}

func (e *ExternalProjectType) IsLaravel() bool {
	if err := e.loadInfo(); err != nil {
		return false
	}
	return e.info.Laravel
}

func (e *ExternalProjectType) ValuesTemplate() string {
	out, err := exec.Command(e.Binary, "values-template").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func (e *ExternalProjectType) ApplicationTemplate() string {
	out, err := exec.Command(e.Binary, "application-template").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func (e *ExternalProjectType) Dockerfile(cfg config.ProjectConfig) string {
	cfgJSON, _ := json.Marshal(cfg)
	cmd := exec.Command(e.Binary, "dockerfile")
	cmd.Stdin = strings.NewReader(string(cfgJSON))
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func (e *ExternalProjectType) Dockerignore() string {
	out, err := exec.Command(e.Binary, "dockerignore").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func (e *ExternalProjectType) ScaffoldFiles(data types.ScaffoldData) map[string]string {
	dataJSON, _ := json.Marshal(data)
	cmd := exec.Command(e.Binary, "scaffold")
	cmd.Stdin = strings.NewReader(string(dataJSON))
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var files map[string]string
	if err := json.Unmarshal(out, &files); err != nil {
		return nil
	}
	return files
}
