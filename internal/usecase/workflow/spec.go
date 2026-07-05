package workflow

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadDefinitionFile(path string) (Definition, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, fmt.Errorf("read workflow definition: %w", err)
	}
	return ParseDefinition(body)
}

func ParseDefinition(body []byte) (Definition, error) {
	var def Definition
	if err := yaml.Unmarshal(body, &def); err != nil {
		return Definition{}, fmt.Errorf("decode workflow definition: %w", err)
	}
	return def, nil
}

func (t *TriggerSet) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		return nil
	}
	events := map[string]struct{}{}
	switch value.Kind {
	case yaml.ScalarNode:
		if value.Value != "" {
			events[value.Value] = struct{}{}
		}
	case yaml.SequenceNode:
		for _, item := range value.Content {
			if item.Value != "" {
				events[item.Value] = struct{}{}
			}
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(value.Content); i += 2 {
			key := value.Content[i].Value
			if key != "" {
				events[key] = struct{}{}
			}
		}
	default:
		return fmt.Errorf("workflow on must be a string, list, or map")
	}
	t.Events = sortedSet(events)
	return nil
}

func (m *Matrix) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Kind == 0 {
		return nil
	}
	if value.Kind != yaml.MappingNode {
		return errors.New("strategy.matrix must be a map")
	}
	m.Values = map[string][]string{}
	for i := 0; i+1 < len(value.Content); i += 2 {
		key := value.Content[i].Value
		item := value.Content[i+1]
		switch key {
		case "include":
			decoded, err := decodeMapList(item)
			if err != nil {
				return fmt.Errorf("decode matrix include: %w", err)
			}
			m.Include = decoded
		case "exclude":
			decoded, err := decodeMapList(item)
			if err != nil {
				return fmt.Errorf("decode matrix exclude: %w", err)
			}
			m.Exclude = decoded
		default:
			values, err := decodeStringList(item)
			if err != nil {
				return fmt.Errorf("decode matrix %s: %w", key, err)
			}
			m.Values[key] = values
		}
	}
	if len(m.Values) == 0 {
		m.Values = nil
	}
	return nil
}

func decodeStringList(node *yaml.Node) ([]string, error) {
	switch node.Kind {
	case yaml.SequenceNode:
		out := make([]string, 0, len(node.Content))
		for _, item := range node.Content {
			out = append(out, item.Value)
		}
		return out, nil
	case yaml.ScalarNode:
		if node.Value == "" {
			return nil, nil
		}
		return []string{node.Value}, nil
	default:
		return nil, errors.New("expected scalar or sequence")
	}
}

func decodeMapList(node *yaml.Node) ([]map[string]string, error) {
	if node.Kind == 0 {
		return nil, nil
	}
	if node.Kind != yaml.SequenceNode {
		return nil, errors.New("expected sequence")
	}
	out := make([]map[string]string, 0, len(node.Content))
	for _, item := range node.Content {
		if item.Kind != yaml.MappingNode {
			return nil, errors.New("expected map item")
		}
		entry := map[string]string{}
		for i := 0; i+1 < len(item.Content); i += 2 {
			entry[item.Content[i].Value] = item.Content[i+1].Value
		}
		out = append(out, entry)
	}
	return out, nil
}

func sortedSet(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}
