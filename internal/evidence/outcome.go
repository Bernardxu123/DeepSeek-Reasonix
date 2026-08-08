package evidence

import "strings"

// OutcomeSample decomposes one tool round's receipts by outcome: information
// gathered (Exploration), verification attempts run, and verification-command
// state transitions (Objective fail→pass, Regression pass→fail). Counts are
// unit-weighted; policy weighting is an offline concern.
type OutcomeSample struct {
	Round        int
	Exploration  int
	Verification int
	Objective    int
	Regression   int
	Churn        int
	// LegacyGain is the live novelty scorer's verdict on the same receipts, so
	// offline analysis can compare the two policies without replaying.
	LegacyGain int
}

// OutcomeTracker is the shadow counterpart of ProgressTracker: same per-round
// receipts, scored by outcome instead of novelty. It never influences guard
// behavior — samples exist only for trajectory recording and offline analysis.
type OutcomeTracker struct {
	legacy     *ProgressTracker
	round      int
	readPaths  map[string]bool
	commands   map[string]bool
	failures   map[string]bool
	actions    map[string]bool
	verifySeen map[string]bool
	verifyPass map[string]bool
}

func NewOutcomeTracker() *OutcomeTracker {
	return &OutcomeTracker{
		legacy:     NewProgressTracker(),
		readPaths:  map[string]bool{},
		commands:   map[string]bool{},
		failures:   map[string]bool{},
		actions:    map[string]bool{},
		verifySeen: map[string]bool{},
		verifyPass: map[string]bool{},
	}
}

// ScoreRound folds one round's receipts into the tracker and returns the
// round's outcome decomposition.
func (t *OutcomeTracker) ScoreRound(receipts []Receipt) OutcomeSample {
	if t == nil {
		return OutcomeSample{}
	}
	t.round++
	s := OutcomeSample{Round: t.round}
	for _, r := range receipts {
		t.scoreReceipt(r, &s)
	}
	s.LegacyGain = t.legacy.ScoreRound(receipts)
	return s
}

func (t *OutcomeTracker) scoreReceipt(r Receipt, s *OutcomeSample) {
	if command := strings.TrimSpace(r.Command); command != "" {
		t.scoreCommand(command, r, s)
		return
	}
	switch {
	case r.Success && (r.Mutation || r.Write):
		// A mutation is a state transition, not proof of progress: it counts
		// as churn until a verification transition vouches for it.
		s.Churn++
	case r.Success && (r.ToolName == "task" || r.ToolName == "parallel_tasks" || r.ToolName == "fleet"):
		// A delegation return is new information at best — never objective
		// progress on its own.
		s.Exploration++
	case r.Success && (r.StepProof || r.TodoStep != nil || len(r.Todos) > 0):
		// Bookkeeping moves no outcome dimension.
	case r.Success && r.Read && r.OutputBytes > 0 && len(r.Paths) > 0:
		for _, path := range r.Paths {
			if path == "" || t.readPaths[path] {
				continue
			}
			t.readPaths[path] = true
			s.Exploration++
		}
	case r.Success:
		sig := r.ToolName + "\x00" + string(r.Args)
		if !t.actions[sig] {
			t.actions[sig] = true
			s.Exploration++
		}
	}
}

func (t *OutcomeTracker) scoreCommand(command string, r Receipt, s *OutcomeSample) {
	if r.Success && (r.Mutation || r.Write) {
		s.Churn++
	}
	verify := IsDeliveryVerificationCommand(command)
	if verify {
		s.Verification++
		seen, wasPass := t.verifySeen[command], t.verifyPass[command]
		t.verifySeen[command] = true
		t.verifyPass[command] = r.Success
		if seen && r.Success && !wasPass {
			s.Objective++
		}
		if seen && !r.Success && wasPass {
			s.Regression++
		}
	}
	if r.Success {
		if !verify && !t.commands[command] {
			s.Exploration++
		}
		t.commands[command] = true
		return
	}
	if !t.failures[command] {
		t.failures[command] = true
		s.Exploration++
	}
}
