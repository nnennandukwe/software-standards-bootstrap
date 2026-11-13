package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/render"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

type validationResponse struct {
	SchemaVersion   int                       `json:"schema_version"`
	Valid           bool                      `json:"valid"`
	BaselineCommit  string                    `json:"baseline_commit,omitempty"`
	ArtifactCount   int                       `json:"artifact_count"`
	RuleCount       int                       `json:"rule_count"`
	RecipeCount     int                       `json:"verification_recipe_count"`
	SkillCount      int                       `json:"skill_count"`
	AutomationCount int                       `json:"automation_proposal_count"`
	Diagnostics     []rulepack.Diagnostic     `json:"diagnostics"`
	Projection      *render.ProjectionMetrics `json:"projection,omitempty"`
	Warnings        []render.SizeWarning      `json:"warnings"`
	Pack            *rulepack.Pack            `json:"pack,omitempty"`
}

func runValidate(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	format := flags.String("format", "text", "output format")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, validateHelp)
			return 0
		}
		fmt.Fprintf(stderr, "error: %s\nnext: ssb validate --repo PATH [--format text|json]\n", cleanFlagError(err))
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "error: validate does not accept positional arguments")
		fmt.Fprintln(stderr, "next: ssb validate --repo PATH [--format text|json]")
		return 2
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintln(stderr, "error: --format must be text or json")
		fmt.Fprintln(stderr, "next: ssb validate --repo PATH --format text")
		return 2
	}

	repo, err := workspace.Open(ctx, *repoPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\nnext: ssb validate --repo <repository-path>\n", err)
		if errors.Is(err, workspace.ErrPrecondition) {
			return 2
		}
		return commandErrorExitCode(err, 3)
	}
	pack, diagnostics, err := rulepack.Validate(ctx, repo)
	if err != nil {
		fmt.Fprintf(stderr, "error: validation failed unexpectedly: %s\n", err)
		return commandErrorExitCode(err, 3)
	}
	response := validationResponse{
		SchemaVersion:   3,
		Valid:           len(diagnostics) == 0,
		BaselineCommit:  pack.BaselineCommit,
		ArtifactCount:   len(pack.Manifest.Artifacts),
		RuleCount:       len(pack.Rules),
		RecipeCount:     len(pack.Recipes),
		SkillCount:      len(pack.Skills),
		AutomationCount: len(pack.Automations),
		Diagnostics:     diagnostics,
	}
	if response.Diagnostics == nil {
		response.Diagnostics = make([]rulepack.Diagnostic, 0)
	}
	response.Warnings = make([]render.SizeWarning, 0)
	if response.Valid {
		plan, err := render.Build(pack)
		if err != nil {
			fmt.Fprintf(stderr, "error: build projection diagnostics: %s\n", err)
			return 3
		}
		response.Projection = &plan.Metrics
		response.Warnings = plan.Warnings
		response.Pack = &pack
	}

	if *format == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(response); err != nil {
			fmt.Fprintf(stderr, "error: write JSON validation result: %s\n", err)
			return 3
		}
	} else if response.Valid {
		fmt.Fprintf(
			stdout,
			"Actionable pack valid: %d artifact(s) \u2014 %d semantic rule(s), %d verification recipe(s), %d Agent Skill(s), %d automation proposal(s); baseline %s\n",
			response.ArtifactCount,
			response.RuleCount,
			response.RecipeCount,
			response.SkillCount,
			response.AutomationCount,
			response.BaselineCommit,
		)
		writeProjectionMeasurements(stdout, *response.Projection)
		writeProjectionWarnings(stderr, response.Warnings)
		fmt.Fprintln(stdout, "Next: review and edit source files, then run ssb render --repo PATH.")
	} else {
		writeValidationDiagnostics(stderr, diagnostics)
	}
	if !response.Valid {
		return 1
	}
	return 0
}
