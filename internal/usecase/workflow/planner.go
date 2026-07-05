package workflow

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

var ErrInvalid = errors.New("workflow definition is invalid")

func PlanDefinition(def Definition, options PlanOptions) (Plan, error) {
	options = normalizeOptions(options)
	if err := validateDefinitionShape(def, options); err != nil {
		return Plan{}, err
	}
	order, levels, err := topologicalOrder(def.Jobs)
	if err != nil {
		return Plan{}, err
	}
	artifacts, caches, outputWarnings, err := normalizeWorkflowOutputs(def)
	if err != nil {
		return Plan{}, err
	}
	securityIntent, err := planSecurityIntent(def.Security)
	if err != nil {
		return Plan{}, err
	}
	releaseIntent, err := planReleaseIntent(def.Release)
	if err != nil {
		return Plan{}, err
	}
	deploymentIntent, err := planDeploymentIntent(def.Deployment)
	if err != nil {
		return Plan{}, err
	}
	permissionRequests, permissionWarnings, err := planPermissionRequests(def.Permissions)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{
		WorkflowID:         workflowID(def),
		Name:               def.Metadata.Name,
		Triggers:           append([]string(nil), def.On.Events...),
		PermissionRequests: permissionRequests,
		ArtifactOutputs:    artifacts,
		CacheHints:         caches,
		SecurityIntent:     securityIntent,
		ReleaseIntent:      releaseIntent,
		DeploymentIntent:   deploymentIntent,
		EstimatedMode:      "plan-only",
		ConversionReady:    true,
		Warnings:           append([]string(nil), outputWarnings...),
		CreatedAt:          time.Now().UTC(),
	}
	plan.Warnings = append(plan.Warnings, triggerWarnings(def.On.Events)...)
	plan.SecurityWarnings = append(plan.SecurityWarnings, intentWarnings(securityIntent, releaseIntent, deploymentIntent)...)
	plan.SecurityWarnings = append(plan.SecurityWarnings, permissionWarnings...)
	for _, jobID := range order {
		job := def.Jobs[jobID]
		expansions, err := expandMatrix(job.Strategy.Matrix, options.MaxMatrixSize)
		if err != nil {
			return Plan{}, fmt.Errorf("%w: job %q %v", ErrInvalid, jobID, err)
		}
		if len(expansions) == 0 {
			expansions = []map[string]string{{}}
		}
		for index, matrix := range expansions {
			plannedID := plannedJobID(jobID, matrix, index, len(expansions))
			conversionReady := true
			planned := PlannedJob{
				ID:              plannedID,
				BaseID:          jobID,
				Name:            firstNonEmpty(job.Name, jobID),
				Needs:           append([]string(nil), job.Needs...),
				RunsOn:          append([]string(nil), job.RunsOn...),
				Labels:          copyStringMap(job.Labels),
				Matrix:          copyStringMap(matrix),
				TimeoutMinutes:  job.TimeoutMinutes,
				StepCount:       len(job.Steps),
				ConversionReady: true,
			}
			for stepIndex, step := range job.Steps {
				stepReady := step.Uses == ""
				if !stepReady {
					conversionReady = false
					plan.ConversionReady = false
					plan.UnsupportedFeatures = append(plan.UnsupportedFeatures, fmt.Sprintf("job %s step %d uses %q; external action execution is foundation-only", jobID, stepIndex, step.Uses))
				}
				if step.Run == "" && step.Uses == "" {
					return Plan{}, fmt.Errorf("%w: job %q step %d requires run or uses", ErrInvalid, jobID, stepIndex)
				}
				mergedEnv := mergeEnv(def.Env, job.Env, step.Env)
				if err := validateEnv(mergedEnv, options.MaxEnvSize); err != nil {
					return Plan{}, fmt.Errorf("%w: job %q step %d: %v", ErrInvalid, jobID, stepIndex, err)
				}
				plan.Steps = append(plan.Steps, PlannedStep{
					ID:              fmt.Sprintf("%s/step-%d", plannedID, stepIndex+1),
					JobID:           plannedID,
					Name:            firstNonEmpty(step.Name, fmt.Sprintf("step-%d", stepIndex+1)),
					Run:             step.Run,
					Uses:            step.Uses,
					TimeoutMinutes:  step.TimeoutMinutes,
					ContinueOnError: step.ContinueOnError,
					Env:             redactEnv(mergedEnv),
					ConversionReady: stepReady,
				})
			}
			planned.ConversionReady = conversionReady
			plan.Jobs = append(plan.Jobs, planned)
			if len(matrix) > 0 {
				plan.MatrixExpansions = append(plan.MatrixExpansions, MatrixExpansion{JobID: plannedID, Values: copyStringMap(matrix)})
			}
			if len(job.RunsOn) > 0 {
				plan.RunnerRequirements = append(plan.RunnerRequirements, RunnerRequirement{JobID: plannedID, RunsOn: append([]string(nil), job.RunsOn...)})
			} else {
				plan.SecurityWarnings = append(plan.SecurityWarnings, fmt.Sprintf("job %s has no runsOn labels; runner matching will be broad", jobID))
			}
		}
	}
	for _, jobID := range order {
		for _, dep := range def.Jobs[jobID].Needs {
			plan.Edges = append(plan.Edges, Edge{From: dep, To: jobID})
		}
	}
	for _, level := range levels {
		sort.Strings(level)
	}
	plan.Warnings = dedupeSorted(plan.Warnings)
	plan.SecurityWarnings = dedupeSorted(plan.SecurityWarnings)
	plan.UnsupportedFeatures = dedupeSorted(plan.UnsupportedFeatures)
	return plan, nil
}

func ToPipelineDefinition(def Definition, options PlanOptions) (PipelineConversion, error) {
	plan, err := PlanDefinition(def, options)
	if err != nil {
		return PipelineConversion{}, err
	}
	if !plan.ConversionReady {
		return PipelineConversion{}, fmt.Errorf("%w: workflow contains foundation-only features that cannot execute as PipelineRun", ErrInvalid)
	}
	_, levels, err := topologicalOrder(def.Jobs)
	if err != nil {
		return PipelineConversion{}, err
	}
	pipeline := pipelineusecase.Definition{
		APIVersion: def.APIVersion,
		Kind:       "Pipeline",
		Metadata:   pipelineusecase.Metadata{Name: def.Metadata.Name},
	}
	for levelIndex, level := range levels {
		stage := pipelineusecase.Stage{Name: fmt.Sprintf("workflow-%d", levelIndex+1)}
		for _, jobID := range level {
			job := def.Jobs[jobID]
			expansions, err := expandMatrix(job.Strategy.Matrix, normalizeOptions(options).MaxMatrixSize)
			if err != nil {
				return PipelineConversion{}, err
			}
			if len(expansions) == 0 {
				expansions = []map[string]string{{}}
			}
			for index, matrix := range expansions {
				pipelineJob := pipelineusecase.Job{
					Name:           plannedJobID(jobID, matrix, index, len(expansions)),
					Executor:       "shell",
					Labels:         copyStringMap(job.Labels),
					TimeoutSeconds: job.TimeoutMinutes * 60,
				}
				for stepIndex, step := range job.Steps {
					if step.Uses != "" {
						return PipelineConversion{}, fmt.Errorf("%w: uses step %q is not executable by PipelineRun foundation", ErrInvalid, step.Uses)
					}
					pipelineJob.Steps = append(pipelineJob.Steps, pipelineusecase.Step{
						Name:           firstNonEmpty(step.Name, fmt.Sprintf("step-%d", stepIndex+1)),
						Run:            step.Run,
						TimeoutSeconds: step.TimeoutMinutes * 60,
					})
				}
				stage.Jobs = append(stage.Jobs, pipelineJob)
			}
		}
		pipeline.Spec.Stages = append(pipeline.Spec.Stages, stage)
	}
	if err := pipeline.Validate(); err != nil {
		return PipelineConversion{}, err
	}
	return PipelineConversion{Definition: pipeline, Warnings: []string{"workflow converted to PipelineRun-compatible staged shell jobs; advanced provider action semantics are not executed"}}, nil
}

func ToPipelineDefinitionFromPlan(plan Plan) (PipelineConversion, error) {
	if !plan.ConversionReady {
		return PipelineConversion{}, fmt.Errorf("%w: workflow plan contains foundation-only features that cannot execute as PipelineRun", ErrInvalid)
	}
	if strings.TrimSpace(plan.WorkflowID) == "" {
		return PipelineConversion{}, fmt.Errorf("%w: workflow plan id is required", ErrInvalid)
	}
	stepsByJob := map[string][]PlannedStep{}
	for _, step := range plan.Steps {
		if step.Uses != "" || !step.ConversionReady {
			return PipelineConversion{}, fmt.Errorf("%w: uses step %q is not executable by PipelineRun foundation", ErrInvalid, step.Uses)
		}
		stepsByJob[step.JobID] = append(stepsByJob[step.JobID], step)
	}
	pipeline := pipelineusecase.Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Pipeline",
		Metadata:   pipelineusecase.Metadata{Name: firstNonEmpty(plan.Name, plan.WorkflowID)},
	}
	for index, job := range plan.Jobs {
		if !job.ConversionReady {
			return PipelineConversion{}, fmt.Errorf("%w: workflow job %q is not executable by PipelineRun foundation", ErrInvalid, job.ID)
		}
		stage := pipelineusecase.Stage{Name: fmt.Sprintf("workflow-%d", index+1)}
		pipelineJob := pipelineusecase.Job{
			Name:           firstNonEmpty(job.ID, job.Name),
			Executor:       "shell",
			Labels:         copyStringMap(job.Labels),
			TimeoutSeconds: job.TimeoutMinutes * 60,
		}
		for stepIndex, step := range stepsByJob[job.ID] {
			if strings.TrimSpace(step.Run) == "" {
				return PipelineConversion{}, fmt.Errorf("%w: workflow job %q step %d has no run command", ErrInvalid, job.ID, stepIndex)
			}
			pipelineJob.Steps = append(pipelineJob.Steps, pipelineusecase.Step{
				Name:           firstNonEmpty(step.Name, fmt.Sprintf("step-%d", stepIndex+1)),
				Run:            step.Run,
				TimeoutSeconds: step.TimeoutMinutes * 60,
			})
		}
		if len(pipelineJob.Steps) == 0 {
			return PipelineConversion{}, fmt.Errorf("%w: workflow job %q has no executable steps", ErrInvalid, job.ID)
		}
		stage.Jobs = append(stage.Jobs, pipelineJob)
		pipeline.Spec.Stages = append(pipeline.Spec.Stages, stage)
	}
	if err := pipeline.Validate(); err != nil {
		return PipelineConversion{}, err
	}
	warnings := []string{
		"workflow plan converted to a queued PipelineRun; stage ordering is derived from the stored plan and external action semantics are not executed",
	}
	warnings = append(warnings, plan.SecurityWarnings...)
	warnings = append(warnings, plan.Warnings...)
	return PipelineConversion{Definition: pipeline, Warnings: dedupeSorted(warnings)}, nil
}

func validateDefinitionShape(def Definition, options PlanOptions) error {
	if def.Kind != "Workflow" {
		return fmt.Errorf("%w: workflow kind must be Workflow", ErrInvalid)
	}
	if strings.TrimSpace(def.Metadata.Name) == "" {
		return fmt.Errorf("%w: workflow metadata.name is required", ErrInvalid)
	}
	if len(def.Jobs) == 0 {
		return fmt.Errorf("%w: workflow must define at least one job", ErrInvalid)
	}
	if len(def.Jobs) > options.MaxJobs {
		return fmt.Errorf("%w: workflow defines %d jobs, max %d", ErrInvalid, len(def.Jobs), options.MaxJobs)
	}
	if err := validateEnv(def.Env, options.MaxEnvSize); err != nil {
		return fmt.Errorf("%w: workflow env: %v", ErrInvalid, err)
	}
	totalSteps := 0
	for id, job := range def.Jobs {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("%w: job id is required", ErrInvalid)
		}
		if job.TimeoutMinutes < 0 {
			return fmt.Errorf("%w: job %q timeoutMinutes must be zero or greater", ErrInvalid, id)
		}
		if err := validateEnv(job.Env, options.MaxEnvSize); err != nil {
			return fmt.Errorf("%w: job %q env: %v", ErrInvalid, id, err)
		}
		if err := validateWorkflowJobLabels(id, job.Labels); err != nil {
			return err
		}
		if len(job.Steps) == 0 {
			return fmt.Errorf("%w: job %q must define at least one step", ErrInvalid, id)
		}
		totalSteps += len(job.Steps)
		if totalSteps > options.MaxSteps {
			return fmt.Errorf("%w: workflow defines more than %d steps", ErrInvalid, options.MaxSteps)
		}
		for i, step := range job.Steps {
			if step.TimeoutMinutes < 0 {
				return fmt.Errorf("%w: job %q step %d timeoutMinutes must be zero or greater", ErrInvalid, id, i)
			}
			if err := validateEnv(step.Env, options.MaxEnvSize); err != nil {
				return fmt.Errorf("%w: job %q step %d env: %v", ErrInvalid, id, i, err)
			}
		}
	}
	if err := validateIntentSecrets("security", def.Security); err != nil {
		return err
	}
	if err := validateIntentSecrets("release", def.Release); err != nil {
		return err
	}
	if err := validateIntentSecrets("deployment", def.Deployment); err != nil {
		return err
	}
	return nil
}

func normalizeWorkflowOutputs(def Definition) ([]ArtifactSpec, []CacheSpec, []string, error) {
	artifacts := make([]ArtifactSpec, 0, len(def.Artifacts))
	caches := make([]CacheSpec, 0, len(def.Cache))
	warnings := []string{}
	for index, artifact := range def.Artifacts {
		artifact.Name = strings.TrimSpace(artifact.Name)
		artifact.Path = strings.TrimSpace(artifact.Path)
		artifact.Type = strings.TrimSpace(artifact.Type)
		artifact.ContentHash = strings.TrimSpace(artifact.ContentHash)
		artifact.StorageRef = strings.TrimSpace(artifact.StorageRef)
		artifact.Metadata = compactStringMap(artifact.Metadata)
		if artifact.Name == "" {
			return nil, nil, nil, fmt.Errorf("%w: artifact %d name is required", ErrInvalid, index)
		}
		if artifact.Path == "" && artifact.StorageRef == "" {
			return nil, nil, nil, fmt.Errorf("%w: artifact %q requires path or storageRef", ErrInvalid, artifact.Name)
		}
		if artifact.RetentionDays < 0 {
			return nil, nil, nil, fmt.Errorf("%w: artifact %q retentionDays must be zero or greater", ErrInvalid, artifact.Name)
		}
		if err := validateStringMetadata("artifact "+artifact.Name, artifact.Metadata); err != nil {
			return nil, nil, nil, err
		}
		if artifact.StorageRef != "" {
			warnings = append(warnings, "artifact "+artifact.Name+" declares storageRef metadata only; workflow planning does not read artifact content")
		}
		artifacts = append(artifacts, artifact)
	}
	for index, cache := range def.Cache {
		cache.Key = strings.TrimSpace(cache.Key)
		cache.Path = compactStrings(cache.Path)
		cache.RestoreKeys = compactStrings(cache.RestoreKeys)
		cache.Scope = strings.TrimSpace(cache.Scope)
		cache.Metadata = compactStringMap(cache.Metadata)
		if cache.Key == "" {
			return nil, nil, nil, fmt.Errorf("%w: cache %d key is required", ErrInvalid, index)
		}
		if len(cache.Path) == 0 {
			return nil, nil, nil, fmt.Errorf("%w: cache %q requires at least one path", ErrInvalid, cache.Key)
		}
		if err := validateStringMetadata("cache "+cache.Key, cache.Metadata); err != nil {
			return nil, nil, nil, err
		}
		caches = append(caches, cache)
	}
	return artifacts, caches, warnings, nil
}

func planSecurityIntent(values map[string]any) (*SecurityIntentPlan, error) {
	if len(values) == 0 {
		return nil, nil
	}
	allowed := map[string]struct{}{"enabled": {}, "scanners": {}, "required": {}, "policy": {}}
	intent := &SecurityIntentPlan{
		Enabled:  boolValue(values, "enabled", true),
		Scanners: stringSliceValue(values, "scanners"),
		Required: boolValue(values, "required", false),
		Policy:   stringValue(values, "policy"),
		PlanOnly: true,
		Warnings: []string{"security intent is plan-only; workflow planning does not execute scanners"},
	}
	intent.UnsupportedKeys = unsupportedKeys(values, allowed)
	if len(intent.UnsupportedKeys) > 0 {
		intent.Warnings = append(intent.Warnings, "unsupported security keys are ignored in workflow planning: "+strings.Join(intent.UnsupportedKeys, ", "))
	}
	intent.Warnings = dedupeSorted(intent.Warnings)
	return intent, nil
}

func planReleaseIntent(values map[string]any) (*ReleaseIntentPlan, error) {
	if len(values) == 0 {
		return nil, nil
	}
	allowed := map[string]struct{}{"enabled": {}, "name": {}, "environment": {}, "artifacts": {}, "requireDigest": {}}
	intent := &ReleaseIntentPlan{
		Enabled:       boolValue(values, "enabled", true),
		Name:          stringValue(values, "name"),
		Environment:   stringValue(values, "environment"),
		Artifacts:     stringSliceValue(values, "artifacts"),
		RequireDigest: boolValue(values, "requireDigest", false),
		PlanOnly:      true,
		Warnings:      []string{"release intent is plan-only; workflow planning does not create releases or bind artifacts"},
	}
	intent.UnsupportedKeys = unsupportedKeys(values, allowed)
	if len(intent.UnsupportedKeys) > 0 {
		intent.Warnings = append(intent.Warnings, "unsupported release keys are ignored in workflow planning: "+strings.Join(intent.UnsupportedKeys, ", "))
	}
	intent.Warnings = dedupeSorted(intent.Warnings)
	return intent, nil
}

func planDeploymentIntent(values map[string]any) (*DeploymentIntentPlan, error) {
	if len(values) == 0 {
		return nil, nil
	}
	allowed := map[string]struct{}{"enabled": {}, "targetType": {}, "targetName": {}, "target": {}, "environment": {}, "apply": {}, "sync": {}, "planOnly": {}}
	target := stringValue(values, "targetName")
	if target == "" {
		target = stringValue(values, "target")
	}
	intent := &DeploymentIntentPlan{
		Enabled:        boolValue(values, "enabled", true),
		TargetType:     stringValue(values, "targetType"),
		TargetName:     target,
		Environment:    stringValue(values, "environment"),
		PlanOnly:       true,
		ApplyRequested: boolValue(values, "apply", false),
		SyncRequested:  boolValue(values, "sync", false),
		Warnings:       []string{"deployment intent is plan-only; workflow planning does not apply Kubernetes resources, sync Argo CD, or deploy hosts"},
	}
	if intent.ApplyRequested {
		intent.Warnings = append(intent.Warnings, "apply=true was requested but remains guarded and is not executed by workflow planning")
	}
	if intent.SyncRequested {
		intent.Warnings = append(intent.Warnings, "sync=true was requested but remains guarded and is not executed by workflow planning")
	}
	intent.UnsupportedKeys = unsupportedKeys(values, allowed)
	if len(intent.UnsupportedKeys) > 0 {
		intent.Warnings = append(intent.Warnings, "unsupported deployment keys are ignored in workflow planning: "+strings.Join(intent.UnsupportedKeys, ", "))
	}
	intent.Warnings = dedupeSorted(intent.Warnings)
	return intent, nil
}

func planPermissionRequests(values map[string]string) ([]PermissionRequest, []string, error) {
	if len(values) == 0 {
		return nil, nil, nil
	}
	scopes := make([]string, 0, len(values))
	for scope := range values {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	requests := make([]PermissionRequest, 0, len(scopes))
	warnings := []string{}
	for _, rawScope := range scopes {
		scope := strings.TrimSpace(rawScope)
		access := normalizePermissionAccess(values[rawScope])
		if scope == "" {
			return nil, nil, fmt.Errorf("%w: workflow permission scope is required", ErrInvalid)
		}
		if secretLike(scope) && strings.ToLower(scope) != "id-token" {
			return nil, nil, fmt.Errorf("%w: workflow permission scope %q looks secret-like and is not allowed", ErrInvalid, scope)
		}
		if access == "" {
			return nil, nil, fmt.Errorf("%w: workflow permission %q access is required", ErrInvalid, scope)
		}
		if secretLike(access) {
			return nil, nil, fmt.Errorf("%w: workflow permission %q access looks secret-like and is not allowed", ErrInvalid, scope)
		}
		request := PermissionRequest{
			Scope:    scope,
			Access:   access,
			PlanOnly: true,
		}
		if !knownPermissionAccess(access) {
			request.Warnings = append(request.Warnings, fmt.Sprintf("workflow permission %s requests unknown access %q; runtime authorization remains RBAC-gated", scope, access))
		}
		if broadPermissionAccess(access) {
			request.Warnings = append(request.Warnings, fmt.Sprintf("workflow permission %s requests %s access; execution remains guarded by RBAC, runner policy, and explicit confirmation", scope, access))
		}
		if strings.EqualFold(scope, "id-token") && access != "none" {
			request.Warnings = append(request.Warnings, "workflow id-token permission is foundation-only; Nivora workflow planning does not mint identity tokens")
		}
		request.Warnings = dedupeSorted(request.Warnings)
		warnings = append(warnings, request.Warnings...)
		requests = append(requests, request)
	}
	return requests, dedupeSorted(warnings), nil
}

func normalizePermissionAccess(value string) string {
	access := strings.ToLower(strings.TrimSpace(value))
	access = strings.ReplaceAll(access, "_", "-")
	switch access {
	case "read-only":
		return "read"
	case "plan-only":
		return "plan"
	default:
		return access
	}
}

func knownPermissionAccess(access string) bool {
	switch access {
	case "none", "read", "plan", "write", "run", "admin":
		return true
	default:
		return false
	}
}

func broadPermissionAccess(access string) bool {
	switch access {
	case "write", "run", "admin":
		return true
	default:
		return false
	}
}

func triggerWarnings(events []string) []string {
	warnings := []string{}
	for _, event := range events {
		switch event {
		case "schedule":
			warnings = append(warnings, "schedule trigger is a placeholder; no scheduler dispatch is registered by workflow planning")
		case "push", "pull_request", "tag", "repository_snapshot", "release_requested":
			warnings = append(warnings, event+" trigger is modeled for planning; provider webhook dispatch is not required in this foundation")
		}
	}
	return warnings
}

func intentWarnings(intents ...any) []string {
	warnings := []string{}
	for _, intent := range intents {
		switch typed := intent.(type) {
		case *SecurityIntentPlan:
			if typed != nil {
				warnings = append(warnings, typed.Warnings...)
			}
		case *ReleaseIntentPlan:
			if typed != nil {
				warnings = append(warnings, typed.Warnings...)
			}
		case *DeploymentIntentPlan:
			if typed != nil {
				warnings = append(warnings, typed.Warnings...)
			}
		}
	}
	return warnings
}

func validateIntentSecrets(section string, values map[string]any) error {
	for key, value := range values {
		if err := validateIntentValue(section+"."+key, key, value); err != nil {
			return err
		}
	}
	return nil
}

func validateIntentValue(path string, key string, value any) error {
	if allowedReferenceKey(key) {
		return nil
	}
	if secretLike(key) {
		text := strings.TrimSpace(fmt.Sprint(value))
		if !safeSecretRef(text) {
			return fmt.Errorf("%w: %s looks secret-like and must use secretRef: or credentialRef:", ErrInvalid, path)
		}
	}
	switch typed := value.(type) {
	case map[string]any:
		for childKey, childValue := range typed {
			if err := validateIntentValue(path+"."+childKey, childKey, childValue); err != nil {
				return err
			}
		}
	case []any:
		for index, item := range typed {
			if err := validateIntentValue(fmt.Sprintf("%s[%d]", path, index), key, item); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateStringMetadata(section string, values map[string]string) error {
	for key, value := range values {
		if allowedReferenceKey(key) {
			continue
		}
		if secretLike(key) && !safeSecretRef(value) {
			return fmt.Errorf("%w: %s metadata %q looks secret-like and must use secretRef: or credentialRef:", ErrInvalid, section, key)
		}
	}
	return nil
}

func validateWorkflowJobLabels(jobID string, values map[string]string) error {
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			return fmt.Errorf("%w: job %q labels must not contain empty keys or values", ErrInvalid, jobID)
		}
		if secretLike(key) || secretLike(value) {
			return fmt.Errorf("%w: job %q label %q looks secret-like and is not allowed", ErrInvalid, jobID, key)
		}
	}
	return nil
}

func allowedReferenceKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "_", ""))
	return normalized == "secretref" || normalized == "credentialref" || normalized == "keyref"
}

func unsupportedKeys(values map[string]any, allowed map[string]struct{}) []string {
	out := []string{}
	for key := range values {
		if _, ok := allowed[key]; !ok {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func stringValue(values map[string]any, key string) string {
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func boolValue(values map[string]any, key string, fallback bool) bool {
	value, ok := values[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "yes", "1":
			return true
		case "false", "no", "0":
			return false
		default:
			return fallback
		}
	default:
		return fallback
	}
}

func stringSliceValue(values map[string]any, key string) []string {
	value, ok := values[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return compactStrings(typed)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, strings.TrimSpace(fmt.Sprint(item)))
		}
		return compactStrings(out)
	case string:
		return compactStrings([]string{typed})
	default:
		return compactStrings([]string{fmt.Sprint(typed)})
	}
}

func topologicalOrder(jobs map[string]Job) ([]string, [][]string, error) {
	ids := make([]string, 0, len(jobs))
	for id := range jobs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	inDegree := map[string]int{}
	children := map[string][]string{}
	for _, id := range ids {
		inDegree[id] = 0
	}
	for id, job := range jobs {
		for _, dep := range job.Needs {
			if _, ok := jobs[dep]; !ok {
				return nil, nil, fmt.Errorf("%w: job %q needs unknown job %q", ErrInvalid, id, dep)
			}
			children[dep] = append(children[dep], id)
			inDegree[id]++
		}
	}
	ready := []string{}
	for _, id := range ids {
		if inDegree[id] == 0 {
			ready = append(ready, id)
		}
	}
	var order []string
	var levels [][]string
	for len(ready) > 0 {
		sort.Strings(ready)
		level := append([]string(nil), ready...)
		levels = append(levels, level)
		next := []string{}
		for _, id := range ready {
			order = append(order, id)
			for _, child := range children[id] {
				inDegree[child]--
				if inDegree[child] == 0 {
					next = append(next, child)
				}
			}
		}
		ready = next
	}
	if len(order) != len(jobs) {
		return nil, nil, fmt.Errorf("%w: workflow job dependency cycle detected", ErrInvalid)
	}
	return order, levels, nil
}

func expandMatrix(matrix Matrix, max int) ([]map[string]string, error) {
	if len(matrix.Values) == 0 && len(matrix.Include) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(matrix.Values))
	for key := range matrix.Values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	expansions := []map[string]string{{}}
	for _, key := range keys {
		values := append([]string(nil), matrix.Values[key]...)
		sort.Strings(values)
		if len(values) == 0 {
			return nil, fmt.Errorf("matrix key %q must include at least one value", key)
		}
		next := make([]map[string]string, 0, len(expansions)*len(values))
		for _, expansion := range expansions {
			for _, value := range values {
				copy := copyStringMap(expansion)
				if copy == nil {
					copy = map[string]string{}
				}
				copy[key] = value
				next = append(next, copy)
				if len(next) > max {
					return nil, fmt.Errorf("matrix expands beyond max %d", max)
				}
			}
		}
		expansions = next
	}
	expansions = applyMatrixExclude(expansions, matrix.Exclude)
	for _, include := range matrix.Include {
		expansions = append(expansions, copyStringMap(include))
		if len(expansions) > max {
			return nil, fmt.Errorf("matrix expands beyond max %d", max)
		}
	}
	sort.Slice(expansions, func(i, j int) bool { return matrixKey(expansions[i]) < matrixKey(expansions[j]) })
	return expansions, nil
}

func applyMatrixExclude(expansions []map[string]string, exclude []map[string]string) []map[string]string {
	if len(exclude) == 0 {
		return expansions
	}
	out := make([]map[string]string, 0, len(expansions))
	for _, expansion := range expansions {
		skip := false
		for _, excluded := range exclude {
			if matrixMatches(expansion, excluded) {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, expansion)
		}
	}
	return out
}

func matrixMatches(expansion map[string]string, expected map[string]string) bool {
	for key, value := range expected {
		if expansion[key] != value {
			return false
		}
	}
	return true
}

func validateEnv(env map[string]string, maxValueSize int) error {
	for key, value := range env {
		if len(value) > maxValueSize {
			return fmt.Errorf("env %q exceeds max value size", key)
		}
		if secretLike(key) && !safeSecretRef(value) {
			return fmt.Errorf("env %q looks secret-like and must use secretRef: or credentialRef:", key)
		}
	}
	return nil
}

func redactEnv(env map[string]string) map[string]string {
	if len(env) == 0 {
		return nil
	}
	out := make(map[string]string, len(env))
	for key, value := range env {
		if secretLike(key) {
			out[key] = "[REDACTED]"
			continue
		}
		out[key] = value
	}
	return out
}

func secretLike(key string) bool {
	lower := strings.ToLower(key)
	parts := []string{"token", "password", "secret", "private_key", "kubeconfig", "authorization", "access_key", "bearer"}
	for _, part := range parts {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return false
}

func safeSecretRef(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lower, "secretref:") || strings.HasPrefix(lower, "credentialref:")
}

func mergeEnv(values ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, env := range values {
		for key, value := range env {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func plannedJobID(base string, matrix map[string]string, index int, total int) string {
	if total <= 1 || len(matrix) == 0 {
		return base
	}
	return fmt.Sprintf("%s[%s]", base, matrixKey(matrix))
}

func matrixKey(matrix map[string]string) string {
	keys := make([]string, 0, len(matrix))
	for key := range matrix {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+matrix[key])
	}
	return strings.Join(parts, ",")
}

func workflowID(def Definition) string {
	name := strings.ToLower(def.Metadata.Name)
	name = strings.NewReplacer(" ", "-", "_", "-", "/", "-", "\\", "-").Replace(name)
	if name == "" {
		return "workflow"
	}
	return "workflow-" + name
}

func normalizeOptions(options PlanOptions) PlanOptions {
	if options.MaxJobs <= 0 {
		options.MaxJobs = DefaultMaxJobs
	}
	if options.MaxSteps <= 0 {
		options.MaxSteps = DefaultMaxSteps
	}
	if options.MaxMatrixSize <= 0 {
		options.MaxMatrixSize = DefaultMaxMatrixSize
	}
	if options.MaxEnvSize <= 0 {
		options.MaxEnvSize = DefaultMaxEnvSize
	}
	return options
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func compactStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, value := range in {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func dedupeSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
