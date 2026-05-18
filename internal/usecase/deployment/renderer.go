package deployment

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

type ManifestRenderer interface {
	Render(ctx context.Context, manifestPaths []string, namespace string) ([]ManifestDocument, error)
}

type StaticManifestRenderer struct{}

func NewStaticManifestRenderer() StaticManifestRenderer {
	return StaticManifestRenderer{}
}

func (r StaticManifestRenderer) Render(ctx context.Context, manifestPaths []string, namespace string) ([]ManifestDocument, error) {
	var documents []ManifestDocument
	for _, path := range manifestPaths {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read manifest %q: %w", path, err)
		}
		decoder := yaml.NewDecoder(bytes.NewReader(body))
		index := 0
		for {
			var node yaml.Node
			err := decoder.Decode(&node)
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("decode manifest %q document %d: %w", path, index, err)
			}
			if emptyDocument(node) {
				index++
				continue
			}
			content, err := yaml.Marshal(&node)
			if err != nil {
				return nil, fmt.Errorf("encode manifest %q document %d: %w", path, index, err)
			}
			summary, err := summarizeManifest(path, index, content, namespace)
			if err != nil {
				return nil, err
			}
			documents = append(documents, ManifestDocument{
				SourceFile: path,
				Index:      index,
				Content:    string(content),
				Resource:   summary,
			})
			index++
		}
	}
	if len(documents) == 0 {
		return nil, fmt.Errorf("no manifest documents rendered")
	}
	return documents, nil
}

func emptyDocument(node yaml.Node) bool {
	if node.Kind == 0 {
		return true
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) == 0 {
		return true
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) == 1 {
		child := node.Content[0]
		return child.Kind == yaml.ScalarNode && child.Value == ""
	}
	return false
}

func summarizeManifest(sourceFile string, index int, content []byte, defaultNamespace string) (ManifestResourceSummary, error) {
	var manifest map[string]any
	if err := yaml.Unmarshal(content, &manifest); err != nil {
		return ManifestResourceSummary{}, fmt.Errorf("parse manifest %q document %d: %w", sourceFile, index, err)
	}
	apiVersion, _ := manifest["apiVersion"].(string)
	kind, _ := manifest["kind"].(string)
	metadata, _ := manifest["metadata"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	if namespace == "" {
		namespace = defaultNamespace
	}
	if apiVersion == "" {
		return ManifestResourceSummary{}, fmt.Errorf("manifest %q document %d apiVersion is required", sourceFile, index)
	}
	if kind == "" {
		return ManifestResourceSummary{}, fmt.Errorf("manifest %q document %d kind is required", sourceFile, index)
	}
	if name == "" {
		return ManifestResourceSummary{}, fmt.Errorf("manifest %q document %d metadata.name is required", sourceFile, index)
	}
	return ManifestResourceSummary{
		APIVersion: apiVersion,
		Kind:       kind,
		Name:       name,
		Namespace:  namespace,
		SourceFile: sourceFile,
		Index:      index,
	}, nil
}
