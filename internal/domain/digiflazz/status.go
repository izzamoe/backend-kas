package digiflazz

type OrderStatus string

const (
	OrderStatusInquiry    OrderStatus = "inquiry"
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusSuccess    OrderStatus = "success"
	OrderStatusFailed     OrderStatus = "failed"
	OrderStatusCancelled  OrderStatus = "canceled"
)

func (s OrderStatus) String() string { return string(s) }

type EventType string

const (
	EventTypeTopup   EventType = "topup"
	EventTypeInquiry EventType = "inquiry"
	EventTypePay     EventType = "pay"
	EventTypeStatus  EventType = "status"
	EventTypeDeposit EventType = "deposit"
	EventTypeWebhook EventType = "webhook"
	EventTypeError   EventType = "error"
)

func (e EventType) String() string { return string(e) }
