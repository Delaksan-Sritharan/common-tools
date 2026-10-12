// Copyright (c) 2026 WSO2 LLC. (https://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/wso2-open-operations/common-tools/apps/waste-management/backend/internal/sheets"
)

// sheetsFetcher abstracts the Google Sheets read used by WasteHandler.
type sheetsFetcher interface {
	FetchRows(ctx context.Context) ([]sheets.WasteRow, error)
}

// WasteHandler serves waste rows read directly from the configured Google
// Sheet on every request. No caching: the webapp controls its own refresh
// cadence, and each request reflects the sheet's current state.
type WasteHandler struct {
	sheets sheetsFetcher
}

// NewWasteHandler creates a WasteHandler backed by the given Sheets client.
func NewWasteHandler(sheetsClient sheetsFetcher) *WasteHandler {
	return &WasteHandler{sheets: sheetsClient}
}

// wasteBody is the JSON payload for GET /api/waste.
type wasteBody struct {
	Rows     []sheets.WasteRow `json:"rows"`
	CachedAt string            `json:"cachedAt"`
}

// GetWaste handles GET /api/waste.
func (h *WasteHandler) GetWaste(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	rows, err := h.sheets.FetchRows(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch waste data", "err", err)
		writeError(w, http.StatusBadGateway, ErrMsgUpstream)
		return
	}

	writeJSONValue(w, http.StatusOK, wasteBody{Rows: rows, CachedAt: time.Now().UTC().Format(time.RFC3339)})
}
