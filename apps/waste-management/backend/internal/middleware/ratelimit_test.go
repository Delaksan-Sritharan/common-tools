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

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimit_AllowsBurstThenThrottles(t *testing.T) {
	calls := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	})
	h := RateLimit(1, 2, next)

	for i := range 2 {
		req := httptest.NewRequest(http.MethodGet, "/api/waste", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d", i, rec.Code, http.StatusOK)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/waste", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
	if calls != 2 {
		t.Errorf("next called %d times, want 2", calls)
	}
}

func TestRateLimit_LimitIsSharedAcrossClients(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	h := RateLimit(1, 1, next)

	reqA := httptest.NewRequest(http.MethodGet, "/api/waste", nil)
	reqA.RemoteAddr = "203.0.113.1:5555"
	recA := httptest.NewRecorder()
	h.ServeHTTP(recA, reqA)
	if recA.Code != http.StatusOK {
		t.Fatalf("client A: status = %d, want %d", recA.Code, http.StatusOK)
	}

	reqB := httptest.NewRequest(http.MethodGet, "/api/waste", nil)
	reqB.RemoteAddr = "203.0.113.2:5555"
	recB := httptest.NewRecorder()
	h.ServeHTTP(recB, reqB)
	if recB.Code != http.StatusTooManyRequests {
		t.Errorf("client B: status = %d, want %d (shares client A's bucket)", recB.Code, http.StatusTooManyRequests)
	}
}
