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
	plan := Plan{
		WorkflowID:      workflowID(def),
		Name:            def.Metadata.Name,
		Triggers:        append([]string(nil), def.On.Events...),
		ArtifactOutputs: append([]ArtifactSpec(nil), def.Artifacts...),
		CacheHints:      append([]CacheSpec(nil), def.Cache...),
		EstimatedMode:   "plan-only",
		ConversionReady: true,
		CreatedAt:       time.Now().UTC(),
	}
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
	return nil
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
