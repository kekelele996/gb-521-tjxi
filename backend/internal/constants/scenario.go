package constants

type ScenarioStatus string

const (
	ScenarioStatusDraft         ScenarioStatus = "draft"
	ScenarioStatusPendingReview ScenarioStatus = "pending_review"
	ScenarioStatusApproved      ScenarioStatus = "approved"
	ScenarioStatusArchived      ScenarioStatus = "archived"
)

type Role string

const (
	RoleEngineer Role = "engineer"
	RoleReviewer Role = "reviewer"
	RoleAdmin    Role = "admin"
)

func ValidScenarioStatus(value string) bool {
	switch ScenarioStatus(value) {
	case ScenarioStatusDraft, ScenarioStatusPendingReview, ScenarioStatusApproved, ScenarioStatusArchived:
		return true
	default:
		return false
	}
}

func CanTransitionScenario(from, to ScenarioStatus) bool {
	switch from {
	case ScenarioStatusDraft:
		return to == ScenarioStatusPendingReview
	case ScenarioStatusPendingReview:
		return to == ScenarioStatusApproved || to == ScenarioStatusDraft
	case ScenarioStatusApproved:
		return to == ScenarioStatusArchived
	default:
		return false
	}
}

type ScenarioVersionKind string

const (
	ScenarioVersionCreated   ScenarioVersionKind = "created"
	ScenarioVersionSubmitted ScenarioVersionKind = "submitted"
	ScenarioVersionApproved  ScenarioVersionKind = "approved"
	ScenarioVersionRejected  ScenarioVersionKind = "rejected"
	ScenarioVersionArchived  ScenarioVersionKind = "archived"
	ScenarioVersionEdited    ScenarioVersionKind = "edited"
	ScenarioVersionRestored  ScenarioVersionKind = "restored"
	ScenarioVersionBackfill  ScenarioVersionKind = "backfill"
)

// TransitionVersionKind 把状态迁移目标映射到留痕类别，驳回目标状态为 draft。
func TransitionVersionKind(to ScenarioStatus) ScenarioVersionKind {
	switch to {
	case ScenarioStatusPendingReview:
		return ScenarioVersionSubmitted
	case ScenarioStatusApproved:
		return ScenarioVersionApproved
	case ScenarioStatusArchived:
		return ScenarioVersionArchived
	default:
		return ScenarioVersionRejected
	}
}

func ValidRole(value string) bool {
	switch Role(value) {
	case RoleEngineer, RoleReviewer, RoleAdmin:
		return true
	default:
		return false
	}
}
