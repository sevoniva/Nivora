package ssh

import (
	"context"
	"fmt"
	"strings"
	"time"

	portexecutor "github.com/sevoniva/nivora/internal/ports/executor"
)

type CommandRequest struct {
	HostID         string
	HostName       string
	Address        string
	Command        string
	CredentialRef  string
	TimeoutSeconds int
}

type UploadRequest struct {
	HostID         string
	HostName       string
	Address        string
	Source         string
	Destination    string
	CredentialRef  string
	TimeoutSeconds int
}

type Runner interface {
	Run(ctx context.Context, request CommandRequest) (portexecutor.HostDeploymentResult, error)
	Upload(ctx context.Context, request UploadRequest) (portexecutor.HostDeploymentResult, error)
}

type Executor struct {
	runner Runner
}

func New() Executor {
	return Executor{}
}

func NewWithRunner(runner Runner) Executor {
	return Executor{runner: runner}
}

func (e Executor) Prepare(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	if err := validateRemoteRequest(request); err != nil {
		return rejected(ctx, request, err)
	}
	return e.run(ctx, request, fmt.Sprintf("mkdir -p %s", shellQuote(request.ReleaseDir)), "prepared host release directory")
}

func (e Executor) Upload(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	if err := validateRemoteRequest(request); err != nil {
		return rejected(ctx, request, err)
	}
	if e.runner == nil {
		return rejected(ctx, request, fmt.Errorf("ssh runner transport is not configured"))
	}
	result, err := e.runner.Upload(ctx, UploadRequest{
		HostID:         request.HostID,
		HostName:       request.HostName,
		Address:        request.Address,
		Source:         request.Artifact,
		Destination:    request.ReleaseDir + "/artifact",
		CredentialRef:  request.CredentialRef,
		TimeoutSeconds: timeoutSeconds(request),
	})
	return redactResult(result), redactError(err)
}

func (e Executor) Execute(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	if err := validateRemoteRequest(request); err != nil {
		return rejected(ctx, request, err)
	}
	commands := []string{
		fmt.Sprintf("ln -sfn %s %s", shellQuote(request.ReleaseDir), shellQuote(request.DeployPath+"/next")),
		fmt.Sprintf("if [ -L %s ]; then ln -sfn $(readlink %s) %s; fi", shellQuote(request.DeployPath+"/current"), shellQuote(request.DeployPath+"/current"), shellQuote(request.DeployPath+"/previous")),
		fmt.Sprintf("ln -sfn %s %s", shellQuote(request.ReleaseDir), shellQuote(request.DeployPath+"/current")),
	}
	if request.RestartCommand != "" {
		commands = append(commands, request.RestartCommand)
	} else if request.ServiceName != "" {
		commands = append(commands, "systemctl restart "+shellQuote(request.ServiceName))
	}
	return e.run(ctx, request, strings.Join(commands, " && "), "host symlink switch executed")
}

func (e Executor) HealthCheck(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	if err := validateRemoteRequest(request); err != nil {
		return rejected(ctx, request, err)
	}
	check := strings.TrimSpace(request.HealthCheck)
	if check == "" {
		return success(ctx, request, "host health check skipped; no health check configured", "")
	}
	switch request.HealthCheckType {
	case "http":
		return e.run(ctx, request, "curl -fsS --max-time "+fmt.Sprint(timeoutSeconds(request))+" "+shellQuote(check), "HTTP health check completed")
	case "tcp":
		host, port, ok := strings.Cut(check, ":")
		if !ok || host == "" || port == "" {
			return rejected(ctx, request, fmt.Errorf("tcp health check must be host:port"))
		}
		return e.run(ctx, request, "nc -z -w "+fmt.Sprint(timeoutSeconds(request))+" "+shellQuote(host)+" "+shellQuote(port), "TCP health check completed")
	default:
		return e.run(ctx, request, check, "command health check completed")
	}
}

func (e Executor) Rollback(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	if err := validateRemoteRequest(request); err != nil {
		return rejected(ctx, request, err)
	}
	commands := []string{
		fmt.Sprintf("if [ -L %s ]; then ln -sfn $(readlink %s) %s; fi", shellQuote(request.DeployPath+"/current"), shellQuote(request.DeployPath+"/current"), shellQuote(request.DeployPath+"/next")),
		fmt.Sprintf("if [ -L %s ]; then ln -sfn $(readlink %s) %s; fi", shellQuote(request.DeployPath+"/previous"), shellQuote(request.DeployPath+"/previous"), shellQuote(request.DeployPath+"/current")),
	}
	if request.RestartCommand != "" {
		commands = append(commands, request.RestartCommand)
	} else if request.ServiceName != "" {
		commands = append(commands, "systemctl restart "+shellQuote(request.ServiceName))
	}
	return e.run(ctx, request, strings.Join(commands, " && "), "host symlink rollback executed")
}

func (e Executor) run(ctx context.Context, request portexecutor.HostDeploymentRequest, command string, message string) (portexecutor.HostDeploymentResult, error) {
	select {
	case <-ctx.Done():
		return portexecutor.HostDeploymentResult{}, ctx.Err()
	default:
	}
	if e.runner == nil {
		return rejected(ctx, request, fmt.Errorf("ssh runner transport is not configured"))
	}
	runCtx := ctx
	cancel := func() {}
	if timeoutSeconds(request) > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds(request))*time.Second)
	}
	defer cancel()
	result, err := e.runner.Run(runCtx, CommandRequest{
		HostID:         request.HostID,
		HostName:       request.HostName,
		Address:        request.Address,
		Command:        command,
		CredentialRef:  request.CredentialRef,
		TimeoutSeconds: timeoutSeconds(request),
	})
	if result.Message == "" {
		result.Message = message
	}
	return redactResult(result), redactError(err)
}

func validateRemoteRequest(request portexecutor.HostDeploymentRequest) error {
	if !request.Apply || !request.Confirmed || !request.AllowRemote {
		return fmt.Errorf("remote SSH host deployment requires apply=true, confirm=true, and allowRemote=true")
	}
	if request.CredentialRef == "" {
		return fmt.Errorf("remote SSH host deployment requires CredentialRef")
	}
	if request.Address == "" {
		return fmt.Errorf("remote SSH host deployment requires host address")
	}
	if request.ReleaseDir == "" || request.DeployPath == "" {
		return fmt.Errorf("remote SSH host deployment requires releaseDir and deployPath")
	}
	return nil
}

func rejected(ctx context.Context, request portexecutor.HostDeploymentRequest, err error) (portexecutor.HostDeploymentResult, error) {
	select {
	case <-ctx.Done():
		return portexecutor.HostDeploymentResult{}, ctx.Err()
	default:
	}
	clean := redactError(err)
	return portexecutor.HostDeploymentResult{HostID: request.HostID, HostName: request.HostName, Status: "Rejected", Message: clean.Error()}, clean
}

func success(ctx context.Context, request portexecutor.HostDeploymentRequest, message string, command string) (portexecutor.HostDeploymentResult, error) {
	select {
	case <-ctx.Done():
		return portexecutor.HostDeploymentResult{}, ctx.Err()
	default:
	}
	return portexecutor.HostDeploymentResult{HostID: request.HostID, HostName: request.HostName, Status: "Succeeded", Message: message, Command: command}, nil
}

func timeoutSeconds(request portexecutor.HostDeploymentRequest) int {
	if request.TimeoutSeconds > 0 {
		return request.TimeoutSeconds
	}
	return 30
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func redactResult(result portexecutor.HostDeploymentResult) portexecutor.HostDeploymentResult {
	result.Stdout = redact(result.Stdout)
	result.Stderr = redact(result.Stderr)
	result.Message = redact(result.Message)
	return result
}

func redactError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s", redact(err.Error()))
}

func redact(value string) string {
	replacements := []string{"password", "token", "secret", "private_key", "authorization", "credential"}
	clean := value
	for _, marker := range replacements {
		clean = strings.ReplaceAll(clean, marker, "[redacted]")
		clean = strings.ReplaceAll(clean, strings.ToUpper(marker), "[redacted]")
	}
	return clean
}

var _ portexecutor.HostExecutor = Executor{}
