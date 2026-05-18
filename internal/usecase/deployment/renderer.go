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
	labels := stringMap(metadata["labels"])
	annotations := stringMap(metadata["annotations"])
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
		APIVersion:  apiVersion,
		Kind:        kind,
		Name:        name,
		Namespace:   namespace,
		Labels:      labels,
		Annotations: annotations,
		SourceFile:  sourceFile,
		Index:       index,
	}, nil
}

func stringMap(value any) map[string]string {
	raw, ok := value.(map[string]any)
	if !ok || len(raw) == 0 {
		return nil
	}
	converted := make(map[string]string, len(raw))
	for key, value := range raw {
		if text, ok := value.(string); ok {
			converted[key] = text
		}
	}
	if len(converted) == 0 {
		return nil
	}
	return converted
}

func ExtractManifestImages(documents []ManifestDocument) []ManifestImage {
	var images []ManifestImage
	for _, doc := range documents {
		var manifest map[string]any
		if err := yaml.Unmarshal([]byte(doc.Content), &manifest); err != nil {
			continue
		}
		images = append(images, extractImagesFromManifest(doc.Resource, manifest)...)
	}
	return images
}

func extractImagesFromManifest(resource ManifestResourceSummary, manifest map[string]any) []ManifestImage {
	switch resource.Kind {
	case "Deployment", "StatefulSet", "DaemonSet":
		return imagesFromPodSpec(resource, nestedMap(manifest, "spec", "template", "spec"))
	case "Job":
		return imagesFromPodSpec(resource, nestedMap(manifest, "spec", "template", "spec"))
	case "CronJob":
		return imagesFromPodSpec(resource, nestedMap(manifest, "spec", "jobTemplate", "spec", "template", "spec"))
	default:
		return nil
	}
}

func imagesFromPodSpec(resource ManifestResourceSummary, podSpec map[string]any) []ManifestImage {
	var images []ManifestImage
	for _, key := range []string{"initContainers", "containers"} {
		for _, item := range listOfMaps(podSpec[key]) {
			name, _ := item["name"].(string)
			image, _ := item["image"].(string)
			if image == "" {
				continue
			}
			images = append(images, ManifestImage{
				ResourceKind: resource.Kind,
				ResourceName: resource.Name,
				Container:    name,
				Image:        image,
			})
		}
	}
	return images
}

func nestedMap(root map[string]any, keys ...string) map[string]any {
	current := root
	for _, key := range keys {
		next, _ := current[key].(map[string]any)
		if next == nil {
			return nil
		}
		current = next
	}
	return current
}

func listOfMaps(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	maps := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if mapped, ok := item.(map[string]any); ok {
			maps = append(maps, mapped)
		}
	}
	return maps
}
