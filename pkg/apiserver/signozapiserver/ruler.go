package signozapiserver

import (
	"net/http"

	"github.com/SigNoz/signoz/pkg/http/handler"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/gorilla/mux"
)

func (provider *provider) addRulerRoutes(router *mux.Router) error {
	if err := router.Handle("/api/v2/rules", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.ListRules), handler.OpenAPIDef{
		ID:                  "ListRules",
		Tags:                []string{"rules"},
		Summary:             "List alert rules",
		Description:         "This endpoint lists all alert rules with their current evaluation state",
		Response:            make([]*ruletypes.Rule, 0),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules/{id}", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.GetRuleByID), handler.OpenAPIDef{
		ID:                  "GetRuleByID",
		Tags:                []string{"rules"},
		Summary:             "Get alert rule by ID",
		Description:         "This endpoint returns an alert rule by ID",
		Response:            new(ruletypes.Rule),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusNotFound},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules", handler.New(provider.authZ.EditAccess(provider.rulerHandler.CreateRule), handler.OpenAPIDef{
		ID:                  "CreateRule",
		Tags:                []string{"rules"},
		Summary:             "Create alert rule",
		Description:         "This endpoint creates a new alert rule",
		Request:             new(ruletypes.PostableRule),
		RequestContentType:  "application/json",
		RequestExamples:     postableRuleExamples(),
		Response:            new(ruletypes.Rule),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusCreated,
		ErrorStatusCodes:    []int{http.StatusBadRequest},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules/{id}", handler.New(provider.authZ.EditAccess(provider.rulerHandler.UpdateRuleByID), handler.OpenAPIDef{
		ID:                 "UpdateRuleByID",
		Tags:               []string{"rules"},
		Summary:            "Update alert rule",
		Description:        "This endpoint updates an alert rule by ID",
		Request:            new(ruletypes.PostableRule),
		RequestContentType: "application/json",
		RequestExamples:    postableRuleExamples(),
		SuccessStatusCode:  http.StatusNoContent,
		ErrorStatusCodes:   []int{http.StatusBadRequest, http.StatusNotFound},
		SecuritySchemes:    newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPut).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules/{id}", handler.New(provider.authZ.EditAccess(provider.rulerHandler.DeleteRuleByID), handler.OpenAPIDef{
		ID:                "DeleteRuleByID",
		Tags:              []string{"rules"},
		Summary:           "Delete alert rule",
		Description:       "This endpoint deletes an alert rule by ID",
		SuccessStatusCode: http.StatusNoContent,
		ErrorStatusCodes:  []int{http.StatusNotFound},
		SecuritySchemes:   newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodDelete).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules/{id}", handler.New(provider.authZ.EditAccess(provider.rulerHandler.PatchRuleByID), handler.OpenAPIDef{
		ID:                  "PatchRuleByID",
		Tags:                []string{"rules"},
		Summary:             "Patch alert rule",
		Description:         "This endpoint applies a partial update to an alert rule by ID",
		Request:             new(ruletypes.PostableRule),
		RequestContentType:  "application/json",
		RequestExamples:     postableRuleExamples(),
		Response:            new(ruletypes.Rule),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusNotFound},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPatch).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules/test", handler.New(provider.authZ.EditAccess(provider.rulerHandler.TestRule), handler.OpenAPIDef{
		ID:                  "TestRule",
		Tags:                []string{"rules"},
		Summary:             "Test alert rule",
		Description:         "This endpoint fires a test notification for the given rule definition",
		Request:             new(ruletypes.PostableRule),
		RequestContentType:  "application/json",
		RequestExamples:     postableRuleExamples(),
		Response:            new(ruletypes.GettableTestRule),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules/notification_template/preview", handler.New(provider.authZ.EditAccess(provider.rulerHandler.PreviewNotificationTemplate), handler.OpenAPIDef{
		ID:                  "PreviewNotificationTemplate",
		Tags:                []string{"rules"},
		Summary:             "Preview alert notification template",
		Description:         "This endpoint renders a notification message template against the rule labels and annotations without sending a notification",
		Request:             new(ruletypes.PreviewNotificationTemplateRequest),
		RequestContentType:  "application/json",
		Response:            new(ruletypes.PreviewNotificationTemplateResponse),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules/sop/preview", handler.New(provider.authZ.EditAccess(provider.rulerHandler.PreviewSOP), handler.OpenAPIDef{
		ID:                  "PreviewSOP",
		Tags:                []string{"rules"},
		Summary:             "Preview alert SOP binding",
		Description:         "This endpoint previews SOP source, binding, search, preview metadata, and service-account/API auth boundary guidance for an alert rule without fetching an external SOP document",
		Request:             new(ruletypes.PreviewSOPRequest),
		RequestContentType:  "application/json",
		Response:            new(ruletypes.PreviewSOPResponse),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules/sop/pilot/managed_markdown/fetch", handler.New(provider.authZ.EditAccess(provider.rulerHandler.FetchPilotManagedMarkdownSOP), handler.OpenAPIDef{
		ID:                  "FetchPilotManagedMarkdownSOP",
		Tags:                []string{"rules"},
		Summary:             "Fetch a managed Markdown SOP for pilot validation",
		Description:         "This pilot endpoint fetches an inline managed Markdown SOP source only after the audit-required server-side contract is accepted; it does not persist sources, call external connectors, or accept browser credentials",
		Request:             new(ruletypes.PilotManagedMarkdownSOPFetchRequest),
		RequestContentType:  "application/json",
		Response:            new(ruletypes.PilotSOPFetchResponse),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v1/downtime_schedules", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.ListDowntimeSchedules), handler.OpenAPIDef{
		ID:                  "ListDowntimeSchedules",
		Tags:                []string{"downtimeschedules"},
		Summary:             "List downtime schedules",
		Description:         "This endpoint lists all planned maintenance / downtime schedules",
		RequestQuery:        new(ruletypes.ListPlannedMaintenanceParams),
		Response:            make([]*ruletypes.PlannedMaintenance, 0),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v1/downtime_schedules/{id}", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.GetDowntimeScheduleByID), handler.OpenAPIDef{
		ID:                  "GetDowntimeScheduleByID",
		Tags:                []string{"downtimeschedules"},
		Summary:             "Get downtime schedule by ID",
		Description:         "This endpoint returns a downtime schedule by ID",
		Response:            new(ruletypes.PlannedMaintenance),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusNotFound},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v1/downtime_schedules", handler.New(provider.authZ.EditAccess(provider.rulerHandler.CreateDowntimeSchedule), handler.OpenAPIDef{
		ID:                  "CreateDowntimeSchedule",
		Tags:                []string{"downtimeschedules"},
		Summary:             "Create downtime schedule",
		Description:         "This endpoint creates a new planned maintenance / downtime schedule",
		Request:             new(ruletypes.PostablePlannedMaintenance),
		RequestContentType:  "application/json",
		Response:            new(ruletypes.PlannedMaintenance),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusCreated,
		ErrorStatusCodes:    []int{http.StatusBadRequest},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v1/downtime_schedules/{id}", handler.New(provider.authZ.EditAccess(provider.rulerHandler.UpdateDowntimeScheduleByID), handler.OpenAPIDef{
		ID:                 "UpdateDowntimeScheduleByID",
		Tags:               []string{"downtimeschedules"},
		Summary:            "Update downtime schedule",
		Description:        "This endpoint updates a downtime schedule by ID",
		Request:            new(ruletypes.PostablePlannedMaintenance),
		RequestContentType: "application/json",
		SuccessStatusCode:  http.StatusNoContent,
		ErrorStatusCodes:   []int{http.StatusBadRequest, http.StatusNotFound},
		SecuritySchemes:    newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPut).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v1/downtime_schedules/{id}", handler.New(provider.authZ.EditAccess(provider.rulerHandler.DeleteDowntimeScheduleByID), handler.OpenAPIDef{
		ID:                "DeleteDowntimeScheduleByID",
		Tags:              []string{"downtimeschedules"},
		Summary:           "Delete downtime schedule",
		Description:       "This endpoint deletes a downtime schedule by ID",
		SuccessStatusCode: http.StatusNoContent,
		ErrorStatusCodes:  []int{http.StatusNotFound},
		SecuritySchemes:   newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodDelete).GetError(); err != nil {
		return err
	}

	return nil
}
