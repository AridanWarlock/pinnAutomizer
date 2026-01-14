package schema

const EventsTable = "events"

const (
	EventsTableColumnID        = "id"
	EventsTableColumnTopic     = "topic"
	EventsTableColumnData      = "data"
	EventsTableColumnCreatedAt = "created_at"
)

var EventsTableColumns = []string{
	EventsTableColumnID,
	EventsTableColumnTopic,
	EventsTableColumnData,
	EventsTableColumnCreatedAt,
}
