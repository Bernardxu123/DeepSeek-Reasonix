package evidence

import (
	"encoding/json"
	"testing"
)

func TestOutcomeTrackerSeparatesExplorationFromObjective(t *testing.T) {
	tr := NewOutcomeTracker()

	read := readReceipt("a.go")
	read.OutputBytes = 10
	s := tr.ScoreRound([]Receipt{read})
	if s.Exploration != 1 || s.Objective != 0 {
		t.Fatalf("new read = %+v, want exploration 1 objective 0", s)
	}

	// First verification failure: an attempt plus localization, no objective.
	s = tr.ScoreRound([]Receipt{bashReceipt("go test ./x", false)})
	if s.Verification != 1 || s.Exploration != 1 || s.Objective != 0 {
		t.Fatalf("first failing verify = %+v, want verification 1 exploration 1", s)
	}

	// A mutation is churn, not objective progress — the legacy scorer disagrees.
	write := ReceiptFromToolCall("write_file", json.RawMessage(`{"path":"b.go","content":"x"}`), true, false)
	s = tr.ScoreRound([]Receipt{write})
	if s.Churn != 1 || s.Objective != 0 || s.Exploration != 0 {
		t.Fatalf("mutation = %+v, want churn 1 only", s)
	}
	if s.LegacyGain != gainMutation {
		t.Fatalf("mutation legacy gain = %d, want %d", s.LegacyGain, gainMutation)
	}

	// The failing verification turning green is the objective transition.
	s = tr.ScoreRound([]Receipt{bashReceipt("go test ./x", true)})
	if s.Objective != 1 || s.Verification != 1 || s.Regression != 0 {
		t.Fatalf("fail→pass verify = %+v, want objective 1", s)
	}

	// The same verification breaking again is a regression.
	s = tr.ScoreRound([]Receipt{bashReceipt("go test ./x", false)})
	if s.Regression != 1 || s.Objective != 0 {
		t.Fatalf("pass→fail verify = %+v, want regression 1", s)
	}
}

func TestOutcomeTrackerDelegationAndRepeatsAreExplorationAtBest(t *testing.T) {
	tr := NewOutcomeTracker()

	task := ReceiptFromToolCall("task", json.RawMessage(`{"prompt":"dig"}`), true, false)
	s := tr.ScoreRound([]Receipt{task})
	if s.Exploration != 1 || s.Objective != 0 {
		t.Fatalf("delegation = %+v, want exploration 1 objective 0", s)
	}

	// A first passing verification run establishes a baseline, not progress.
	s = tr.ScoreRound([]Receipt{bashReceipt("go vet ./...", true)})
	if s.Verification != 1 || s.Objective != 0 || s.Exploration != 0 {
		t.Fatalf("baseline verify = %+v, want verification 1 only", s)
	}
	s = tr.ScoreRound([]Receipt{bashReceipt("go vet ./...", true)})
	if s.Verification != 1 || s.Objective != 0 {
		t.Fatalf("repeated passing verify = %+v, want no objective", s)
	}

	// A repeat delegation still returned content the host cannot judge — it
	// stays exploration and can never move the objective dimension.
	repeat := ReceiptFromToolCall("task", json.RawMessage(`{"prompt":"dig"}`), true, false)
	s = tr.ScoreRound([]Receipt{repeat})
	if s.Exploration != 1 || s.Objective != 0 {
		t.Fatalf("repeated delegation = %+v, want exploration 1 objective 0", s)
	}

	var nilTracker *OutcomeTracker
	if got := nilTracker.ScoreRound([]Receipt{task}); got != (OutcomeSample{}) {
		t.Fatalf("nil tracker sample = %+v, want zero", got)
	}
}
