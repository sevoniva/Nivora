package deployment

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Definition struct {
	APIVersion string   `json:"apiVersion" yaml:"apiVersion"`
	Kind       string   `json:"kind" yaml:"kind"`
	Metadata   Metadata `json:"metadata" yaml:"metadata"`
	Spec       Spec     `json:"spec" yaml:"spec"`
}

type Metadata struct {
	Name string `json:"name" yaml:"name"`
}

type Spec struct {
	Application string     `json:"application" yaml:"application"`
	Environment string     `json:"environment" yaml:"environment"`
	Target      Target     `json:"target" yaml:"target"`
	Artifact    Artifact   `json:"artifact,omitempty" yaml:"artifact,omitempty"`
	Artifacts   []Artifact `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`
	Manifests   []string   `json:"manifests" yaml:"manifests"`
	GitOps      GitOps     `json:"gitops,omitempty" yaml:"gitops,omitempty"`
	Host        HostSpec   `json:"host,omitempty" yaml:"host,omitempty"`
	Options     Options    `json:"options,omitempty" yaml:"options,omitempty"`
}

type Target struct {
	Type               string `json:"type" yaml:"type"`
	Name               string `json:"name" yaml:"name"`
	Context            string `json:"context,omitempty" yaml:"context,omitempty"`
	Namespace          string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	ApplicationName    string `json:"applicationName,omitempty" yaml:"applicationName,omitempty"`
	RepositoryID       string `json:"repositoryId,omitempty" yaml:"repositoryId,omitempty"`
	RepositoryName     string `json:"repositoryName,omitempty" yaml:"repositoryName,omitempty"`
	RepositoryProvider string `json:"repositoryProvider,omitempty" yaml:"repositoryProvider,omitempty"`
	RepoURL            string `json:"repoURL,omitempty" yaml:"repoURL,omitempty"`
	Path               string `json:"path,omitempty" yaml:"path,omitempty"`
	Revision           string `json:"revision,omitempty" yaml:"revision,omitempty"`
	Project            string `json:"project,omitempty" yaml:"project,omitempty"`
	ClusterURL         string `json:"clusterURL,omitempty" yaml:"clusterURL,omitempty"`
	ClusterName        string `json:"clusterName,omitempty" yaml:"clusterName,omitempty"`
	SyncPolicy         string `json:"syncPolicy,omitempty" yaml:"syncPolicy,omitempty"`
	CredentialsRef     string `json:"credentialsRef,omitempty" yaml:"credentialsRef,omitempty"`
}

type HostSpec struct {
	Hosts                 []Host            `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	DeployPath            string            `json:"deployPath" yaml:"deployPath"`
	ServiceName           string            `json:"serviceName,omitempty" yaml:"serviceName,omitempty"`
	HealthCheck           string            `json:"healthCheck,omitempty" yaml:"healthCheck,omitempty"`
	HealthChecks          []HostHealthCheck `json:"healthChecks,omitempty" yaml:"healthChecks,omitempty"`
	RestartCommand        string            `json:"restartCommand,omitempty" yaml:"restartCommand,omitempty"`
	Strategy              string            `json:"strategy,omitempty" yaml:"strategy,omitempty"`
	CredentialRef         string            `json:"credentialRef,omitempty" yaml:"credentialRef,omitempty"`
	BatchSize             int               `json:"batchSize,omitempty" yaml:"batchSize,omitempty"`
	PauseOnFailure        bool              `json:"pauseOnFailure,omitempty" yaml:"pauseOnFailure,omitempty"`
	DryRun                bool              `json:"dryRun" yaml:"dryRun"`
	AllowRemoteHostDeploy bool              `json:"allowRemoteHostDeploy,omitempty" yaml:"allowRemoteHostDeploy,omitempty"`
	Labels                map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata              map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type Host struct {
	ID            string            `json:"id,omitempty" yaml:"id,omitempty"`
	Name          string            `json:"name" yaml:"name"`
	Address       string            `json:"address,omitempty" yaml:"address,omitempty"`
	EnvironmentID string            `json:"environmentId,omitempty" yaml:"environmentId,omitempty"`
	CredentialRef string            `json:"credentialRef,omitempty" yaml:"credentialRef,omitempty"`
	Labels        map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type Artifact struct {
	Name      string         `json:"name" yaml:"name"`
	Type      string         `json:"type" yaml:"type"`
	Reference string         `json:"reference" yaml:"reference"`
	Digest    string         `json:"digest,omitempty" yaml:"digest,omitempty"`
	Target    ArtifactTarget `json:"target,omitempty" yaml:"target,omitempty"`
}

type ArtifactTarget struct {
	ImageName  string `json:"imageName,omitempty" yaml:"imageName,omitempty"`
	Substitute bool   `json:"substitute,omitempty" yaml:"substitute,omitempty"`
}

type GitOps struct {
	Mode               string   `json:"mode,omitempty" yaml:"mode,omitempty"`
	WriteToWorkingTree bool     `json:"writeToWorkingTree" yaml:"writeToWorkingTree"`
	WorkingTree        string   `json:"workingTree,omitempty" yaml:"workingTree,omitempty"`
	Commit             bool     `json:"commit" yaml:"commit"`
	CommitMessage      string   `json:"commitMessage,omitempty" yaml:"commitMessage,omitempty"`
	Push               bool     `json:"push" yaml:"push"`
	AllowPush          bool     `json:"allowPush" yaml:"allowPush"`
	Remote             string   `json:"remote,omitempty" yaml:"remote,omitempty"`
	Branch             string   `json:"branch,omitempty" yaml:"branch,omitempty"`
	Rollback           bool     `json:"rollback" yaml:"rollback"`
	RollbackRevision   string   `json:"rollbackRevision,omitempty" yaml:"rollbackRevision,omitempty"`
	Sync               bool     `json:"sync" yaml:"sync"`
	AllowSync          bool     `json:"allowSync" yaml:"allowSync"`
	Prune              bool     `json:"prune" yaml:"prune"`
	Force              bool     `json:"force" yaml:"force"`
	Wait               bool     `json:"wait" yaml:"wait"`
	TimeoutSeconds     int      `json:"timeoutSeconds,omitempty" yaml:"timeoutSeconds,omitempty"`
	RequireStatus      bool     `json:"requireStatus" yaml:"requireStatus"`
	StatusRead         bool     `json:"statusRead" yaml:"statusRead"`
	Files              []string `json:"files,omitempty" yaml:"files,omitempty"`
}

type Options struct {
	DryRun               bool `json:"dryRun" yaml:"dryRun"`
	Apply                bool `json:"apply" yaml:"apply"`
	Wait                 bool `json:"wait" yaml:"wait"`
	TimeoutSeconds       int  `json:"timeoutSeconds,omitempty" yaml:"timeoutSeconds,omitempty"`
	ApprovalRequired     bool `json:"approvalRequired,omitempty" yaml:"approvalRequired,omitempty"`
	ChangeWindowRequired bool `json:"changeWindowRequired,omitempty" yaml:"changeWindowRequired,omitempty"`
}

func LoadDefinitionFile(path string) (Definition, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, fmt.Errorf("read deployment definition: %w", err)
	}
	return ParseDefinition(body)
}

func ParseDefinition(body []byte) (Definition, error) {
	var def Definition
	if err := yaml.Unmarshal(body, &def); err != nil {
		return Definition{}, fmt.Errorf("decode deployment definition: %w", err)
	}
	if err := def.Validate(); err != nil {
		return Definition{}, err
	}
	return def, nil
}

func (d Definition) Validate() error {
	if d.Kind != "Deployment" {
		return errors.New("deployment kind must be Deployment")
	}
	if d.Metadata.Name == "" {
		return errors.New("deployment metadata.name is required")
	}
	if d.Spec.Application == "" {
		return errors.New("deployment spec.application is required")
	}
	if d.Spec.Environment == "" {
		return errors.New("deployment spec.environment is required")
	}
	if d.Spec.Target.Type == "" {
		return errors.New("deployment target.type is required")
	}
	if d.Spec.Target.Type != "kubernetes-yaml" && d.Spec.Target.Type != "argocd" && d.Spec.Target.Type != "host" {
		return fmt.Errorf("deployment target.type %q is not supported in the current deployment runtime", d.Spec.Target.Type)
	}
	if d.Spec.Target.Name == "" {
		return errors.New("deployment target.name is required")
	}
	if d.Spec.Target.Type == "kubernetes-yaml" && d.Spec.Target.Namespace == "" {
		return errors.New("deployment target.namespace is required for kubernetes-yaml targets")
	}
	if d.Spec.Target.Type == "kubernetes-yaml" && len(d.Spec.Manifests) == 0 {
		return errors.New("deployment must reference at least one manifest")
	}
	if d.Spec.Target.Type == "argocd" {
		if d.Spec.Target.ApplicationName == "" {
			return errors.New("deployment target.applicationName is required for argocd targets")
		}
		if d.Spec.Target.RepoURL == "" && d.Spec.Target.RepositoryID == "" {
			return errors.New("deployment target.repoURL or target.repositoryId is required for argocd targets")
		}
		if d.Spec.Target.Path == "" {
			return errors.New("deployment target.path is required for argocd targets")
		}
	}
	if d.Spec.Target.Type == "host" {
		if d.Spec.Host.DeployPath == "" {
			return errors.New("deployment host.deployPath is required for host targets")
		}
		if d.Spec.Host.Strategy == "" {
			d.Spec.Host.Strategy = "symlink"
		}
		if len(d.Spec.Host.Hosts) == 0 {
			d.Spec.Host.Hosts = []Host{{ID: "local-noop-host", Name: d.Spec.Target.Name, EnvironmentID: d.Spec.Environment}}
		}
		if d.Spec.Artifact.Reference != "" && len(d.Spec.Artifacts) == 0 {
			d.Spec.Artifacts = []Artifact{d.Spec.Artifact}
		}
		if len(d.Spec.Artifacts) == 0 {
			return errors.New("deployment artifact is required for host targets")
		}
		if d.Spec.Host.BatchSize < 0 {
			return errors.New("deployment host.batchSize cannot be negative")
		}
		for i, check := range d.Spec.Host.HealthChecks {
			if check.Type == "" {
				return fmt.Errorf("deployment host.healthChecks[%d].type is required", i)
			}
			if check.Type != "http" && check.Type != "tcp" && check.Type != "command" {
				return fmt.Errorf("deployment host.healthChecks[%d].type %q is not supported", i, check.Type)
			}
			if check.TimeoutSeconds < 0 {
				return fmt.Errorf("deployment host.healthChecks[%d].timeoutSeconds cannot be negative", i)
			}
			if check.Type == "command" && check.Command == "" {
				return fmt.Errorf("deployment host.healthChecks[%d].command is required for command health checks", i)
			}
			if check.Type != "command" && check.Target == "" {
				return fmt.Errorf("deployment host.healthChecks[%d].target is required for %s health checks", i, check.Type)
			}
		}
	}
	for i, path := range d.Spec.Manifests {
		if path == "" {
			return fmt.Errorf("deployment manifest %d path is required", i)
		}
	}
	for i, artifact := range d.Spec.Artifacts {
		if artifact.Name == "" {
			return fmt.Errorf("deployment artifact %d name is required", i)
		}
		if artifact.Type == "" {
			return fmt.Errorf("deployment artifact %q type is required", artifact.Name)
		}
		if artifact.Reference == "" {
			return fmt.Errorf("deployment artifact %q reference is required", artifact.Name)
		}
	}
	if d.Spec.Options.Apply && d.Spec.Options.DryRun {
		return errors.New("deployment options.apply=true requires options.dryRun=false")
	}
	if d.Spec.Options.TimeoutSeconds < 0 {
		return errors.New("deployment options.timeoutSeconds cannot be negative")
	}
	if d.Spec.GitOps.Sync && d.Spec.GitOps.Mode == "" {
		d.Spec.GitOps.Mode = "plan"
	}
	return nil
}

func (o Options) dryRunOnly() bool {
	return !o.Apply
}
