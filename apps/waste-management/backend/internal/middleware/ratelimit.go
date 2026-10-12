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

	"golang.org/x/time/rate"
)

// RateLimit throttles all requests through a single shared token bucket.
// The thing being protected is the Google Sheets quota for one OAuth
// account, which every caller draws from together - so the limit must be
// global, not per client: a per-IP limiter still lets one well-behaved
// client (or a client spoofing its IP) exhaust a quota that everyone
// shares, and it can be bypassed by forging forwarding headers. A single
// limiter protects the quota regardless of how many distinct clients call
// in, and can't be evaded by spoofing anything.
func RateLimit(requestsPerSecond float64, burst int, next http.Handler) http.Handler {
	limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), burst)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			w.Header().Set("Retry-After", "1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"Too many requests."}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
