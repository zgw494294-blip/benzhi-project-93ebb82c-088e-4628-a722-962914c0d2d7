package domain

import "fmt"

type ProductionState string

const (
	StateDraft      ProductionState = "DRAFT"
	StateTimelined  ProductionState = "TIMELINED"
	StateWriting    ProductionState = "WRITING"
	StateRehearsing ProductionState = "REHEARSING"
	StateReviewing  ProductionState = "REVIEWING"
	StateRevising   ProductionState = "REVISING"
	StateApproved   ProductionState = "APPROVED"
	StateReleased   ProductionState = "RELEASED"
)

var transitions = map[ProductionState]map[ProductionState]bool{
	StateDraft:      {StateTimelined: true},
	StateTimelined:  {StateWriting: true},
	StateWriting:    {StateRehearsing: true},
	StateRehearsing: {StateReviewing: true},
	StateReviewing:  {StateRevising: true, StateApproved: true},
	StateRevising:   {StateWriting: true, StateRehearsing: true},
	StateApproved:   {StateReleased: true, StateRevising: true},
}

func (p *Production) Transition(next ProductionState) error {
	if p.State == next {
		return nil
	}
	if !transitions[p.State][next] {
		return NewRuleError("state", fmt.Sprintf("状态不能从 %s 变为 %s", p.State, next))
	}
	p.State = next
	return nil
}

func (p Production) EditableMetadata() bool {
	return p.State == StateDraft || p.State == StateTimelined
}

func (p Production) EditableTimeline() bool {
	return p.State == StateDraft || p.State == StateTimelined
}

func (p Production) EditableCues() bool {
	return p.State == StateTimelined || p.State == StateWriting || p.State == StateRevising
}
