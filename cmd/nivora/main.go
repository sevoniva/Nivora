package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	ociartifact "github.com/sevoniva/nivora/internal/adapters/artifact/oci"
	"github.com/sevoniva/nivora/internal/adapters/eventbus/memory"
	"github.com/sevoniva/nivora/internal/app/runner"
	"github.com/sevoniva/nivora/internal/app/server"
	"github.com/sevoniva/nivora/internal/app/worker"
	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	domainplugin "github.com/sevoniva/nivora/internal/domain/plugin"
	"github.com/sevoniva/nivora/internal/infra/config"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	credentialusecase "github.com/sevoniva/nivora/internal/usecase/credential"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	pluginusecase "github.com/sevoniva/nivora/internal/usecase/plugin"
	policyusecase "github.com/sevoniva/nivora/internal/usecase/policy"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
	"github.com/sevoniva/nivora/internal/version"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "nivora",
		Short: "Nivora DevOps delivery control plane",
	}

	root.AddCommand(newVersionCommand())
	root.AddCommand(newRunCommand("server", "configs/server.yaml", server.Run))
	root.AddCommand(newRunCommand("worker", "configs/worker.yaml", worker.Run))
	root.AddCommand(newConfigCommand())
	root.AddCommand(newAuthCommand())
	root.AddCommand(newOrgCommand())
	root.AddCommand(newProjectCommand())
	root.AddCommand(newApplicationCommand())
	root.AddCommand(newEnvironmentCommand())
	root.AddCommand(newRepositoryCommand())
	root.AddCommand(newTargetCommand())
	root.AddCommand(newApprovalsCommand())
	root.AddCommand(newChangeWindowCommand())
	root.AddCommand(newNotificationCommand())
	root.AddCommand(newCloudCommand())
	root.AddCommand(newPipelineCommand())
	root.AddCommand(newArtifactCommand())
	root.AddCommand(newReleaseCommand())
	root.AddCommand(newSecurityCommand())
	root.AddCommand(newPolicyCommand())
	root.AddCommand(newSecretCommand())
	root.AddCommand(newCredentialCommand())
	root.AddCommand(newPluginsCommand())
	root.AddCommand(newMCPCommand())
	root.AddCommand(newDeploymentCommand())
	root.AddCommand(newHostGroupsCommand())
	root.AddCommand(newGitOpsCommand())
	root.AddCommand(newArgoCDCommand())
	root.AddCommand(newRunnerCommand())
	root.AddCommand(newRuntimeCommand())
	root.AddCommand(newQuotaCommand())
	root.AddCommand(newUsageCommand())
	root.AddCommand(newAuditCommand())
	root.AddCommand(newEvidenceCommand())
	root.AddCommand(newDoctorCommand())
	return root
}

func newAuditCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "audit", Short: "Audit search and verification utilities"}
	cmd.AddCommand(newAuditSearchCommand())
	cmd.AddCommand(newAuditVerifyCommand())
	return cmd
}

func newAuditVerifyCommand() *cobra.Command {
	var serverURL string
	var scopeType string
	var scopeID string
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify tamper-evident audit hash chain",
		RunE: func(cmd *cobra.Command, args []string) error {
			values := url.Values{}
			values.Set("scopeType", scopeType)
			values.Set("scopeId", scopeID)
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/audit/verify?"+values.Encode(), nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server base URL")
	cmd.Flags().StringVar(&scopeType, "scope-type", "", "Audit scope type (pipeline, deployment, release, release_execution, auth, credential, security, approval, cloud)")
	cmd.Flags().StringVar(&scopeID, "scope-id", "", "Audit scope ID (optional)")
	return cmd
}

func newAuditSearchCommand() *cobra.Command {
	var serverURL string
	var subject string
	var actorID string
	var action string
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search audit records",
		RunE: func(cmd *cobra.Command, args []string) error {
			values := url.Values{}
			values.Set("subject", subject)
			values.Set("actorId", actorID)
			values.Set("action", action)
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/audit/search?"+values.Encode(), nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&subject, "subject", "", "subject substring")
	cmd.Flags().StringVar(&actorID, "actor-id", "", "actor id")
	cmd.Flags().StringVar(&action, "action", "", "action substring")
	return cmd
}

func newEvidenceCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "evidence", Short: "Evidence bundle utilities"}
	cmd.AddCommand(newEvidenceGenerateCommand())
	cmd.AddCommand(newEvidenceExportCommand())
	return cmd
}

func newEvidenceGenerateCommand() *cobra.Command {
	var serverURL string
	var subjectType string
	var subjectID string
	var releaseID string
	var deploymentID string
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate and persist an evidence bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			if releaseID != "" {
				subjectType = "release"
				subjectID = releaseID
			}
			if deploymentID != "" {
				subjectType = "deploymentRun"
				subjectID = deploymentID
			}
			if subjectType == "" || subjectID == "" {
				return fmt.Errorf("--subject-type and --subject-id are required unless --release or --deployment is set")
			}
			body, err := json.Marshal(map[string]string{"subjectType": subjectType, "subjectId": subjectID})
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/evidence/bundles", body)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&subjectType, "subject-type", "", "Evidence subject type")
	cmd.Flags().StringVar(&subjectID, "subject-id", "", "Evidence subject ID")
	cmd.Flags().StringVar(&releaseID, "release", "", "Generate evidence for a Release")
	cmd.Flags().StringVar(&deploymentID, "deployment", "", "Generate evidence for a DeploymentRun")
	return cmd
}

func newEvidenceExportCommand() *cobra.Command {
	var serverURL string
	var format string
	cmd := &cobra.Command{
		Use:   "export <bundle-id> | <subject-type> <subject-id>",
		Short: "Export an evidence bundle as JSON or Markdown",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			values := url.Values{}
			if format != "" {
				values.Set("format", format)
			}
			query := ""
			if encoded := values.Encode(); encoded != "" {
				query = "?" + encoded
			}
			path := "/api/v1/evidence/bundles/" + args[0] + "/export" + query
			if len(args) == 2 {
				path = "/api/v1/evidence/" + args[0] + "/" + args[1] + query
			}
			if format == "markdown" {
				body, err := doRaw(cmd.Context(), http.MethodGet, serverURL, path, nil)
				if err != nil {
					return err
				}
				_, _ = cmd.OutOrStdout().Write(body)
				return nil
			}
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, path, nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&format, "format", "json", "export format: json or markdown")
	return cmd
}

func newQuotaCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "quota", Short: "Tenant quota utilities"}
	cmd.AddCommand(newScopedGetCommand("view", "View quota for a scope", "/api/v1/tenancy/quota"))
	return cmd
}

func newUsageCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "usage", Short: "Tenant usage utilities"}
	cmd.AddCommand(newScopedGetCommand("summary", "View usage summary for a scope", "/api/v1/tenancy/usage"))
	return cmd
}

func newScopedGetCommand(name string, short string, path string) *cobra.Command {
	var serverURL string
	var scopeType string
	var scopeID string
	cmd := &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if scopeType != "" || scopeID != "" {
				values := url.Values{}
				values.Set("scopeType", scopeType)
				values.Set("scopeId", scopeID)
				query = "?" + values.Encode()
			}
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, path+query, nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&scopeType, "scope-type", "global", "tenant scope type")
	cmd.Flags().StringVar(&scopeID, "scope-id", "", "tenant scope id")
	return cmd
}

func newApprovalsCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "approvals", Short: "Approval request utilities"}
	cmd.AddCommand(newApprovalListCommand())
	cmd.AddCommand(newApprovalDecisionCommand("approve", "Approve an approval request", "/approve"))
	cmd.AddCommand(newApprovalDecisionCommand("reject", "Reject an approval request", "/reject"))
	cmd.AddCommand(newApprovalDecisionCommand("cancel", "Cancel an approval request", "/cancel"))
	cmd.AddCommand(newApprovalDecisionCommand("expire", "Expire an approval request", "/expire"))
	return cmd
}

func newApprovalListCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List approval requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/approvals", nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newApprovalDecisionCommand(name string, short string, actionPath string) *cobra.Command {
	var serverURL string
	var comment string
	var approver string
	cmd := &cobra.Command{
		Use:   name + " <id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := json.Marshal(map[string]string{"approver": approver, "comment": comment})
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/approvals/"+args[0]+actionPath, body)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&comment, "comment", "", "approval decision comment")
	cmd.Flags().StringVar(&approver, "approver", "local-user", "approver identity for local development")
	return cmd
}

func newChangeWindowCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "change-window", Short: "Change window utilities"}
	cmd.AddCommand(newChangeWindowEvaluateCommand())
	return cmd
}

func newChangeWindowEvaluateCommand() *cobra.Command {
	var serverURL string
	var environmentID string
	var at string
	cmd := &cobra.Command{
		Use:   "evaluate --env <environment-id>",
		Short: "Evaluate whether a deployment is inside an allowed change window",
		RunE: func(cmd *cobra.Command, args []string) error {
			if environmentID == "" {
				return fmt.Errorf("--env is required")
			}
			body, err := json.Marshal(map[string]string{"environmentId": environmentID, "at": at})
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/change-windows/evaluate", body)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&environmentID, "env", "", "environment id")
	cmd.Flags().StringVar(&at, "at", "", "RFC3339 evaluation time; defaults to server time")
	return cmd
}

func newNotificationCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "notification", Short: "Notification utilities"}
	cmd.AddCommand(newNotificationTestCommand())
	return cmd
}

func newNotificationTestCommand() *cobra.Command {
	var serverURL string
	var channel string
	cmd := &cobra.Command{
		Use:   "test --channel noop",
		Short: "Send a test notification through a configured provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := json.Marshal(map[string]any{"channel": channel, "type": "test", "subject": "Nivora test notification", "recipients": []string{"local"}})
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/notifications/test", body)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&channel, "channel", "noop", "notification channel")
	return cmd
}

func newCloudCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "cloud", Short: "Cloud inventory utilities"}
	cmd.AddCommand(newCloudGetCommand("providers", "List configured cloud provider types", "/api/v1/cloud/providers"))
	cmd.AddCommand(newCloudAccountCommand())
	cmd.AddCommand(newCloudAccountInspectCommand("inventory", "Get a cloud inventory snapshot", "/inventory"))
	cmd.AddCommand(newCloudAccountInspectCommand("clusters", "List cloud clusters", "/clusters"))
	cmd.AddCommand(newCloudAccountInspectCommand("hosts", "List cloud hosts", "/hosts"))
	cmd.AddCommand(newCloudAccountInspectCommand("registries", "List cloud registries", "/registries"))
	return cmd
}

func newCloudAccountCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "account", Short: "Cloud account utilities"}
	cmd.AddCommand(newCloudAccountValidateCommand())
	return cmd
}

func newCloudGetCommand(name string, short string, path string) *cobra.Command {
	var serverURL string
	var local bool
	cmd := &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			if local && path == "/api/v1/cloud/providers" {
				providers, err := server.NewCloudService().Providers(cmd.Context())
				if err != nil {
					return err
				}
				printJSON(cmd.OutOrStdout(), providers)
				return nil
			}
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, path, nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().BoolVar(&local, "local", false, "use local provider metadata without contacting a server")
	return cmd
}

func newCloudAccountValidateCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "validate <id>",
		Short: "Validate a cloud account credential reference",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/cloud/accounts/"+args[0]+"/validate", nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newCloudAccountInspectCommand(name string, short string, suffix string) *cobra.Command {
	var serverURL string
	var region string
	cmd := &cobra.Command{
		Use:   name + " <account-id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "/api/v1/cloud/accounts/" + args[0] + suffix
			if region != "" {
				path += "?region=" + region
			}
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, path, nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&region, "region", "", "optional cloud region filter")
	return cmd
}

func newPluginsCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "plugins", Short: "Plugin and adapter registry utilities"}
	cmd.AddCommand(newPluginsListCommand())
	cmd.AddCommand(newPluginsInspectCommand())
	cmd.AddCommand(newPluginsValidateCommand())
	return cmd
}

func newPluginsListCommand() *cobra.Command {
	var serverURL string
	var local bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered built-in and configured plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			if local {
				plugins, err := pluginusecase.NewDefaultRegistry().List(cmd.Context())
				if err != nil {
					return err
				}
				printJSON(cmd.OutOrStdout(), plugins)
				return nil
			}
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/plugins", nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().BoolVar(&local, "local", false, "use the local built-in registry without contacting a server")
	return cmd
}

func newPluginsInspectCommand() *cobra.Command {
	var serverURL string
	var local bool
	cmd := &cobra.Command{
		Use:   "inspect <name>",
		Short: "Inspect a plugin manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if local {
				manifest, err := pluginusecase.NewDefaultRegistry().Get(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				printJSON(cmd.OutOrStdout(), manifest)
				return nil
			}
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/plugins/"+args[0], nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().BoolVar(&local, "local", false, "use the local built-in registry without contacting a server")
	return cmd
}

func newPluginsValidateCommand() *cobra.Command {
	var serverURL string
	var file string
	var local bool
	cmd := &cobra.Command{
		Use:   "validate --file <plugin.yaml>",
		Short: "Validate a plugin manifest for API and version compatibility",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			body, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			var manifest domainplugin.Manifest
			if err := yaml.Unmarshal(body, &manifest); err != nil {
				if jsonErr := json.Unmarshal(body, &manifest); jsonErr != nil {
					return err
				}
			}
			if local {
				result, err := pluginusecase.NewDefaultRegistry().Validate(cmd.Context(), manifest)
				if err != nil {
					return err
				}
				printJSON(cmd.OutOrStdout(), result)
				if !result.Valid {
					return fmt.Errorf("plugin manifest is not compatible")
				}
				return nil
			}
			payload, err := json.Marshal(manifest)
			if err != nil {
				return err
			}
			result, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/plugins/validate", payload)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), result)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&file, "file", "", "plugin manifest file")
	cmd.Flags().BoolVar(&local, "local", false, "validate with the local built-in compatibility rules")
	return cmd
}

func newAuthCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Authentication and RBAC utilities"}
	cmd.AddCommand(newAuthInspectCommand("login-token", "Validate a bearer token from an environment variable", "/api/v1/auth/whoami"))
	cmd.AddCommand(newAuthInspectCommand("whoami", "Show the current authenticated subject", "/api/v1/auth/whoami"))
	cmd.AddCommand(newAuthInspectCommand("permissions", "List known permissions", "/api/v1/auth/permissions"))
	cmd.AddCommand(newAuthInspectCommand("token-info", "Show token metadata without printing token values", "/api/v1/auth/token-info"))
	cmd.AddCommand(newAuthServiceAccountCommand())
	cmd.AddCommand(newAuthTokenCommand())
	return cmd
}

func newAuthServiceAccountCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "service-account", Short: "Service account utilities"}
	cmd.AddCommand(newAuthServiceAccountListCommand())
	cmd.AddCommand(newAuthServiceAccountCreateCommand())
	return cmd
}

func newAuthServiceAccountListCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List service accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSONWithToken(cmd.Context(), http.MethodGet, serverURL, "/api/v1/service-accounts", nil, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	return cmd
}

func newAuthServiceAccountCreateCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	var name string
	var role string
	var scopeType string
	var scopeID string
	cmd := &cobra.Command{
		Use:   "create --name <name>",
		Short: "Create a service account",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			body, err := json.Marshal(map[string]string{"name": name, "role": role, "scopeType": scopeType, "scopeId": scopeID})
			if err != nil {
				return err
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodPost, serverURL, "/api/v1/service-accounts", body, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	cmd.Flags().StringVar(&name, "name", "", "service account name")
	cmd.Flags().StringVar(&role, "role", "developer", "service account role")
	cmd.Flags().StringVar(&scopeType, "scope-type", "project", "service account scope type")
	cmd.Flags().StringVar(&scopeID, "scope-id", "", "service account scope id")
	return cmd
}

func newAuthTokenCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "token", Short: "API token utilities"}
	cmd.AddCommand(newAuthTokenCreateCommand())
	cmd.AddCommand(newAuthTokenListCommand())
	cmd.AddCommand(newAuthTokenRotateCommand())
	cmd.AddCommand(newAuthTokenRevokeCommand())
	return cmd
}

func newAuthTokenCreateCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	var name string
	var subjectID string
	cmd := &cobra.Command{
		Use:   "create --subject-id <service-account-id>",
		Short: "Create an API token; the raw token is printed only once",
		RunE: func(cmd *cobra.Command, args []string) error {
			if subjectID == "" {
				return fmt.Errorf("--subject-id is required")
			}
			body, err := json.Marshal(map[string]string{"name": name, "subjectId": subjectID})
			if err != nil {
				return err
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodPost, serverURL, "/api/v1/auth/tokens", body, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	cmd.Flags().StringVar(&name, "name", "", "token name")
	cmd.Flags().StringVar(&subjectID, "subject-id", "", "service account subject id")
	return cmd
}

func newAuthTokenListCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List API token metadata without token values",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSONWithToken(cmd.Context(), http.MethodGet, serverURL, "/api/v1/auth/tokens", nil, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	return cmd
}

func newAuthTokenRotateCommand() *cobra.Command {
	return newAuthTokenMutateCommand("rotate", "Rotate an API token; the raw token is printed only once", "/rotate")
}

func newAuthTokenRevokeCommand() *cobra.Command {
	return newAuthTokenMutateCommand("revoke", "Revoke an API token", "/revoke")
}

func newAuthTokenMutateCommand(name string, short string, suffix string) *cobra.Command {
	var serverURL string
	var tokenEnv string
	cmd := &cobra.Command{
		Use:   name + " <token-id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSONWithToken(cmd.Context(), http.MethodPost, serverURL, "/api/v1/auth/tokens/"+args[0]+suffix, nil, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	return cmd
}

func newAuthInspectCommand(name string, short string, path string) *cobra.Command {
	var serverURL string
	var tokenEnv string
	cmd := &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSONWithToken(cmd.Context(), http.MethodGet, serverURL, path, nil, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	return cmd
}

func newProjectCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "project", Short: "Project utilities"}
	cmd.AddCommand(newCatalogListCommand("list", "List projects", "/api/v1/projects", "org-id"))
	cmd.AddCommand(newCatalogGetCommand("get", "Get a project", "/api/v1/projects"))
	cmd.AddCommand(newProjectCreateCommand())
	cmd.AddCommand(newCatalogUpdateCommand("update", "Update a project", "/api/v1/projects"))
	cmd.AddCommand(newCatalogDisableCommand("disable", "Disable a project", "/api/v1/projects"))
	members := &cobra.Command{Use: "members", Short: "Project membership utilities"}
	members.AddCommand(newProjectMembersListCommand())
	cmd.AddCommand(members)
	return cmd
}

func newOrgCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "org", Short: "Organization catalog utilities"}
	cmd.AddCommand(newCatalogListCommand("list", "List organizations", "/api/v1/orgs", ""))
	cmd.AddCommand(newCatalogGetCommand("get", "Get an organization", "/api/v1/orgs"))
	cmd.AddCommand(newOrgCreateCommand())
	cmd.AddCommand(newCatalogUpdateCommand("update", "Update an organization", "/api/v1/orgs"))
	cmd.AddCommand(newCatalogDisableCommand("disable", "Disable an organization", "/api/v1/orgs"))
	return cmd
}

func newApplicationCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "application", Aliases: []string{"app"}, Short: "Application catalog utilities"}
	cmd.AddCommand(newCatalogListCommand("list", "List applications", "/api/v1/applications", "project-id"))
	cmd.AddCommand(newCatalogGetCommand("get", "Get an application", "/api/v1/applications"))
	cmd.AddCommand(newApplicationCreateCommand())
	cmd.AddCommand(newCatalogUpdateCommand("update", "Update an application", "/api/v1/applications"))
	cmd.AddCommand(newCatalogDisableCommand("disable", "Disable an application", "/api/v1/applications"))
	return cmd
}

func newEnvironmentCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "environment", Aliases: []string{"env"}, Short: "Environment catalog utilities"}
	cmd.AddCommand(newCatalogListCommand("list", "List environments", "/api/v1/environments", "project-id"))
	cmd.AddCommand(newCatalogGetCommand("get", "Get an environment", "/api/v1/environments"))
	cmd.AddCommand(newEnvironmentCreateCommand())
	cmd.AddCommand(newCatalogUpdateCommand("update", "Update an environment", "/api/v1/environments"))
	cmd.AddCommand(newCatalogDisableCommand("disable", "Disable an environment", "/api/v1/environments"))
	return cmd
}

func newRepositoryCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "repository", Aliases: []string{"repo"}, Short: "SCM repository catalog utilities"}
	cmd.AddCommand(newCatalogListCommand("list", "List repositories", "/api/v1/repositories", "project-id"))
	cmd.AddCommand(newCatalogGetCommand("get", "Get a repository", "/api/v1/repositories"))
	cmd.AddCommand(newRepositoryCreateCommand())
	cmd.AddCommand(newRepositoryUpdateCommand())
	cmd.AddCommand(newCatalogDisableCommand("disable", "Disable a repository", "/api/v1/repositories"))
	return cmd
}

func newTargetCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "target", Short: "Release target catalog utilities"}
	cmd.AddCommand(newTargetListCommand())
	cmd.AddCommand(newCatalogGetCommand("get", "Get a release target", "/api/v1/release-targets"))
	cmd.AddCommand(newTargetCreateCommand())
	cmd.AddCommand(newTargetUpdateCommand())
	cmd.AddCommand(newCatalogDisableCommand("disable", "Disable a release target", "/api/v1/release-targets"))
	cmd.AddCommand(newTargetValidateCommand())
	return cmd
}

func newTargetListCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	var projectID string
	var environmentID string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List release targets",
		RunE: func(cmd *cobra.Command, args []string) error {
			values := url.Values{}
			if projectID != "" {
				values.Set("projectId", projectID)
			}
			if environmentID != "" {
				values.Set("environmentId", environmentID)
			}
			query := ""
			if len(values) > 0 {
				query = "?" + values.Encode()
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodGet, serverURL, "/api/v1/release-targets"+query, nil, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	cmd.Flags().StringVar(&projectID, "project-id", "", "project id filter")
	cmd.Flags().StringVar(&environmentID, "environment-id", "", "environment id filter")
	return cmd
}

func newOrgCreateCommand() *cobra.Command {
	return newCatalogCreateCommand("create", "Create an organization", "/api/v1/orgs", "")
}

func newProjectCreateCommand() *cobra.Command {
	return newCatalogCreateCommand("create", "Create a project", "/api/v1/projects", "org-id")
}

func newApplicationCreateCommand() *cobra.Command {
	return newCatalogCreateCommand("create", "Create an application", "/api/v1/applications", "project-id")
}

func newEnvironmentCreateCommand() *cobra.Command {
	return newCatalogCreateCommand("create", "Create an environment", "/api/v1/environments", "project-id")
}

func newCatalogListCommand(name string, short string, path string, parentFlag string) *cobra.Command {
	var serverURL string
	var tokenEnv string
	var parentID string
	cmd := &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if parentFlag != "" && parentID != "" {
				query = "?" + url.Values{catalogParentQueryKey(parentFlag): []string{parentID}}.Encode()
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodGet, serverURL, path+query, nil, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	if parentFlag != "" {
		cmd.Flags().StringVar(&parentID, parentFlag, "", "parent resource id filter")
	}
	return cmd
}

func catalogParentQueryKey(parentFlag string) string {
	switch parentFlag {
	case "org-id":
		return "orgId"
	case "project-id":
		return "projectId"
	default:
		return strings.ReplaceAll(parentFlag, "-", "")
	}
}

func newCatalogGetCommand(name string, short string, path string) *cobra.Command {
	var serverURL string
	var tokenEnv string
	cmd := &cobra.Command{
		Use:   name + " <id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSONWithToken(cmd.Context(), http.MethodGet, serverURL, path+"/"+args[0], nil, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	return cmd
}

func newCatalogCreateCommand(name string, short string, path string, parentFlag string) *cobra.Command {
	var serverURL string
	var tokenEnv string
	var resourceName string
	var description string
	var parentID string
	cmd := &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			if resourceName == "" {
				return fmt.Errorf("--name is required")
			}
			bodyMap := map[string]any{"name": resourceName}
			if description != "" {
				bodyMap["description"] = description
			}
			switch parentFlag {
			case "org-id":
				if parentID == "" {
					return fmt.Errorf("--org-id is required")
				}
				bodyMap["orgId"] = parentID
			case "project-id":
				if parentID == "" {
					return fmt.Errorf("--project-id is required")
				}
				bodyMap["projectId"] = parentID
			}
			body, err := json.Marshal(bodyMap)
			if err != nil {
				return err
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodPost, serverURL, path, body, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	cmd.Flags().StringVar(&resourceName, "name", "", "resource name")
	cmd.Flags().StringVar(&description, "description", "", "resource description")
	if parentFlag != "" {
		cmd.Flags().StringVar(&parentID, parentFlag, "", "parent resource id")
	}
	return cmd
}

func newCatalogUpdateCommand(name string, short string, path string) *cobra.Command {
	var serverURL string
	var tokenEnv string
	var resourceName string
	var description string
	var enabled bool
	cmd := &cobra.Command{
		Use:   name + " <id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bodyMap := map[string]any{}
			if cmd.Flags().Changed("name") {
				bodyMap["name"] = resourceName
			}
			if cmd.Flags().Changed("description") {
				bodyMap["description"] = description
			}
			if cmd.Flags().Changed("enabled") {
				bodyMap["enabled"] = enabled
			}
			if len(bodyMap) == 0 {
				return fmt.Errorf("at least one of --name, --description, or --enabled must be set")
			}
			body, err := json.Marshal(bodyMap)
			if err != nil {
				return err
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodPatch, serverURL, path+"/"+args[0], body, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	cmd.Flags().StringVar(&resourceName, "name", "", "resource name")
	cmd.Flags().StringVar(&description, "description", "", "resource description")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "resource enabled state")
	return cmd
}

func newCatalogDisableCommand(name string, short string, path string) *cobra.Command {
	var serverURL string
	var tokenEnv string
	cmd := &cobra.Command{
		Use:   name + " <id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSONWithToken(cmd.Context(), http.MethodDelete, serverURL, path+"/"+args[0], nil, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	return cmd
}

func newRepositoryCreateCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	var projectID string
	var name string
	var repoURL string
	var provider string
	var defaultBranch string
	var credentialRef string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an SCM repository catalog record",
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectID == "" {
				return fmt.Errorf("--project-id is required")
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if repoURL == "" {
				return fmt.Errorf("--url is required")
			}
			bodyMap := map[string]any{"projectId": projectID, "name": name, "url": repoURL}
			if provider != "" {
				bodyMap["provider"] = provider
			}
			if defaultBranch != "" {
				bodyMap["defaultBranch"] = defaultBranch
			}
			if credentialRef != "" {
				bodyMap["credentialRef"] = credentialRef
			}
			body, err := json.Marshal(bodyMap)
			if err != nil {
				return err
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodPost, serverURL, "/api/v1/repositories", body, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	cmd.Flags().StringVar(&projectID, "project-id", "", "parent project id")
	cmd.Flags().StringVar(&name, "name", "", "repository name")
	cmd.Flags().StringVar(&repoURL, "url", "", "repository URL")
	cmd.Flags().StringVar(&provider, "provider", "generic", "SCM provider name")
	cmd.Flags().StringVar(&defaultBranch, "default-branch", "main", "default branch")
	cmd.Flags().StringVar(&credentialRef, "credential-ref", "", "CredentialRef id for future SCM access")
	return cmd
}

func newRepositoryUpdateCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	var name string
	var repoURL string
	var provider string
	var defaultBranch string
	var credentialRef string
	var enabled bool
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an SCM repository catalog record",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bodyMap := map[string]any{}
			if cmd.Flags().Changed("name") {
				bodyMap["name"] = name
			}
			if cmd.Flags().Changed("url") {
				bodyMap["url"] = repoURL
			}
			if cmd.Flags().Changed("provider") {
				bodyMap["provider"] = provider
			}
			if cmd.Flags().Changed("default-branch") {
				bodyMap["defaultBranch"] = defaultBranch
			}
			if cmd.Flags().Changed("credential-ref") {
				bodyMap["credentialRef"] = credentialRef
			}
			if cmd.Flags().Changed("enabled") {
				bodyMap["enabled"] = enabled
			}
			if len(bodyMap) == 0 {
				return fmt.Errorf("at least one update flag must be set")
			}
			body, err := json.Marshal(bodyMap)
			if err != nil {
				return err
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodPatch, serverURL, "/api/v1/repositories/"+args[0], body, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	cmd.Flags().StringVar(&name, "name", "", "repository name")
	cmd.Flags().StringVar(&repoURL, "url", "", "repository URL")
	cmd.Flags().StringVar(&provider, "provider", "", "SCM provider name")
	cmd.Flags().StringVar(&defaultBranch, "default-branch", "", "default branch")
	cmd.Flags().StringVar(&credentialRef, "credential-ref", "", "CredentialRef id for future SCM access")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "resource enabled state")
	return cmd
}

func newTargetCreateCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	var environmentID string
	var name string
	var targetType string
	var contextName string
	var namespace string
	var configRef string
	var credentialRef string
	var allowApply bool
	var allowSync bool
	var allowRemoteHostDeploy bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a release target catalog record",
		RunE: func(cmd *cobra.Command, args []string) error {
			if environmentID == "" {
				return fmt.Errorf("--environment-id is required")
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if targetType == "" {
				return fmt.Errorf("--type is required")
			}
			bodyMap := map[string]any{"environmentId": environmentID, "name": name, "targetType": targetType}
			if contextName != "" {
				bodyMap["context"] = contextName
			}
			if namespace != "" {
				bodyMap["namespace"] = namespace
			}
			if configRef != "" {
				bodyMap["configRef"] = configRef
			}
			if credentialRef != "" {
				bodyMap["credentialRef"] = credentialRef
			}
			if cmd.Flags().Changed("allow-apply") {
				bodyMap["allowApply"] = allowApply
			}
			if cmd.Flags().Changed("allow-sync") {
				bodyMap["allowSync"] = allowSync
			}
			if cmd.Flags().Changed("allow-remote-host-deploy") {
				bodyMap["allowRemoteHostDeploy"] = allowRemoteHostDeploy
			}
			body, err := json.Marshal(bodyMap)
			if err != nil {
				return err
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodPost, serverURL, "/api/v1/release-targets", body, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	cmd.Flags().StringVar(&environmentID, "environment-id", "", "parent environment id")
	cmd.Flags().StringVar(&name, "name", "", "target name")
	cmd.Flags().StringVar(&targetType, "type", "", "target type: kubernetes-yaml, argocd, host, webhook, or noop")
	cmd.Flags().StringVar(&contextName, "context", "", "target context name")
	cmd.Flags().StringVar(&namespace, "namespace", "", "target namespace")
	cmd.Flags().StringVar(&configRef, "config-ref", "", "config reference id")
	cmd.Flags().StringVar(&credentialRef, "credential-ref", "", "CredentialRef id")
	cmd.Flags().BoolVar(&allowApply, "allow-apply", false, "explicitly allow Kubernetes apply for this target")
	cmd.Flags().BoolVar(&allowSync, "allow-sync", false, "explicitly allow Argo CD sync for this target")
	cmd.Flags().BoolVar(&allowRemoteHostDeploy, "allow-remote-host-deploy", false, "explicitly allow remote host deployment for this target")
	return cmd
}

func newTargetUpdateCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	var name string
	var targetType string
	var contextName string
	var namespace string
	var configRef string
	var credentialRef string
	var allowApply bool
	var allowSync bool
	var allowRemoteHostDeploy bool
	var enabled bool
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a release target catalog record",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bodyMap := map[string]any{}
			if cmd.Flags().Changed("name") {
				bodyMap["name"] = name
			}
			if cmd.Flags().Changed("type") {
				bodyMap["targetType"] = targetType
			}
			if cmd.Flags().Changed("context") {
				bodyMap["context"] = contextName
			}
			if cmd.Flags().Changed("namespace") {
				bodyMap["namespace"] = namespace
			}
			if cmd.Flags().Changed("config-ref") {
				bodyMap["configRef"] = configRef
			}
			if cmd.Flags().Changed("credential-ref") {
				bodyMap["credentialRef"] = credentialRef
			}
			if cmd.Flags().Changed("allow-apply") {
				bodyMap["allowApply"] = allowApply
			}
			if cmd.Flags().Changed("allow-sync") {
				bodyMap["allowSync"] = allowSync
			}
			if cmd.Flags().Changed("allow-remote-host-deploy") {
				bodyMap["allowRemoteHostDeploy"] = allowRemoteHostDeploy
			}
			if cmd.Flags().Changed("enabled") {
				bodyMap["enabled"] = enabled
			}
			if len(bodyMap) == 0 {
				return fmt.Errorf("at least one update flag must be set")
			}
			body, err := json.Marshal(bodyMap)
			if err != nil {
				return err
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodPatch, serverURL, "/api/v1/release-targets/"+args[0], body, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	cmd.Flags().StringVar(&name, "name", "", "target name")
	cmd.Flags().StringVar(&targetType, "type", "", "target type")
	cmd.Flags().StringVar(&contextName, "context", "", "target context name")
	cmd.Flags().StringVar(&namespace, "namespace", "", "target namespace")
	cmd.Flags().StringVar(&configRef, "config-ref", "", "config reference id")
	cmd.Flags().StringVar(&credentialRef, "credential-ref", "", "CredentialRef id")
	cmd.Flags().BoolVar(&allowApply, "allow-apply", false, "explicitly allow Kubernetes apply for this target")
	cmd.Flags().BoolVar(&allowSync, "allow-sync", false, "explicitly allow Argo CD sync for this target")
	cmd.Flags().BoolVar(&allowRemoteHostDeploy, "allow-remote-host-deploy", false, "explicitly allow remote host deployment for this target")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "target enabled state")
	return cmd
}

func newTargetValidateCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	cmd := &cobra.Command{
		Use:   "validate <id>",
		Short: "Validate a release target for use",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSONWithToken(cmd.Context(), http.MethodPost, serverURL, "/api/v1/release-targets/"+args[0]+"/validate", nil, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	return cmd
}

func newProjectMembersListCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	cmd := &cobra.Command{
		Use:   "list <project-id>",
		Short: "List project members",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSONWithToken(cmd.Context(), http.MethodGet, serverURL, "/api/v1/projects/"+args[0]+"/members", nil, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	return cmd
}

func newSecretCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "secret", Short: "Secret reference utilities"}
	cmd.AddCommand(newSecretPutCommand())
	cmd.AddCommand(newSecretRotateCommand())
	cmd.AddCommand(newSecretListCommand())
	cmd.AddCommand(newSecretProviderCommand())
	cmd.AddCommand(newSecretDeleteCommand())
	return cmd
}

func newSecretPutCommand() *cobra.Command {
	var valueEnv string
	var scopeType string
	var scopeID string
	var key string
	var serverURL string
	var local bool
	cmd := &cobra.Command{
		Use:   "put --name <name> --value-env <ENV_NAME>",
		Short: "Store a secret value and return only its SecretRef",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if valueEnv == "" {
				return fmt.Errorf("--value-env is required; inline secret values are intentionally unsupported")
			}
			value, ok := os.LookupEnv(valueEnv)
			if !ok {
				return fmt.Errorf("environment variable %s is not set", valueEnv)
			}
			input := credentialusecase.SecretCreateInput{Name: name, ScopeType: scopeType, ScopeID: scopeID, Key: key, Value: value}
			if local {
				ref, err := server.NewCredentialService().PutSecret(cmd.Context(), input)
				if err != nil {
					return err
				}
				printJSON(cmd.OutOrStdout(), ref)
				return nil
			}
			body, err := json.Marshal(input)
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/secrets", body)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().String("name", "", "secret name")
	cmd.Flags().StringVar(&valueEnv, "value-env", "", "environment variable containing the secret value")
	cmd.Flags().StringVar(&scopeType, "scope-type", "global", "secret scope type")
	cmd.Flags().StringVar(&scopeID, "scope-id", "", "secret scope id")
	cmd.Flags().StringVar(&key, "key", "", "provider key for the secret")
	cmd.Flags().BoolVar(&local, "local", false, "use an in-process dev provider")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newSecretRotateCommand() *cobra.Command {
	var valueEnv string
	var serverURL string
	cmd := &cobra.Command{
		Use:   "rotate <secret-id> --value-env <ENV_NAME>",
		Short: "Rotate a secret value and return only updated SecretRef metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if valueEnv == "" {
				return fmt.Errorf("--value-env is required; inline secret values are intentionally unsupported")
			}
			value, ok := os.LookupEnv(valueEnv)
			if !ok {
				return fmt.Errorf("environment variable %s is not set", valueEnv)
			}
			body, err := json.Marshal(credentialusecase.SecretRotateInput{Value: value})
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/secrets/"+args[0]+"/rotate", body)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&valueEnv, "value-env", "", "environment variable containing the new secret value")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newSecretListCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List SecretRefs from a Nivora server",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/secrets/refs", nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newSecretProviderCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "provider", Short: "Secret provider utilities"}
	cmd.AddCommand(newSecretProviderValidateCommand())
	return cmd
}

func newSecretProviderValidateCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate the configured secret provider without returning secret values",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/secrets/provider/validate", nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newSecretDeleteCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "delete <secret-id>",
		Short: "Delete a secret by id on a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodDelete, serverURL, "/api/v1/secrets/"+args[0], nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newCredentialCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "credential", Short: "Credential metadata utilities"}
	cmd.AddCommand(newCredentialCreateCommand())
	cmd.AddCommand(newCredentialValidateCommand())
	return cmd
}

func newCredentialCreateCommand() *cobra.Command {
	var file string
	var local bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   "create --file <credential.yaml>",
		Short: "Create credential metadata bound to a SecretRef",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			def, err := credentialusecase.LoadDefinitionFile(file)
			if err != nil {
				return err
			}
			input := def.CreateInput()
			if local {
				cred, err := server.NewCredentialService().CreateCredential(cmd.Context(), input)
				if err != nil {
					return err
				}
				printJSON(cmd.OutOrStdout(), cred)
				return nil
			}
			body, err := json.Marshal(input)
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/credentials", body)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "credential definition file")
	cmd.Flags().BoolVar(&local, "local", false, "create in an in-process dev store")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newCredentialValidateCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "validate <credential-id>",
		Short: "Validate that a credential SecretRef can be resolved",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/credentials/"+args[0]+"/validate", nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newSecurityCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "security", Short: "Security scan utilities"}
	scan := &cobra.Command{Use: "scan", Short: "Run local security scans"}
	scan.AddCommand(newSecurityScanArtifactCommand())
	scan.AddCommand(newSecurityScanManifestCommand())
	scans := &cobra.Command{Use: "scans", Short: "Query stored security scans"}
	scans.AddCommand(newSecurityScansListCommand())
	findings := &cobra.Command{Use: "findings", Short: "Query stored security findings"}
	findings.AddCommand(newSecurityFindingsListCommand())
	cmd.AddCommand(scan)
	cmd.AddCommand(scans)
	cmd.AddCommand(findings)
	return cmd
}

func newSecurityScanArtifactCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "artifact <reference> --local",
		Short: "Run a local artifact security scan through the noop scanner",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			record, err := server.NewSecurityService().Scan(cmd.Context(), securityusecase.ScanInput{SubjectType: "artifact", SubjectID: args[0], Reference: args[0]})
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), record)
			return nil
		},
	}
	cmd.Flags().Bool("local", true, "use the in-process Phase 3.0 noop scanner")
	return cmd
}

func newSecurityScanManifestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manifest <manifest.yaml> --local",
		Short: "Run a local manifest security scan through the noop scanner and built-in manifest checks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			record, err := server.NewSecurityService().Scan(cmd.Context(), securityusecase.ScanInput{SubjectType: "manifest", SubjectID: args[0], Content: string(body)})
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), record)
			return nil
		},
	}
	cmd.Flags().Bool("local", true, "use the in-process Phase 3.0 noop scanner")
	return cmd
}

func newSecurityScansListCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	var subjectType string
	var subjectID string
	var status string
	var limit int
	var offset int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List stored security scans from the Nivora API",
		RunE: func(cmd *cobra.Command, args []string) error {
			values := url.Values{}
			if subjectType != "" {
				values.Set("subjectType", subjectType)
			}
			if subjectID != "" {
				values.Set("subjectId", subjectID)
			}
			if status != "" {
				values.Set("status", status)
			}
			if cmd.Flags().Changed("limit") {
				values.Set("limit", fmt.Sprintf("%d", limit))
			}
			if cmd.Flags().Changed("offset") {
				values.Set("offset", fmt.Sprintf("%d", offset))
			}
			query := ""
			if len(values) > 0 {
				query = "?" + values.Encode()
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodGet, serverURL, "/api/v1/security/scans"+query, nil, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	cmd.Flags().StringVar(&subjectType, "subject-type", "", "filter by subject type")
	cmd.Flags().StringVar(&subjectID, "subject-id", "", "filter by subject id")
	cmd.Flags().StringVar(&status, "status", "", "filter by scan status")
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum rows to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "rows to skip")
	return cmd
}

func newSecurityFindingsListCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	var scanID string
	var subjectType string
	var subjectID string
	var severity string
	var category string
	var limit int
	var offset int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List stored security findings from the Nivora API",
		RunE: func(cmd *cobra.Command, args []string) error {
			values := url.Values{}
			if scanID != "" {
				values.Set("scanId", scanID)
			}
			if subjectType != "" {
				values.Set("subjectType", subjectType)
			}
			if subjectID != "" {
				values.Set("subjectId", subjectID)
			}
			if severity != "" {
				values.Set("severity", severity)
			}
			if category != "" {
				values.Set("category", category)
			}
			if cmd.Flags().Changed("limit") {
				values.Set("limit", fmt.Sprintf("%d", limit))
			}
			if cmd.Flags().Changed("offset") {
				values.Set("offset", fmt.Sprintf("%d", offset))
			}
			query := ""
			if len(values) > 0 {
				query = "?" + values.Encode()
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodGet, serverURL, "/api/v1/security/findings"+query, nil, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	cmd.Flags().StringVar(&scanID, "scan-id", "", "filter by scan id")
	cmd.Flags().StringVar(&subjectType, "subject-type", "", "filter by subject type")
	cmd.Flags().StringVar(&subjectID, "subject-id", "", "filter by subject id")
	cmd.Flags().StringVar(&severity, "severity", "", "filter by severity")
	cmd.Flags().StringVar(&category, "category", "", "filter by category")
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum rows to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "rows to skip")
	return cmd
}

func newPolicyCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "policy", Short: "Policy gate utilities"}
	cmd.AddCommand(newPolicyListCommand())
	cmd.AddCommand(newPolicyCreateCommand())
	cmd.AddCommand(newPolicyGetCommand())
	cmd.AddCommand(newPolicyUpdateCommand())
	cmd.AddCommand(newPolicyDisableCommand())
	evaluate := &cobra.Command{
		Use:   "evaluate --subject <reference>",
		Short: "Evaluate built-in policy gates locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			subject, _ := cmd.Flags().GetString("subject")
			requireDigest, _ := cmd.Flags().GetBool("require-digest")
			if subject == "" {
				return fmt.Errorf("--subject is required")
			}
			result := server.NewSecurityService().Evaluate(securityusecase.EvaluateInput{
				SubjectType: "artifact",
				SubjectID:   subject,
				Reference:   subject,
				Policy:      securityusecase.PolicyConfig{CriticalDenyThreshold: 1, HighWarnThreshold: 1, RequireDigest: requireDigest},
			})
			printJSON(cmd.OutOrStdout(), result)
			return nil
		},
	}
	evaluate.Flags().String("subject", "", "artifact reference or subject")
	evaluate.Flags().Bool("require-digest", false, "deny mutable artifact references without sha256 digest")
	cmd.AddCommand(evaluate)
	return cmd
}

func newPolicyListCommand() *cobra.Command {
	var projectID string
	var environmentID string
	var serverURL string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List policy definitions from a Nivora server",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{}
			if projectID != "" {
				query.Set("projectId", projectID)
			}
			if environmentID != "" {
				query.Set("environmentId", environmentID)
			}
			path := "/api/v1/policies"
			if encoded := query.Encode(); encoded != "" {
				path += "?" + encoded
			}
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, path, nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&projectID, "project-id", "", "filter policies by project id")
	cmd.Flags().StringVar(&environmentID, "environment-id", "", "filter policies by environment id")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newPolicyCreateCommand() *cobra.Command {
	var input policyusecase.CreateInput
	var serverURL string
	cmd := &cobra.Command{
		Use:   "create --name <name>",
		Short: "Create a built-in policy gate definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			if input.Name == "" {
				return fmt.Errorf("--name is required")
			}
			body, err := json.Marshal(input)
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/policies", body)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	bindPolicyCreateFlags(cmd, &input)
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newPolicyGetCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "get <policy-id>",
		Short: "Get a policy definition",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/policies/"+args[0], nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newPolicyUpdateCommand() *cobra.Command {
	var name string
	var description string
	var policyType string
	var mode string
	var projectID string
	var environmentID string
	var criticalDeny int
	var highWarn int
	var requireDigest bool
	var approvalOnCritical bool
	var enabled bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   "update <policy-id>",
		Short: "Update a policy definition",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bodyMap := map[string]any{}
			if cmd.Flags().Changed("name") {
				bodyMap["name"] = name
			}
			if cmd.Flags().Changed("description") {
				bodyMap["description"] = description
			}
			if cmd.Flags().Changed("type") {
				bodyMap["type"] = policyType
			}
			if cmd.Flags().Changed("mode") {
				bodyMap["mode"] = mode
			}
			if cmd.Flags().Changed("project-id") {
				bodyMap["projectId"] = projectID
			}
			if cmd.Flags().Changed("environment-id") {
				bodyMap["environmentId"] = environmentID
			}
			if cmd.Flags().Changed("critical-deny-threshold") {
				bodyMap["criticalDenyThreshold"] = criticalDeny
			}
			if cmd.Flags().Changed("high-warn-threshold") {
				bodyMap["highWarnThreshold"] = highWarn
			}
			if cmd.Flags().Changed("require-digest") {
				bodyMap["requireDigest"] = requireDigest
			}
			if cmd.Flags().Changed("approval-on-critical") {
				bodyMap["approvalOnCritical"] = approvalOnCritical
			}
			if cmd.Flags().Changed("enabled") {
				bodyMap["enabled"] = enabled
			}
			if len(bodyMap) == 0 {
				return fmt.Errorf("at least one update flag is required")
			}
			body, err := json.Marshal(bodyMap)
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPatch, serverURL, "/api/v1/policies/"+args[0], body)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "policy name")
	cmd.Flags().StringVar(&description, "description", "", "policy description")
	cmd.Flags().StringVar(&policyType, "type", "", "policy type")
	cmd.Flags().StringVar(&mode, "mode", "", "policy mode")
	cmd.Flags().StringVar(&projectID, "project-id", "", "project scope id")
	cmd.Flags().StringVar(&environmentID, "environment-id", "", "environment scope id")
	cmd.Flags().IntVar(&criticalDeny, "critical-deny-threshold", 0, "critical finding count that denies")
	cmd.Flags().IntVar(&highWarn, "high-warn-threshold", 0, "high finding count that warns")
	cmd.Flags().BoolVar(&requireDigest, "require-digest", false, "deny artifact references without sha256 digest")
	cmd.Flags().BoolVar(&approvalOnCritical, "approval-on-critical", false, "require approval on critical findings")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "enable or disable the policy")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newPolicyDisableCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "disable <policy-id>",
		Short: "Disable a policy definition without deleting it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodDelete, serverURL, "/api/v1/policies/"+args[0], nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func bindPolicyCreateFlags(cmd *cobra.Command, input *policyusecase.CreateInput) {
	cmd.Flags().StringVar(&input.ID, "id", "", "optional policy id")
	cmd.Flags().StringVar(&input.ProjectID, "project-id", "", "project scope id")
	cmd.Flags().StringVar(&input.EnvironmentID, "environment-id", "", "environment scope id")
	cmd.Flags().StringVar(&input.Name, "name", "", "policy name")
	cmd.Flags().StringVar(&input.Description, "description", "", "policy description")
	cmd.Flags().StringVar(&input.Type, "type", "security", "policy type")
	cmd.Flags().StringVar(&input.Mode, "mode", "warn", "policy mode")
	cmd.Flags().IntVar(&input.CriticalDeny, "critical-deny-threshold", 0, "critical finding count that denies")
	cmd.Flags().IntVar(&input.HighWarn, "high-warn-threshold", 1, "high finding count that warns")
	cmd.Flags().BoolVar(&input.RequireDigest, "require-digest", false, "deny artifact references without sha256 digest")
	cmd.Flags().BoolVar(&input.ApprovalOnCritical, "approval-on-critical", false, "require approval on critical findings")
}

func newArtifactCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "artifact", Short: "Artifact utilities"}
	cmd.AddCommand(newArtifactInspectCommand())
	cmd.AddCommand(newArtifactResolveCommand())
	cmd.AddCommand(newArtifactRegistryCommand())
	return cmd
}

func newArtifactInspectCommand() *cobra.Command {
	var artifactType string
	cmd := &cobra.Command{
		Use:   "inspect <reference>",
		Short: "Inspect and normalize an artifact reference locally",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := server.NewArtifactService().Inspect(cmd.Context(), args[0], domainartifact.ArtifactType(artifactType))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), result)
			return nil
		},
	}
	cmd.Flags().StringVar(&artifactType, "type", "image", "artifact type")
	return cmd
}

func newArtifactResolveCommand() *cobra.Command {
	var artifactType string
	var registryEndpoint string
	var insecure bool
	var usernameEnv string
	var passwordEnv string
	var tokenEnv string
	cmd := &cobra.Command{
		Use:   "resolve <reference>",
		Short: "Resolve artifact digest through generic OCI registry APIs when configured",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service := server.NewArtifactService()
			username := envValue(usernameEnv)
			password := envValue(passwordEnv)
			token := envValue(tokenEnv)
			if registryEndpoint != "" || insecure || username != "" || password != "" || token != "" {
				service = artifactusecase.NewService(
					artifactusecase.NewMemoryStore(),
					ociartifact.New(ociartifact.WithConfig(ociartifact.Config{Endpoint: registryEndpoint, Insecure: insecure, Username: username, Password: password, Token: token})),
					memory.New(),
				)
			}
			result, err := service.Resolve(cmd.Context(), args[0], domainartifact.ArtifactType(artifactType))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), result)
			return nil
		},
	}
	cmd.Flags().StringVar(&artifactType, "type", "image", "artifact type")
	cmd.Flags().StringVar(&registryEndpoint, "registry", "", "optional OCI registry endpoint override")
	cmd.Flags().BoolVar(&insecure, "insecure", false, "allow HTTP OCI registry endpoint for local development")
	cmd.Flags().StringVar(&usernameEnv, "username-env", "", "environment variable containing registry username")
	cmd.Flags().StringVar(&passwordEnv, "password-env", "", "environment variable containing registry password")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "", "environment variable containing registry bearer token")
	return cmd
}

func newArtifactRegistryCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "registry", Aliases: []string{"registries"}, Short: "Artifact registry metadata utilities"}
	cmd.AddCommand(newArtifactRegistryListCommand())
	cmd.AddCommand(newArtifactRegistryCreateCommand())
	cmd.AddCommand(newArtifactRegistryGetCommand())
	cmd.AddCommand(newArtifactRegistryUpdateCommand())
	cmd.AddCommand(newArtifactRegistryDisableCommand())
	cmd.AddCommand(newArtifactRegistryValidateCommand())
	return cmd
}

func newArtifactRegistryListCommand() *cobra.Command {
	var projectID string
	var serverURL string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List artifact registry definitions",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "/api/v1/artifact-registries"
			if projectID != "" {
				path += "?projectId=" + url.QueryEscape(projectID)
			}
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, path, nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&projectID, "project-id", "", "filter registries by project id")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newArtifactRegistryCreateCommand() *cobra.Command {
	var input artifactusecase.RegistryCreateInput
	var capabilities []string
	var serverURL string
	cmd := &cobra.Command{
		Use:   "create --name <name> --endpoint <endpoint>",
		Short: "Create artifact registry metadata without secret values",
		RunE: func(cmd *cobra.Command, args []string) error {
			if input.Name == "" {
				return fmt.Errorf("--name is required")
			}
			if input.Endpoint == "" && input.URL == "" {
				return fmt.Errorf("--endpoint is required")
			}
			input.Capabilities = capabilities
			body, err := json.Marshal(input)
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/artifact-registries", body)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	bindArtifactRegistryCreateFlags(cmd, &input, &capabilities)
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newArtifactRegistryGetCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "get <registry-id>",
		Short: "Get artifact registry metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/artifact-registries/"+args[0], nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newArtifactRegistryUpdateCommand() *cobra.Command {
	var name string
	var registryType string
	var endpoint string
	var projectID string
	var credentialRef string
	var insecure bool
	var enabled bool
	var capabilities []string
	var serverURL string
	cmd := &cobra.Command{
		Use:   "update <registry-id>",
		Short: "Update artifact registry metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bodyMap := map[string]any{}
			if cmd.Flags().Changed("name") {
				bodyMap["name"] = name
			}
			if cmd.Flags().Changed("type") {
				bodyMap["type"] = registryType
			}
			if cmd.Flags().Changed("endpoint") {
				bodyMap["endpoint"] = endpoint
			}
			if cmd.Flags().Changed("project-id") {
				bodyMap["projectId"] = projectID
			}
			if cmd.Flags().Changed("credential-ref") {
				bodyMap["credentialRef"] = credentialRef
			}
			if cmd.Flags().Changed("insecure") {
				bodyMap["insecure"] = insecure
			}
			if cmd.Flags().Changed("enabled") {
				bodyMap["enabled"] = enabled
			}
			if cmd.Flags().Changed("capability") {
				bodyMap["capabilities"] = capabilities
			}
			if len(bodyMap) == 0 {
				return fmt.Errorf("at least one update flag is required")
			}
			body, err := json.Marshal(bodyMap)
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPatch, serverURL, "/api/v1/artifact-registries/"+args[0], body)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "registry name")
	cmd.Flags().StringVar(&registryType, "type", "", "registry type, currently oci")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "registry endpoint host[:port] or URL")
	cmd.Flags().StringVar(&projectID, "project-id", "", "project scope id")
	cmd.Flags().StringVar(&credentialRef, "credential-ref", "", "CredentialRef id for registry access")
	cmd.Flags().BoolVar(&insecure, "insecure", false, "allow HTTP registry endpoint for local development")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "enable or disable the registry")
	cmd.Flags().StringSliceVar(&capabilities, "capability", nil, "registry capability, repeatable")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newArtifactRegistryDisableCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "disable <registry-id>",
		Short: "Disable artifact registry metadata without deleting it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodDelete, serverURL, "/api/v1/artifact-registries/"+args[0], nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newArtifactRegistryValidateCommand() *cobra.Command {
	var input artifactusecase.RegistryCreateInput
	var serverURL string
	cmd := &cobra.Command{
		Use:   "validate --name <name> --endpoint <endpoint>",
		Short: "Validate artifact registry configuration shape",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := json.Marshal(map[string]any{
				"name":     input.Name,
				"type":     input.Type,
				"endpoint": input.Endpoint,
				"insecure": input.Insecure,
			})
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/artifact-registries/validate", body)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&input.Name, "name", "", "registry name")
	cmd.Flags().StringVar(&input.Type, "type", "oci", "registry type, currently oci")
	cmd.Flags().StringVar(&input.Endpoint, "endpoint", "", "registry endpoint")
	cmd.Flags().BoolVar(&input.Insecure, "insecure", false, "allow HTTP registry endpoint for local development")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func bindArtifactRegistryCreateFlags(cmd *cobra.Command, input *artifactusecase.RegistryCreateInput, capabilities *[]string) {
	cmd.Flags().StringVar(&input.ID, "id", "", "optional registry id")
	cmd.Flags().StringVar(&input.ProjectID, "project-id", "", "project scope id")
	cmd.Flags().StringVar(&input.Name, "name", "", "registry name")
	cmd.Flags().StringVar(&input.Type, "type", "oci", "registry type, currently oci")
	cmd.Flags().StringVar(&input.Endpoint, "endpoint", "", "registry endpoint host[:port] or URL")
	cmd.Flags().BoolVar(&input.Insecure, "insecure", false, "allow HTTP registry endpoint for local development")
	cmd.Flags().StringVar(&input.CredentialRef, "credential-ref", "", "CredentialRef id for registry access")
	cmd.Flags().StringSliceVar(capabilities, "capability", nil, "registry capability, repeatable")
}

func newReleaseCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "release", Short: "Release utilities"}
	cmd.AddCommand(newReleaseCreateCommand())
	cmd.AddCommand(newReleaseGetCommand())
	cmd.AddCommand(newReleaseArtifactsCommand())
	cmd.AddCommand(newReleasePlanCommand())
	cmd.AddCommand(newReleaseDeployCommand())
	cmd.AddCommand(newReleaseSecurityCommand())
	cmd.AddCommand(newReleaseExecutionCommand())
	return cmd
}

func newReleaseCreateCommand() *cobra.Command {
	var file string
	var local bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   "create --file <release.yaml>",
		Short: "Create a release and bind artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			def, err := artifactusecase.LoadReleaseDefinitionFile(file)
			if err != nil {
				return err
			}
			if !local {
				body, err := json.Marshal(def)
				if err != nil {
					return err
				}
				payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/releases", body)
				if err != nil {
					return err
				}
				printJSON(cmd.OutOrStdout(), payload)
				return nil
			}
			record, err := server.NewArtifactService().CreateRelease(cmd.Context(), artifactusecase.CreateReleaseInput{Definition: def})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Release: %s\n", record.Release.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Version: %s\n", record.Release.Version)
			fmt.Fprintf(cmd.OutOrStdout(), "Artifacts: %d\n", len(record.Bindings))
			if len(record.Warnings) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Warnings: %d\n", len(record.Warnings))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "release definition file")
	cmd.Flags().BoolVar(&local, "local", true, "create release in the in-process local runtime")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL for --local=false")
	return cmd
}

func newReleaseGetCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "get <release-id>",
		Short: "Get a release from a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/releases/"+args[0], nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newReleaseArtifactsCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "artifacts <release-id>",
		Short: "List release artifacts from a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/releases/"+args[0]+"/artifacts", nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newReleaseSecurityCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "security <release-id>",
		Short: "Get release security gate output from a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/releases/"+args[0]+"/security", nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newReleasePlanCommand() *cobra.Command {
	var file string
	var local bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   "plan --file <release-orchestration.yaml>",
		Short: "Create a multi-target ReleasePlan",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			def, err := releaseorchestration.LoadDefinitionFile(file)
			if err != nil {
				return err
			}
			if !local {
				body, err := json.Marshal(def)
				if err != nil {
					return err
				}
				path := "/api/v1/releases/" + def.Spec.ReleaseID + "/plan"
				if def.Spec.ReleaseID == "" {
					path = "/api/v1/releases/local/plan"
				}
				payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, path, body)
				if err != nil {
					return err
				}
				printJSON(cmd.OutOrStdout(), payload)
				return nil
			}
			record, err := server.NewReleaseOrchestrationService().Plan(cmd.Context(), releaseorchestration.PlanInput{Definition: def})
			if err != nil {
				return err
			}
			printReleasePlanSummary(cmd.OutOrStdout(), record.Plan)
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "release orchestration definition file")
	cmd.Flags().BoolVar(&local, "local", true, "plan with the in-process Phase 2.7 local runtime")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL for --local=false")
	return cmd
}

func newReleaseDeployCommand() *cobra.Command {
	var file string
	var local bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   "deploy --file <release-orchestration.yaml> --local",
		Short: "Execute a multi-target release locally or against a Nivora server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			def, err := releaseorchestration.LoadDefinitionFile(file)
			if err != nil {
				return err
			}
			if !local {
				body, err := json.Marshal(def)
				if err != nil {
					return err
				}
				path := "/api/v1/releases/" + def.Spec.ReleaseID + "/deploy"
				if def.Spec.ReleaseID == "" {
					path = "/api/v1/releases/local/deploy"
				}
				payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, path, body)
				if err != nil {
					return err
				}
				printJSON(cmd.OutOrStdout(), payload)
				return nil
			}
			started := time.Now()
			record, err := server.NewReleaseOrchestrationService().Deploy(cmd.Context(), releaseorchestration.DeployInput{Definition: def})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ReleaseExecution: %s\n", record.Execution.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", record.Execution.Status)
			fmt.Fprintf(cmd.OutOrStdout(), "Targets: %d\n", len(record.Execution.Targets))
			fmt.Fprintf(cmd.OutOrStdout(), "DeploymentRuns: %d\n", len(record.Execution.DeploymentRunIDs))
			fmt.Fprintf(cmd.OutOrStdout(), "Duration: %s\n", time.Since(started).Round(time.Millisecond))
			if record.Execution.Reason != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Reason: %s\n", record.Execution.Reason)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "release orchestration definition file")
	cmd.Flags().BoolVar(&local, "local", true, "deploy with the in-process Phase 2.7 local runtime")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL for --local=false")
	return cmd
}

func newReleaseExecutionCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "execution", Short: "ReleaseExecution utilities"}
	cmd.AddCommand(newReleaseExecutionInspectCommand("get", "Get a ReleaseExecution", ""))
	cmd.AddCommand(newReleaseExecutionInspectCommand("timeline", "Get ReleaseExecution timeline", "/timeline"))
	cmd.AddCommand(newReleaseExecutionInspectCommand("targets", "Get ReleaseExecution targets", "/targets"))
	cmd.AddCommand(newReleaseExecutionCancelCommand())
	cmd.AddCommand(newReleaseExecutionResumeCommand())
	return cmd
}

func newReleaseExecutionInspectCommand(name string, short string, suffix string) *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   name + " <execution-id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/releases/executions/"+args[0]+suffix, nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newReleaseExecutionCancelCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "cancel <execution-id>",
		Short: "Cancel a ReleaseExecution on a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/releases/executions/"+args[0]+"/cancel", nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newReleaseExecutionResumeCommand() *cobra.Command {
	var serverURL string
	var status string
	cmd := &cobra.Command{
		Use:   "resume <execution-id> --approval-status Approved",
		Short: "Resume or stop a ReleaseExecution using an approval decision",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := json.Marshal(map[string]string{"subjectType": "release", "subjectId": args[0], "status": status})
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/releases/executions/"+args[0]+"/resume", body)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&status, "approval-status", "Approved", "approval status: Approved, Rejected, Expired, or Canceled")
	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			info := version.Current()
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s (commit %s, date %s)\n", info.Name, info.Version, info.Commit, info.Date)
		},
	}
}

func newRunCommand(name string, defaultConfig string, run func(context.Context, string) error) *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   name,
		Short: "Run nivora-" + name,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return run(ctx, configPath)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", defaultConfig, "config file path")
	return cmd
}

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Config utilities",
	}

	var file string
	validate := &cobra.Command{
		Use:   "validate",
		Short: "Validate a config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := config.Load(file); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "config %s is valid\n", file)
			return nil
		},
	}
	validate.Flags().StringVar(&file, "file", "configs/server.yaml", "config file to validate")
	cmd.AddCommand(validate)
	return cmd
}

func newPipelineCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Pipeline utilities",
	}
	definitions := &cobra.Command{Use: "definition", Aliases: []string{"definitions"}, Short: "Pipeline definition catalog utilities"}
	definitions.AddCommand(newPipelineDefinitionListCommand())
	definitions.AddCommand(newPipelineDefinitionCreateCommand())
	definitions.AddCommand(newPipelineDefinitionGetCommand())
	definitions.AddCommand(newPipelineDefinitionUpdateCommand())
	definitions.AddCommand(newPipelineDefinitionDisableCommand())
	cmd.AddCommand(definitions)
	cmd.AddCommand(newPipelineRunCommand())
	cmd.AddCommand(newPipelineGetCommand())
	cmd.AddCommand(newPipelineInspectCommand("logs", "Get PipelineRun logs", "/logs"))
	cmd.AddCommand(newPipelineInspectCommand("events", "Get PipelineRun events", "/events"))
	cmd.AddCommand(newPipelineInspectCommand("timeline", "Get PipelineRun timeline", "/timeline"))
	cmd.AddCommand(newPipelineCancelCommand())
	return cmd
}

func newPipelineDefinitionListCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	var projectID string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pipeline definitions",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if projectID != "" {
				query = "?" + url.Values{"projectId": []string{projectID}}.Encode()
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodGet, serverURL, "/api/v1/pipelines"+query, nil, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	cmd.Flags().StringVar(&projectID, "project-id", "", "project id filter")
	return cmd
}

func newPipelineDefinitionCreateCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	var file string
	var projectID string
	var description string
	cmd := &cobra.Command{
		Use:   "create --file <pipeline.yaml>",
		Short: "Create a pipeline definition catalog record",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			def, err := pipelineusecase.LoadDefinitionFile(file)
			if err != nil {
				return err
			}
			body, err := json.Marshal(map[string]any{
				"projectId":   projectID,
				"description": description,
				"definition":  def,
			})
			if err != nil {
				return err
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodPost, serverURL, "/api/v1/pipelines", body, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	cmd.Flags().StringVar(&file, "file", "", "pipeline definition YAML file")
	cmd.Flags().StringVar(&projectID, "project-id", "", "project id")
	cmd.Flags().StringVar(&description, "description", "", "pipeline description")
	return cmd
}

func newPipelineDefinitionGetCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	cmd := &cobra.Command{
		Use:   "get <pipeline-id>",
		Short: "Get a pipeline definition",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSONWithToken(cmd.Context(), http.MethodGet, serverURL, "/api/v1/pipelines/"+args[0], nil, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	return cmd
}

func newPipelineDefinitionUpdateCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	var file string
	var description string
	var enabled bool
	cmd := &cobra.Command{
		Use:   "update <pipeline-id>",
		Short: "Update a pipeline definition",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bodyMap := map[string]any{}
			if cmd.Flags().Changed("description") {
				bodyMap["description"] = description
			}
			if cmd.Flags().Changed("enabled") {
				bodyMap["enabled"] = enabled
			}
			if file != "" {
				def, err := pipelineusecase.LoadDefinitionFile(file)
				if err != nil {
					return err
				}
				bodyMap["definition"] = def
			}
			if len(bodyMap) == 0 {
				return fmt.Errorf("at least one of --file, --description, or --enabled must be set")
			}
			body, err := json.Marshal(bodyMap)
			if err != nil {
				return err
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodPatch, serverURL, "/api/v1/pipelines/"+args[0], body, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	cmd.Flags().StringVar(&file, "file", "", "replacement pipeline definition YAML file")
	cmd.Flags().StringVar(&description, "description", "", "pipeline description")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "pipeline definition enabled state")
	return cmd
}

func newPipelineDefinitionDisableCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	cmd := &cobra.Command{
		Use:   "disable <pipeline-id>",
		Short: "Disable a pipeline definition",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSONWithToken(cmd.Context(), http.MethodDelete, serverURL, "/api/v1/pipelines/"+args[0], nil, os.Getenv(tokenEnv))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	return cmd
}

func newPipelineRunCommand() *cobra.Command {
	var local bool
	var printLogs bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   "run --local <pipeline.yaml>",
		Short: "Run a pipeline definition locally or against a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			def, err := pipelineusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			if !local {
				if serverURL == "" {
					return fmt.Errorf("--server is required when --local=false")
				}
				body, err := json.Marshal(def)
				if err != nil {
					return err
				}
				payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/pipeline-runs", body)
				if err != nil {
					return err
				}
				printPipelineRunSummary(cmd.OutOrStdout(), payload)
				if printLogs {
					printLogSummary(cmd.OutOrStdout(), payload)
				}
				return nil
			}
			started := time.Now()
			result, err := server.NewPipelineService().CreateAndRun(cmd.Context(), pipelineusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "PipelineRun: %s\n", result.Record.Run.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", result.Record.Run.Status)
			fmt.Fprintf(cmd.OutOrStdout(), "Duration: %s\n", time.Since(started).Round(time.Millisecond))
			fmt.Fprintf(cmd.OutOrStdout(), "Logs: %d chunk(s)\n", len(result.Record.Logs))
			if result.Record.Run.FailureReason != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Failure: %s\n", result.Record.Run.FailureReason)
			}
			if printLogs {
				for _, log := range result.Record.Logs {
					fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s", log.Stream, log.Content)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&local, "local", true, "run with the in-process Phase 1 local runtime")
	cmd.Flags().BoolVar(&printLogs, "logs", true, "print captured logs")
	cmd.Flags().StringVar(&serverURL, "server", "", "Nivora server URL for --local=false")
	return cmd
}

func newPipelineGetCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "get <pipeline-run-id>",
		Short: "Get a PipelineRun from a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/pipeline-runs/"+args[0], nil)
			if err != nil {
				return err
			}
			printPipelineRunSummary(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newPipelineInspectCommand(name string, short string, suffix string) *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   name + " <pipeline-run-id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/pipeline-runs/"+args[0]+suffix, nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newPipelineCancelCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "cancel <pipeline-run-id>",
		Short: "Cancel a PipelineRun on a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/pipeline-runs/"+args[0]+"/cancel", nil)
			if err != nil {
				return err
			}
			printPipelineRunSummary(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newDeploymentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployment",
		Short: "DeploymentRun utilities",
	}
	cmd.AddCommand(newDeploymentPlanCommand())
	cmd.AddCommand(newDeploymentRunCommand())
	cmd.AddCommand(newDeploymentDryRunCommand())
	cmd.AddCommand(newDeploymentApplyCommand())
	cmd.AddCommand(newDeploymentRollbackCommand())
	cmd.AddCommand(newDeploymentHostCommand())
	cmd.AddCommand(newDeploymentGetCommand())
	cmd.AddCommand(newDeploymentLocalInspectCommand("health", "Get DeploymentRun health", "/health", func(record deploymentusecase.RunRecord) any { return record.Health }))
	cmd.AddCommand(newDeploymentLocalInspectCommand("diff", "Get DeploymentRun diff", "/diff", func(record deploymentusecase.RunRecord) any { return record.Diff }))
	cmd.AddCommand(newDeploymentLocalInspectCommand("snapshot", "Get DeploymentRun manifest snapshot", "/manifest-snapshot", func(record deploymentusecase.RunRecord) any { return record.Snapshot }))
	cmd.AddCommand(newDeploymentLocalInspectCommand("rollback-plan", "Get DeploymentRun rollback plan", "/rollback-plan", func(record deploymentusecase.RunRecord) any { return record.RollbackPlan }))
	cmd.AddCommand(newDeploymentInspectCommand("argocd-status", "Get DeploymentRun Argo CD status", "/argocd-status"))
	cmd.AddCommand(newDeploymentSyncCommand())
	cmd.AddCommand(newDeploymentInspectCommand("resources", "Get DeploymentRun resources", "/resources"))
	cmd.AddCommand(newDeploymentInspectCommand("logs", "Get DeploymentRun logs", "/logs"))
	cmd.AddCommand(newDeploymentInspectCommand("events", "Get DeploymentRun events", "/events"))
	cmd.AddCommand(newDeploymentInspectCommand("timeline", "Get DeploymentRun timeline", "/timeline"))
	cmd.AddCommand(newDeploymentInspectCommand("security", "Get DeploymentRun security gate output", "/security"))
	cmd.AddCommand(newDeploymentCancelCommand())
	cmd.AddCommand(newDeploymentResumeCommand())
	return cmd
}

func newHostGroupsCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{Use: "host-groups", Short: "Host group utilities"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List host groups from a Nivora server",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/host-groups", nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	})
	cmd.PersistentFlags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newDeploymentHostCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "host", Short: "Host deployment runtime utilities"}
	cmd.AddCommand(newDeploymentHostPlanCommand())
	cmd.AddCommand(newDeploymentHostRunCommand())
	return cmd
}

func newDeploymentHostPlanCommand() *cobra.Command {
	var local bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   "plan --file <deployment.yaml>",
		Short: "Build a safe host deployment plan",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := cmd.Flags().GetString("file")
			if err != nil {
				return err
			}
			if path == "" {
				return fmt.Errorf("--file is required")
			}
			def, err := deploymentusecase.LoadDefinitionFile(path)
			if err != nil {
				return err
			}
			if !local {
				body, err := json.Marshal(def)
				if err != nil {
					return err
				}
				payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/deployments/host/plan", body)
				if err != nil {
					return err
				}
				printJSON(cmd.OutOrStdout(), payload)
				return nil
			}
			result, err := server.NewDeploymentService().Plan(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), result.Record.HostPlan)
			return nil
		},
	}
	cmd.Flags().String("file", "", "host deployment definition file")
	cmd.Flags().BoolVar(&local, "local", true, "plan with the in-process local host runtime")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL for --local=false")
	return cmd
}

func newDeploymentHostRunCommand() *cobra.Command {
	var local bool
	var confirm bool
	var allowRemote bool
	cmd := &cobra.Command{
		Use:   "run --file <deployment.yaml> --local",
		Short: "Run a host DeploymentRun through the safe local/noop runtime",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !local {
				return fmt.Errorf("server-backed host run is not implemented in the CLI; use --local")
			}
			path, err := cmd.Flags().GetString("file")
			if err != nil {
				return err
			}
			if path == "" {
				return fmt.Errorf("--file is required")
			}
			def, err := deploymentusecase.LoadDefinitionFile(path)
			if err != nil {
				return err
			}
			if allowRemote {
				def.Spec.Host.AllowRemoteHostDeploy = true
			}
			started := time.Now()
			result, err := server.NewDeploymentService().CreateAndRun(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def, AllowApply: confirm, Confirm: confirm})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "DeploymentRun: %s\n", result.Record.Run.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", result.Record.Run.Status)
			fmt.Fprintf(cmd.OutOrStdout(), "Duration: %s\n", time.Since(started).Round(time.Millisecond))
			fmt.Fprintf(cmd.OutOrStdout(), "Hosts: %d\n", len(result.Record.HostDetails))
			fmt.Fprintf(cmd.OutOrStdout(), "RollbackPlan: %s\n", result.Record.RollbackPlan.Strategy)
			return nil
		},
	}
	cmd.Flags().String("file", "", "host deployment definition file")
	cmd.Flags().BoolVar(&local, "local", true, "run with the in-process local host runtime")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "confirm explicit host apply")
	cmd.Flags().BoolVar(&allowRemote, "allow-remote-host-deploy", false, "allow guarded remote host deployment when the spec also opts in")
	return cmd
}

func newDeploymentLocalInspectCommand(name string, short string, suffix string, selector func(deploymentusecase.RunRecord) any) *cobra.Command {
	var local bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   name + " [--local <deployment.yaml> | <deployment-run-id>]",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if local {
				def, err := deploymentusecase.LoadDefinitionFile(args[0])
				if err != nil {
					return err
				}
				result, err := server.NewDeploymentService().Plan(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def})
				if err != nil {
					return err
				}
				printJSON(cmd.OutOrStdout(), selector(result.Record))
				return nil
			}
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/deployments/"+args[0]+suffix, nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().BoolVar(&local, "local", false, "evaluate a local deployment definition instead of querying a server")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newDeploymentPlanCommand() *cobra.Command {
	var local bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   "plan --local <deployment.yaml>",
		Short: "Render and plan a YAML DeploymentRun locally or against a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			def, err := deploymentusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			if !local {
				if serverURL == "" {
					return fmt.Errorf("--server is required when --local=false")
				}
				body, err := json.Marshal(def)
				if err != nil {
					return err
				}
				payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/deployments/plan", body)
				if err != nil {
					return err
				}
				printJSON(cmd.OutOrStdout(), payload)
				return nil
			}
			result, err := server.NewDeploymentService().Plan(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			printDeploymentPlanSummary(cmd.OutOrStdout(), result.Record.Plan)
			return nil
		},
	}
	cmd.Flags().BoolVar(&local, "local", true, "plan with the in-process Phase 2.2 local runtime")
	cmd.Flags().StringVar(&serverURL, "server", "", "Nivora server URL for --local=false")
	return cmd
}

func newDeploymentRunCommand() *cobra.Command {
	var local bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   "run --local <deployment.yaml>",
		Short: "Run a non-destructive YAML DeploymentRun locally or against a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			def, err := deploymentusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			if !local {
				if serverURL == "" {
					return fmt.Errorf("--server is required when --local=false")
				}
				body, err := json.Marshal(def)
				if err != nil {
					return err
				}
				payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/deployments", body)
				if err != nil {
					return err
				}
				printDeploymentRunSummary(cmd.OutOrStdout(), payload)
				return nil
			}
			started := time.Now()
			result, err := server.NewDeploymentService().CreateAndRun(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "DeploymentRun: %s\n", result.Record.Run.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", result.Record.Run.Status)
			fmt.Fprintf(cmd.OutOrStdout(), "Duration: %s\n", time.Since(started).Round(time.Millisecond))
			fmt.Fprintf(cmd.OutOrStdout(), "Manifests: %d\n", result.Record.Plan.ManifestCount)
			fmt.Fprintf(cmd.OutOrStdout(), "Logs: %d chunk(s)\n", len(result.Record.Logs))
			if result.Record.Run.Reason != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Reason: %s\n", result.Record.Run.Reason)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&local, "local", true, "run with the in-process Phase 2.2 local runtime")
	cmd.Flags().StringVar(&serverURL, "server", "", "Nivora server URL for --local=false")
	return cmd
}

func newDeploymentDryRunCommand() *cobra.Command {
	cmd := newDeploymentRunCommand()
	cmd.Use = "dry-run --local <deployment.yaml>"
	cmd.Short = "Run a non-destructive YAML DeploymentRun dry-run"
	return cmd
}

func newDeploymentApplyCommand() *cobra.Command {
	var local bool
	var confirm bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   "apply --local <deployment.yaml> --confirm",
		Short: "Run an explicit local YAML apply through the configured manifest client",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("deployment apply requires --confirm")
			}
			def, err := deploymentusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			def.Spec.Options.Apply = true
			def.Spec.Options.DryRun = false
			if !local {
				body, err := json.Marshal(map[string]any{"definition": def, "confirm": true})
				if err != nil {
					return err
				}
				payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/deployments/apply", body)
				if err != nil {
					return err
				}
				printDeploymentRunSummary(cmd.OutOrStdout(), payload)
				return nil
			}
			started := time.Now()
			result, err := server.NewDeploymentService().CreateAndRun(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def, AllowApply: true, Confirm: true})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "DeploymentRun: %s\n", result.Record.Run.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", result.Record.Run.Status)
			fmt.Fprintf(cmd.OutOrStdout(), "Duration: %s\n", time.Since(started).Round(time.Millisecond))
			fmt.Fprintf(cmd.OutOrStdout(), "Apply: %s\n", result.Record.Apply.Message)
			if result.Record.Rollout.Message != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Rollout: %s\n", result.Record.Rollout.Message)
			}
			if result.Record.Run.Reason != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Reason: %s\n", result.Record.Run.Reason)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&local, "local", true, "apply with the in-process Phase 2.2 local runtime")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "confirm explicit local apply")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL for --local=false")
	return cmd
}

func newDeploymentRollbackCommand() *cobra.Command {
	var confirm bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   "rollback <deployment-run-id> --confirm",
		Short: "Run a guarded manifest-restore rollback for a DeploymentRun",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("deployment rollback requires --confirm")
			}
			body, err := json.Marshal(map[string]any{"confirm": true})
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/deployments/"+args[0]+"/rollback", body)
			if err != nil {
				return err
			}
			printDeploymentRunSummary(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().BoolVar(&confirm, "confirm", false, "confirm guarded rollback")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newDeploymentGetCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "get <deployment-run-id>",
		Short: "Get a DeploymentRun from a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/deployments/"+args[0], nil)
			if err != nil {
				return err
			}
			printDeploymentRunSummary(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newDeploymentInspectCommand(name string, short string, suffix string) *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   name + " <deployment-run-id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/deployments/"+args[0]+suffix, nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newDeploymentCancelCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "cancel <deployment-run-id>",
		Short: "Cancel a DeploymentRun on a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/deployments/"+args[0]+"/cancel", nil)
			if err != nil {
				return err
			}
			printDeploymentRunSummary(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newDeploymentResumeCommand() *cobra.Command {
	var serverURL string
	var status string
	cmd := &cobra.Command{
		Use:   "resume <deployment-run-id> --approval-status Approved",
		Short: "Resume or stop a DeploymentRun using an approval decision",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := json.Marshal(map[string]string{"subjectType": "deployment", "subjectId": args[0], "status": status})
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/deployments/"+args[0]+"/resume", body)
			if err != nil {
				return err
			}
			printDeploymentRunSummary(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&status, "approval-status", "Approved", "approval status: Approved, Rejected, Expired, or Canceled")
	return cmd
}

func newDeploymentSyncCommand() *cobra.Command {
	var serverURL string
	var confirm bool
	var allowSync bool
	cmd := &cobra.Command{
		Use:   "sync <deployment-run-id> --confirm --allow-sync",
		Short: "Request guarded Argo CD sync for a DeploymentRun",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm || !allowSync {
				return fmt.Errorf("deployment sync requires --confirm and --allow-sync")
			}
			body, err := json.Marshal(map[string]any{"allowSync": true, "confirmed": true})
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/deployments/"+args[0]+"/sync", body)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "confirm sync request")
	cmd.Flags().BoolVar(&allowSync, "allow-sync", false, "allow guarded sync request")
	return cmd
}

func newGitOpsCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "gitops", Short: "GitOps planning utilities"}
	cmd.AddCommand(newGitOpsPlanCommand())
	cmd.AddCommand(newGitOpsDiffCommand())
	cmd.AddCommand(newGitOpsWriteCommand())
	cmd.AddCommand(newGitOpsCommitCommand())
	cmd.AddCommand(newGitOpsRollbackCommand())
	cmd.AddCommand(newGitOpsDeployCommand())
	return cmd
}

func newGitOpsPlanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan --local <deployment.yaml>",
		Short: "Build a local GitOps change plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			def, err := deploymentusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			result, err := server.NewDeploymentService().Plan(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			printGitOpsPlanSummary(cmd.OutOrStdout(), result.Record.GitOpsPlan)
			return nil
		},
	}
	cmd.Flags().Bool("local", true, "use the in-process Phase 2.3 local runtime")
	return cmd
}

func newGitOpsDiffCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff --local <deployment.yaml>",
		Short: "Build a local GitOps diff plan without writing",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			def, err := deploymentusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			def.Spec.GitOps.WriteToWorkingTree = false
			result, err := server.NewDeploymentService().Plan(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			printGitOpsPlanSummary(cmd.OutOrStdout(), result.Record.GitOpsPlan)
			return nil
		},
	}
	cmd.Flags().Bool("local", true, "use the in-process Phase 2.3 local runtime")
	return cmd
}

func newGitOpsWriteCommand() *cobra.Command {
	var workingTree string
	var confirm bool
	cmd := &cobra.Command{
		Use:   "write --local <deployment.yaml> --working-tree <path> --confirm",
		Short: "Write GitOps changes to a local working tree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("gitops write requires --confirm")
			}
			if workingTree == "" {
				return fmt.Errorf("--working-tree is required")
			}
			def, err := deploymentusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			def.Spec.GitOps.WriteToWorkingTree = true
			def.Spec.GitOps.WorkingTree = workingTree
			result, err := server.NewDeploymentService().CreateAndRun(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "DeploymentRun: %s\n", result.Record.Run.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", result.Record.Run.Status)
			fmt.Fprintf(cmd.OutOrStdout(), "Diff: %s\n", result.Record.GitOpsDiff.Summary)
			return nil
		},
	}
	cmd.Flags().Bool("local", true, "use the in-process Phase 2.3 local runtime")
	cmd.Flags().StringVar(&workingTree, "working-tree", "", "local GitOps working tree root")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "confirm local working tree writes")
	return cmd
}

func newGitOpsCommitCommand() *cobra.Command {
	var workingTree string
	var confirm bool
	var push bool
	var allowPush bool
	var remote string
	var branch string
	var message string
	cmd := &cobra.Command{
		Use:   "commit --local <deployment.yaml> --working-tree <path> --confirm",
		Short: "Write and commit GitOps changes in a local working tree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("gitops commit requires --confirm")
			}
			if workingTree == "" {
				return fmt.Errorf("--working-tree is required")
			}
			if push && !allowPush {
				return fmt.Errorf("gitops push requires --allow-push")
			}
			def, err := deploymentusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			def.Spec.GitOps.WriteToWorkingTree = true
			def.Spec.GitOps.WorkingTree = workingTree
			def.Spec.GitOps.Commit = true
			def.Spec.GitOps.CommitMessage = message
			def.Spec.GitOps.Push = push
			def.Spec.GitOps.AllowPush = allowPush
			def.Spec.GitOps.Remote = remote
			def.Spec.GitOps.Branch = branch
			result, err := server.NewDeploymentService().CreateAndRun(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def, Confirm: true})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "DeploymentRun: %s\n", result.Record.Run.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", result.Record.Run.Status)
			fmt.Fprintf(cmd.OutOrStdout(), "Commit: %s\n", result.Record.GitOpsCommit.Revision)
			if result.Record.GitOpsPush.Pushed {
				fmt.Fprintf(cmd.OutOrStdout(), "Pushed: %s\n", result.Record.GitOpsPush.Revision)
			}
			return nil
		},
	}
	cmd.Flags().Bool("local", true, "use the in-process Phase 6.1 local runtime")
	cmd.Flags().StringVar(&workingTree, "working-tree", "", "local GitOps working tree root")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "confirm local working tree commit")
	cmd.Flags().BoolVar(&push, "push", false, "push commit after creating it; disabled by default")
	cmd.Flags().BoolVar(&allowPush, "allow-push", false, "allow guarded Git push")
	cmd.Flags().StringVar(&remote, "remote", "origin", "Git remote for guarded push")
	cmd.Flags().StringVar(&branch, "branch", "HEAD", "Git branch/ref for guarded push")
	cmd.Flags().StringVar(&message, "message", "", "override generated commit message")
	return cmd
}

func newGitOpsRollbackCommand() *cobra.Command {
	var workingTree string
	var revision string
	var confirm bool
	cmd := &cobra.Command{
		Use:   "rollback --local <deployment.yaml> --working-tree <path> --revision <rev> --confirm",
		Short: "Plan and execute a guarded local GitOps rollback by Git revision",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("gitops rollback requires --confirm")
			}
			if workingTree == "" || revision == "" {
				return fmt.Errorf("gitops rollback requires --working-tree and --revision")
			}
			def, err := deploymentusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			def.Spec.GitOps.WorkingTree = workingTree
			def.Spec.GitOps.Rollback = true
			def.Spec.GitOps.RollbackRevision = revision
			result, err := server.NewDeploymentService().CreateAndRun(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def, Confirm: true})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "DeploymentRun: %s\n", result.Record.Run.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", result.Record.Run.Status)
			fmt.Fprintf(cmd.OutOrStdout(), "RollbackRevision: %s\n", result.Record.GitOpsRollback.Revision)
			return nil
		},
	}
	cmd.Flags().Bool("local", true, "use the in-process Phase 6.1 local runtime")
	cmd.Flags().StringVar(&workingTree, "working-tree", "", "local GitOps working tree root")
	cmd.Flags().StringVar(&revision, "revision", "", "Git revision to check out")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "confirm guarded revision checkout")
	return cmd
}

func newGitOpsDeployCommand() *cobra.Command {
	var allowSync bool
	var confirm bool
	cmd := &cobra.Command{
		Use:   "deploy --local <deployment.yaml>",
		Short: "Run a local GitOps DeploymentRun with guarded Argo CD sync semantics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			def, err := deploymentusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			result, err := server.NewDeploymentService().CreateAndRun(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def, AllowSync: allowSync, Confirm: confirm})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "DeploymentRun: %s\n", result.Record.Run.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", result.Record.Run.Status)
			if result.Record.ArgoCDSync.Message != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "ArgoCDSync: %s\n", result.Record.ArgoCDSync.Message)
			}
			return nil
		},
	}
	cmd.Flags().Bool("local", true, "use the in-process Phase 2.6 local runtime")
	cmd.Flags().BoolVar(&allowSync, "allow-sync", false, "allow guarded sync request")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "confirm guarded sync request")
	return cmd
}

func newArgoCDCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "argocd", Short: "Argo CD foundation utilities"}
	cmd.AddCommand(newArgoCDStatusCommand())
	cmd.AddCommand(newArgoCDResourcesCommand())
	cmd.AddCommand(newArgoCDSyncCommand())
	return cmd
}

func newArgoCDStatusCommand() *cobra.Command {
	var app string
	cmd := &cobra.Command{
		Use:   "status --app <name>",
		Short: "Read modeled Argo CD application status through the local noop provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			if app == "" {
				return fmt.Errorf("--app is required")
			}
			def := gitOpsStatusDefinition(app)
			def.Spec.GitOps.StatusRead = true
			result, err := server.NewDeploymentService().CreateAndRun(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), result.Record.ArgoCD)
			return nil
		},
	}
	cmd.Flags().StringVar(&app, "app", "", "Argo CD application name")
	cmd.Flags().String("server", "", "optional Argo CD URL for future adapters; ignored by the Phase 2.6 noop provider")
	return cmd
}

func newArgoCDResourcesCommand() *cobra.Command {
	var app string
	cmd := &cobra.Command{
		Use:   "resources --app <name>",
		Short: "Read modeled Argo CD application resources through the local noop provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			if app == "" {
				return fmt.Errorf("--app is required")
			}
			def := gitOpsStatusDefinition(app)
			def.Spec.GitOps.StatusRead = true
			result, err := server.NewDeploymentService().CreateAndRun(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), result.Record.ArgoCDResources)
			return nil
		},
	}
	cmd.Flags().StringVar(&app, "app", "", "Argo CD application name")
	return cmd
}

func newArgoCDSyncCommand() *cobra.Command {
	var app string
	var confirm bool
	var allowSync bool
	cmd := &cobra.Command{
		Use:   "sync --app <name> --confirm --allow-sync",
		Short: "Request Argo CD sync through the guarded local noop provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			if app == "" {
				return fmt.Errorf("--app is required")
			}
			if !confirm || !allowSync {
				return fmt.Errorf("argocd sync requires --confirm and --allow-sync")
			}
			def := gitOpsStatusDefinition(app)
			def.Spec.GitOps.Sync = true
			def.Spec.GitOps.AllowSync = true
			def.Spec.GitOps.Wait = true
			result, err := server.NewDeploymentService().CreateAndRun(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def, AllowSync: true, Confirm: true})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "DeploymentRun: %s\n", result.Record.Run.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", result.Record.Run.Status)
			return nil
		},
	}
	cmd.Flags().StringVar(&app, "app", "", "Argo CD application name")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "confirm sync request")
	cmd.Flags().BoolVar(&allowSync, "allow-sync", false, "allow guarded sync request")
	return cmd
}

func gitOpsStatusDefinition(app string) deploymentusecase.Definition {
	return deploymentusecase.Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Deployment",
		Metadata:   deploymentusecase.Metadata{Name: app + "-argocd-status"},
		Spec: deploymentusecase.Spec{
			Application: "argocd-status",
			Environment: "local",
			Target: deploymentusecase.Target{
				Type:            "argocd",
				Name:            "local-noop",
				ApplicationName: app,
				RepoURL:         "placeholder://argocd-status",
				Path:            "apps/" + app,
				Revision:        "HEAD",
			},
			GitOps: deploymentusecase.GitOps{Mode: "plan"},
		},
	}
}

func newRunnerCommand() *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   "runner",
		Short: "Run nivora-runner or use runner utilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return runner.Run(ctx, configPath)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "configs/runner.yaml", "config file path")
	cmd.AddCommand(newRunnerListCommand())
	cmd.AddCommand(newRunnerRegisterCommand())
	cmd.AddCommand(newRunnerStatusCommand())
	cmd.AddCommand(newRunnerTokenCommand())
	cmd.AddCommand(newRunnerHeartbeatCommand())
	cmd.AddCommand(newRunnerClaimCommand())
	cmd.AddCommand(newRunnerOfflineDetectCommand())
	cmd.AddCommand(newRunnerAppendLogCommand())
	cmd.AddCommand(newRunnerUpdateStatusCommand())
	return cmd
}

func newRunnerListCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List runners from a Nivora server",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/runners", nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newRunnerRegisterCommand() *cobra.Command {
	var serverURL string
	var name string
	cmd := &cobra.Command{
		Use:   "register --name <runner-id>",
		Short: "Register a runner on a Nivora server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			body, err := json.Marshal(map[string]any{
				"id":        name,
				"name":      name,
				"status":    "online",
				"executors": []string{"shell"},
				"labels":    map[string]string{"runtime": "local"},
			})
			if err != nil {
				return err
			}
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/runners/register", body)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&name, "name", "local-runner", "runner ID")
	return cmd
}

func newRunnerStatusCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "status <runner-id>",
		Short: "Get runner status from a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/runners/"+args[0], nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newRunnerTokenCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "token", Short: "Runner token utilities"}
	cmd.AddCommand(newRunnerTokenRotateCommand())
	cmd.AddCommand(newRunnerTokenRevokeCommand())
	return cmd
}

func newRunnerTokenRotateCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "rotate <runner-id>",
		Short: "Rotate a runner token; the raw token is returned only once",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/runners/"+args[0]+"/token/rotate", nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newRunnerTokenRevokeCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "revoke <runner-id>",
		Short: "Revoke a runner token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/runners/"+args[0]+"/token/revoke", nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newRunnerHeartbeatCommand() *cobra.Command {
	var serverURL string
	var name string
	var tokenEnv string
	cmd := &cobra.Command{
		Use:   "heartbeat --name <runner-id>",
		Short: "Record a runner heartbeat on a Nivora server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			token, err := runnerTokenFromEnv(tokenEnv)
			if err != nil {
				return err
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodPost, serverURL, "/api/v1/runners/"+name+"/heartbeat", nil, token)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&name, "name", "local-runner", "runner ID")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_RUNNER_TOKEN", "environment variable containing the runner token")
	return cmd
}

func newRunnerClaimCommand() *cobra.Command {
	var serverURL string
	var name string
	var leaseSeconds int
	var tokenEnv string
	cmd := &cobra.Command{
		Use:   "claim --name <runner-id>",
		Short: "Claim one queued job for a runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			token, err := runnerTokenFromEnv(tokenEnv)
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/api/v1/runners/%s/jobs/claim?leaseSeconds=%d", name, leaseSeconds)
			payload, err := doJSONWithToken(cmd.Context(), http.MethodPost, serverURL, path, nil, token)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&name, "name", "local-runner", "runner ID")
	cmd.Flags().IntVar(&leaseSeconds, "lease-seconds", 30, "claim lease duration in seconds")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_RUNNER_TOKEN", "environment variable containing the runner token")
	return cmd
}

func newRunnerOfflineDetectCommand() *cobra.Command {
	var serverURL string
	var timeoutSeconds int
	cmd := &cobra.Command{
		Use:   "offline-detect",
		Short: "Mark stale online runners offline",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := fmt.Sprintf("/api/v1/runners/offline-detect?timeoutSeconds=%d", timeoutSeconds)
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, path, nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().IntVar(&timeoutSeconds, "timeout-seconds", 60, "heartbeat age threshold in seconds")
	return cmd
}

func newRunnerAppendLogCommand() *cobra.Command {
	var serverURL string
	var pipelineRunID string
	var stageRunID string
	var stepRunID string
	var stream string
	var content string
	var runnerID string
	var tokenEnv string
	cmd := &cobra.Command{
		Use:   "logs append <job-run-id> --pipeline-run-id <id> --content <text>",
		Short: "Append a log chunk for a claimed job",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "append" {
				return fmt.Errorf("expected subcommand append")
			}
			if pipelineRunID == "" {
				return fmt.Errorf("--pipeline-run-id is required")
			}
			if runnerID == "" {
				return fmt.Errorf("--runner-id is required")
			}
			token, err := runnerTokenFromEnv(tokenEnv)
			if err != nil {
				return err
			}
			body, err := json.Marshal(map[string]any{
				"pipelineRunId": pipelineRunID,
				"stageRunId":    stageRunID,
				"stepRunId":     stepRunID,
				"stream":        stream,
				"content":       content,
			})
			if err != nil {
				return err
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodPost, serverURL, "/api/v1/runners/"+runnerID+"/jobs/"+args[1]+"/logs", body, token)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&runnerID, "runner-id", "local-runner", "runner ID")
	cmd.Flags().StringVar(&pipelineRunID, "pipeline-run-id", "", "PipelineRun ID")
	cmd.Flags().StringVar(&stageRunID, "stage-run-id", "", "StageRun ID")
	cmd.Flags().StringVar(&stepRunID, "step-run-id", "", "StepRun ID")
	cmd.Flags().StringVar(&stream, "stream", "stdout", "log stream")
	cmd.Flags().StringVar(&content, "content", "", "log content")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_RUNNER_TOKEN", "environment variable containing the runner token")
	return cmd
}

func newRunnerUpdateStatusCommand() *cobra.Command {
	var serverURL string
	var status string
	var reason string
	var runnerID string
	var tokenEnv string
	cmd := &cobra.Command{
		Use:   "status update <job-run-id> --status <status>",
		Short: "Update the status of a claimed job",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "update" {
				return fmt.Errorf("expected subcommand update")
			}
			if status == "" {
				return fmt.Errorf("--status is required")
			}
			if runnerID == "" {
				return fmt.Errorf("--runner-id is required")
			}
			token, err := runnerTokenFromEnv(tokenEnv)
			if err != nil {
				return err
			}
			body, err := json.Marshal(map[string]any{"status": status, "reason": reason})
			if err != nil {
				return err
			}
			payload, err := doJSONWithToken(cmd.Context(), http.MethodPost, serverURL, "/api/v1/runners/"+runnerID+"/jobs/"+args[1]+"/status", body, token)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&runnerID, "runner-id", "local-runner", "runner ID")
	cmd.Flags().StringVar(&status, "status", "", "job status")
	cmd.Flags().StringVar(&reason, "reason", "", "status reason")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_RUNNER_TOKEN", "environment variable containing the runner token")
	return cmd
}

func runnerTokenFromEnv(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("runner token env var name is required")
	}
	token := os.Getenv(name)
	if token == "" {
		return "", fmt.Errorf("%s is not set", name)
	}
	return token, nil
}

func newRuntimeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runtime",
		Short: "Runtime recovery utilities",
	}
	cmd.AddCommand(newRuntimeInspectCommand("status", "Show recoverable runtime work", http.MethodGet, "/api/v1/system/runtime/recovery"))
	cmd.AddCommand(newRuntimeInspectCommand("reconcile", "Run one runtime reconciliation pass", http.MethodPost, "/api/v1/system/runtime/reconcile"))
	return cmd
}

func newRuntimeInspectCommand(name string, short string, method string, path string) *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), method, serverURL, path, nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func doJSON(ctx context.Context, method string, serverURL string, path string, body []byte) (any, error) {
	return doJSONWithToken(ctx, method, serverURL, path, body, "")
}

func doRaw(ctx context.Context, method string, serverURL string, path string, body []byte) ([]byte, error) {
	if serverURL == "" {
		return nil, fmt.Errorf("server URL is required")
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(serverURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return respBody, fmt.Errorf("server returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	return respBody, nil
}

func doJSONWithToken(ctx context.Context, method string, serverURL string, path string, body []byte, token string) (any, error) {
	if serverURL == "" {
		return nil, fmt.Errorf("server URL is required")
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(serverURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var payload any
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &payload); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
	}
	if resp.StatusCode >= 400 {
		return payload, fmt.Errorf("server returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	return payload, nil
}

func printPipelineRunSummary(w io.Writer, payload any) {
	record, _ := payload.(map[string]any)
	run, _ := record["run"].(map[string]any)
	if run == nil {
		printJSON(w, payload)
		return
	}
	fmt.Fprintf(w, "PipelineRun: %v\n", run["id"])
	fmt.Fprintf(w, "Status: %v\n", run["status"])
	if failure, _ := run["failureReason"].(string); failure != "" {
		fmt.Fprintf(w, "Failure: %s\n", failure)
	}
	if logs, _ := record["logs"].([]any); logs != nil {
		fmt.Fprintf(w, "Logs: %d chunk(s)\n", len(logs))
	}
}

func printLogSummary(w io.Writer, payload any) {
	record, _ := payload.(map[string]any)
	logs, _ := record["logs"].([]any)
	if logs == nil {
		return
	}
	for _, item := range logs {
		log, _ := item.(map[string]any)
		if log == nil {
			continue
		}
		fmt.Fprintf(w, "[%v] %v", log["stream"], log["content"])
	}
}

func printDeploymentPlanSummary(w io.Writer, plan deploymentusecase.DeploymentPlan) {
	fmt.Fprintf(w, "DeploymentRun: %s\n", plan.DeploymentRunID)
	fmt.Fprintf(w, "Target: %s\n", plan.TargetType)
	fmt.Fprintf(w, "Namespace: %s\n", plan.Namespace)
	fmt.Fprintf(w, "Manifests: %d\n", plan.ManifestCount)
	fmt.Fprintf(w, "DryRun: %t\n", plan.DryRun)
	fmt.Fprintf(w, "Apply: %t\n", plan.Apply)
	if plan.DiffSummary != "" {
		fmt.Fprintf(w, "Diff: %s\n", plan.DiffSummary)
	}
	if len(plan.Warnings) > 0 {
		fmt.Fprintf(w, "Warnings:\n")
		for _, warning := range plan.Warnings {
			fmt.Fprintf(w, "- %s\n", warning)
		}
	}
}

func printGitOpsPlanSummary(w io.Writer, plan deploymentusecase.GitOpsChangePlan) {
	fmt.Fprintf(w, "Application: %s\n", plan.ApplicationName)
	fmt.Fprintf(w, "Repo: %s\n", plan.RepoURL)
	fmt.Fprintf(w, "Path: %s\n", plan.Path)
	fmt.Fprintf(w, "Revision: %s\n", plan.Revision)
	fmt.Fprintf(w, "Files: %d\n", len(plan.Files))
	fmt.Fprintf(w, "Artifacts: %d\n", len(plan.ArtifactChanges))
	fmt.Fprintf(w, "SyncRequested: %t\n", plan.SyncRequested)
	if len(plan.Warnings) > 0 {
		fmt.Fprintf(w, "Warnings:\n")
		for _, warning := range plan.Warnings {
			fmt.Fprintf(w, "- %s\n", warning)
		}
	}
}

func printReleasePlanSummary(w io.Writer, plan releaseorchestration.ReleasePlan) {
	fmt.Fprintf(w, "ReleasePlan: %s\n", plan.ID)
	fmt.Fprintf(w, "Release: %s\n", plan.ReleaseID)
	fmt.Fprintf(w, "Environment: %s\n", plan.EnvironmentName)
	fmt.Fprintf(w, "Strategy: %s\n", plan.Strategy)
	fmt.Fprintf(w, "Targets: %d\n", len(plan.Targets))
	fmt.Fprintf(w, "DeploymentPlans: %d\n", len(plan.DeploymentPlans))
	if len(plan.ArtifactSummary) > 0 {
		fmt.Fprintf(w, "Artifacts: %d\n", len(plan.ArtifactSummary))
	}
	if len(plan.Warnings) > 0 {
		fmt.Fprintf(w, "Warnings:\n")
		for _, warning := range plan.Warnings {
			fmt.Fprintf(w, "- %s\n", warning)
		}
	}
}

func printDeploymentRunSummary(w io.Writer, payload any) {
	record, _ := payload.(map[string]any)
	run, _ := record["run"].(map[string]any)
	if run == nil {
		printJSON(w, payload)
		return
	}
	fmt.Fprintf(w, "DeploymentRun: %v\n", run["id"])
	fmt.Fprintf(w, "Status: %v\n", run["status"])
	if reason, _ := run["reason"].(string); reason != "" {
		fmt.Fprintf(w, "Reason: %s\n", reason)
	}
	if plan, _ := record["plan"].(map[string]any); plan != nil {
		fmt.Fprintf(w, "Manifests: %v\n", plan["manifestCount"])
	}
	if logs, _ := record["logs"].([]any); logs != nil {
		fmt.Fprintf(w, "Logs: %d chunk(s)\n", len(logs))
	}
}

func printJSON(w io.Writer, payload any) {
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintf(w, "%v\n", payload)
		return
	}
	fmt.Fprintf(w, "%s\n", encoded)
}

func envValue(name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	return os.Getenv(name)
}
