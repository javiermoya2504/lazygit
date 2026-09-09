package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/utils/yaml_utils"
	"gopkg.in/yaml.v3"
)

// ModelSettings edits only AI keys in the last global config file. Earlier
// files still contribute defaults, just like --use-config-file in the TUI.
type ModelSettings struct {
	AI       AIConfig
	Path     string
	document yaml.Node
	original []byte
}

func LoadModelSettings() (*ModelSettings, error) {
	paths := []string{filepath.Join(ConfigDir(), ConfigFilename)}
	custom := os.Getenv("LG_CONFIG_FILE")
	if custom != "" {
		paths = strings.Split(custom, ",")
	}
	settings := &ModelSettings{AI: GetDefaultConfigForPlatform(runtime.GOOS).AI}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil && !(os.IsNotExist(err) && custom == "") {
			return nil, err
		}
		var doc yaml.Node
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("config %s: %w", path, err)
		}
		if len(doc.Content) > 0 && doc.Content[0].Kind != yaml.MappingNode {
			return nil, fmt.Errorf("config %s must be a YAML mapping", path)
		}
		// Reject multiple documents rather than dropping part of the user's file.
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		var extra yaml.Node
		if err := decoder.Decode(&extra); err != nil && err != io.EOF {
			return nil, err
		}
		if err := decoder.Decode(&extra); err != io.EOF {
			return nil, fmt.Errorf("config %s must contain one YAML document", path)
		}
		value := struct {
			AI AIConfig `yaml:"ai"`
		}{AI: settings.AI}
		if err := yaml.Unmarshal(data, &value); err != nil {
			return nil, err
		}
		settings.AI = value.AI
		settings.Path, settings.document, settings.original = path, doc, data
	}
	return settings, nil
}

func (s *ModelSettings) SelectModel(model string) error {
	if len(s.document.Content) == 0 {
		s.document = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
	}
	root := s.document.Content[0]
	_, aiNode := yaml_utils.LookupKey(root, "ai")
	if aiNode == nil {
		aiNode = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "ai"}, aiNode)
	}
	if aiNode.Kind != yaml.MappingNode {
		return fmt.Errorf("ai configuration must be a mapping; no changes saved")
	}
	for _, setting := range []struct {
		key   string
		value any
	}{
		{"enabled", true}, {"provider", "ollama"}, {"model", model}, {"autoGenerateCommitMessage", true},
	} {
		_, node := yaml_utils.LookupKey(aiNode, setting.key)
		if node == nil {
			node = &yaml.Node{}
			aiNode.Content = append(aiNode.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: setting.key}, node)
		}
		head, line, foot := node.HeadComment, node.LineComment, node.FootComment
		if err := node.Encode(setting.value); err != nil {
			return err
		}
		node.HeadComment, node.LineComment, node.FootComment = head, line, foot
	}
	// Give a newly configured model time to load. Preserve explicit user values.
	_, timeout := yaml_utils.LookupKey(aiNode, "timeoutSeconds")
	if timeout == nil && s.AI.TimeoutSeconds == 10 {
		aiNode.Content = append(aiNode.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "timeoutSeconds"}, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "60"})
	}
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&s.document); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	current, err := os.ReadFile(s.Path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if !bytes.Equal(current, s.original) {
		return fmt.Errorf("config changed while downloading; run lazygit-ai use %s again", model)
	}
	path := s.Path
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o600)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".lazygit-ai-config-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(buf.Bytes()); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
