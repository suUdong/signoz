package ruler

import "net/http"

type Handler interface {
	ListRules(http.ResponseWriter, *http.Request)
	GetRuleByID(http.ResponseWriter, *http.Request)
	CreateRule(http.ResponseWriter, *http.Request)
	UpdateRuleByID(http.ResponseWriter, *http.Request)
	DeleteRuleByID(http.ResponseWriter, *http.Request)
	PatchRuleByID(http.ResponseWriter, *http.Request)
	TestRule(http.ResponseWriter, *http.Request)
	PreviewNotificationTemplate(http.ResponseWriter, *http.Request)
	PreviewSOP(http.ResponseWriter, *http.Request)
	FetchPilotManagedMarkdownSOP(http.ResponseWriter, *http.Request)
	ListPilotSOPSources(http.ResponseWriter, *http.Request)
	GetPilotSOPSourceHealth(http.ResponseWriter, *http.Request)
	CreateSOPDocument(http.ResponseWriter, *http.Request)
	ListSOPDocuments(http.ResponseWriter, *http.Request)
	GetSOPDocument(http.ResponseWriter, *http.Request)
	FetchSOPDocumentVersion(http.ResponseWriter, *http.Request)
	PreviewSOPDocumentBinding(http.ResponseWriter, *http.Request)
	PreviewAIStrategy(http.ResponseWriter, *http.Request)
	GetLatestAIStrategyHistory(http.ResponseWriter, *http.Request)

	ListDowntimeSchedules(http.ResponseWriter, *http.Request)
	GetDowntimeScheduleByID(http.ResponseWriter, *http.Request)
	CreateDowntimeSchedule(http.ResponseWriter, *http.Request)
	UpdateDowntimeScheduleByID(http.ResponseWriter, *http.Request)
	DeleteDowntimeScheduleByID(http.ResponseWriter, *http.Request)
}
