package workflow

import (
	"testing"

	"devswarm/internal/types"
)

func TestRenderPrompt(t *testing.T) {
	e := &Engine{}

	out, err := e.renderPrompt("branch={{.Branch}} diff={{.Diff}}", map[string]string{
		"Branch": "devswarm/run/x",
		"Diff":   "abc",
	})
	if err != nil {
		t.Fatalf("renderPrompt returned error: %v", err)
	}
	if out != "branch=devswarm/run/x diff=abc" {
		t.Fatalf("renderPrompt output = %q", out)
	}
}

func TestRenderPrompt_InvalidTemplate(t *testing.T) {
	e := &Engine{}
	_, err := e.renderPrompt("{{.Branch", map[string]string{"Branch": "x"})
	if err == nil {
		t.Fatalf("expected error for invalid template")
	}
}

func TestResolveBaseBranch(t *testing.T) {
	e := &Engine{}

	run := &Run{BaseBranch: "main"}
	stepDef := &types.PipelineStep{DependsOn: nil}
	base, err := e.resolveBaseBranch(run, stepDef)
	if err != nil {
		t.Fatalf("resolveBaseBranch failed: %v", err)
	}
	if base != "main" {
		t.Errorf("expected base 'main', got %q", base)
	}
}

func TestResolveBaseBranch_WithDepends(t *testing.T) {
	e := &Engine{}
	run := &Run{
		BaseBranch: "main",
		Steps: []StepStatus{
			{ID: "ut", ShadowBranch: "devswarm/run/ut"},
		},
	}

	stepDef := &types.PipelineStep{DependsOn: []string{"ut"}}
	base, err := e.resolveBaseBranch(run, stepDef)
	if err != nil {
		t.Fatalf("resolveBaseBranch failed: %v", err)
	}
	if base != "devswarm/run/ut" {
		t.Errorf("expected base 'devswarm/run/ut', got %q", base)
	}
}
