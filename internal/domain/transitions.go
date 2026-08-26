package domain

import (
	"errors"
	"fmt"
)

type Action string

const (
	ActionAcknowledge Action = "acknowledge"
	ActionInvestigate Action = "investigate"
	ActionMitigate    Action = "mitigate"
	ActionResolve     Action = "resolve"
	ActionReopen      Action = "reopen"
	ActionArchive     Action = "archive"
	ActionReject      Action = "reject"
)

var ErrInvalidTransition = errors.New("invalid status transition")

func NextStatus(current Status, action Action) (Status, error) {
	switch current {
	case StatusNew:
		switch action {
		case ActionAcknowledge:
			return StatusAcknowledged, nil
		case ActionReject:
			return StatusRejected, nil
		}
	case StatusAcknowledged:
		switch action {
		case ActionInvestigate:
			return StatusInvestigating, nil
		case ActionReject:
			return StatusRejected, nil
		}
	case StatusInvestigating:
		switch action {
		case ActionMitigate:
			return StatusMitigated, nil
		case ActionResolve:
			return StatusResolved, nil
		case ActionArchive:
			return StatusArchived, nil
		}
	case StatusMitigated:
		switch action {
		case ActionResolve:
			return StatusResolved, nil
		case ActionReopen:
			return StatusInvestigating, nil
		}
	case StatusResolved:
		switch action {
		case ActionArchive:
			return StatusArchived, nil
		case ActionReopen:
			return StatusInvestigating, nil
		}
	case StatusRejected:
		if action == ActionArchive {
			return StatusArchived, nil
		}
	case StatusArchived:
		if action == ActionReopen {
			return StatusInvestigating, nil
		}
	}
	return current, fmt.Errorf("%w: %s cannot %s", ErrInvalidTransition, current, action)
}

func AllowedActions(status Status) []Action {
	switch status {
	case StatusNew:
		return []Action{ActionAcknowledge, ActionReject}
	case StatusAcknowledged:
		return []Action{ActionInvestigate, ActionReject}
	case StatusInvestigating:
		return []Action{ActionMitigate, ActionResolve}
	case StatusMitigated:
		return []Action{ActionResolve, ActionReopen}
	case StatusResolved:
		return []Action{ActionArchive, ActionReopen}
	case StatusRejected:
		return []Action{ActionArchive}
	case StatusArchived:
		return []Action{ActionReopen}
	default:
		return nil
	}
}

func TransitionRole(action Action) string {
	switch action {
	case ActionAcknowledge, ActionInvestigate, ActionMitigate:
		return "operator"
	case ActionResolve, ActionReopen, ActionReject, ActionArchive:
		return "supervisor"
	default:
		return ""
	}
}

func TransitionEventType(action Action) string {
	switch action {
	case ActionAcknowledge:
		return "alert.acknowledged"
	case ActionInvestigate:
		return "alert.investigation_started"
	case ActionMitigate:
		return "alert.mitigated"
	case ActionResolve:
		return "alert.resolved"
	case ActionReopen:
		return "alert.reopened"
	case ActionArchive:
		return "alert.archived"
	case ActionReject:
		return "alert.rejected"
	default:
		return "alert.changed"
	}
}
