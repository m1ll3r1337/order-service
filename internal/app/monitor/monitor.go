package monitor

type OrderCreateMetric interface {
	Success(totalPrice int64)
	Failed(err error)
	PublishFailed()
}

type OrderStatusMetric interface {
	Inc(status string)
	Dec(status string)
	Transition(oldStatus, newStatus string)
	Set(status string, count float64)
}

type OrderMetrics interface {
	Create() OrderCreateMetric
	Status() OrderStatusMetric
}
