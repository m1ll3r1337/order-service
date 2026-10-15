package mmetric

type orderStatus string

const (
	orderStatusUnknown  orderStatus = "unknown"
	orderStatusPending  orderStatus = "pending"
	orderStatusFinished orderStatus = "finished"
	orderStatusCanceled orderStatus = "canceled"
)

var allowedStatuses = map[orderStatus]struct{}{
	orderStatusPending:  {},
	orderStatusFinished: {},
	orderStatusCanceled: {},
}

func normalizeStatus(status string) orderStatus {
	s := orderStatus(status)
	if _, ok := allowedStatuses[s]; !ok {
		return orderStatusUnknown
	}

	return s
}
