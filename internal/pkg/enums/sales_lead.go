package enums

type SalesLeadStatus string

const (
	SalesLeadStatusNew       SalesLeadStatus = "new"
	SalesLeadStatusFollowing SalesLeadStatus = "following"
	SalesLeadStatusVisited   SalesLeadStatus = "visited"
	SalesLeadStatusConverted SalesLeadStatus = "converted"
	SalesLeadStatusInvalid   SalesLeadStatus = "invalid"
	SalesLeadStatusClosed    SalesLeadStatus = "closed"
)

type SalesLeadIntent string

const (
	SalesLeadIntentUnknown SalesLeadIntent = "unknown"
	SalesLeadIntentLow     SalesLeadIntent = "low"
	SalesLeadIntentMedium  SalesLeadIntent = "medium"
	SalesLeadIntentHigh    SalesLeadIntent = "high"
)

type SalesLeadStage string

const (
	SalesLeadStageUnknown     SalesLeadStage = "unknown"
	SalesLeadStageConsulting  SalesLeadStage = "consulting"
	SalesLeadStageComparing   SalesLeadStage = "comparing"
	SalesLeadStageAppointment SalesLeadStage = "appointment"
	SalesLeadStageReadyToBuy  SalesLeadStage = "ready_to_buy"
	SalesLeadStageAfterSales  SalesLeadStage = "after_sales"
)

func IsValidSalesLeadStatus(value string) bool {
	switch SalesLeadStatus(value) {
	case SalesLeadStatusNew, SalesLeadStatusFollowing, SalesLeadStatusVisited, SalesLeadStatusConverted, SalesLeadStatusInvalid, SalesLeadStatusClosed:
		return true
	default:
		return false
	}
}
