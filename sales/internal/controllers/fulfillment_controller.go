package controllers

import (
	"net/http"

	"zeus-sales-service/internal/service"
)

type FulfillmentController struct {
	svc *service.FulfillmentService
}

func NewFulfillmentController(svc *service.FulfillmentService) *FulfillmentController {
	return &FulfillmentController{svc: svc}
}

func (controller *FulfillmentController) HandleQueueStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed), nil)
		return
	}
	status, err := controller.svc.GetQueueStatus(r.Context())
	if err != nil {
		panic(err)
	}
	writeJSON(w, http.StatusOK, status)
}

func (controller *FulfillmentController) HandleProcessQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed), nil)
		return
	}
	manifests, err := controller.svc.ProcessQueue(r.Context())
	if err != nil {
		panic(err)
	}
	writeJSON(w, http.StatusOK, manifests)
}
