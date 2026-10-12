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
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// visitorTTL is how long an idle IP's bucket is kept before being evicted,
// bounding memory use under RateLimit.
const visitorTTL = 3 * time.Minute

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit throttles requests per client IP using a token bucket, so a
// single abusive client can't exhaust the Google Sheets quota shared by
// every visitor to this public, unauthenticated endpoint - now that every
// request reads the sheet live instead of being served from a cache.
func RateLimit(requestsPerSecond float64, burst int, next http.Handler) http.Handler {
	var mu sync.Mutex
	visitors := make(map[string]*visitor)

	go evictStaleVisitors(&mu, visitors)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		mu.Lock()
		v, ok := visitors[ip]
		if !ok {
			v = &visitor{limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), burst)}
			visitors[ip] = v
		}
		v.lastSeen = time.Now()
		allowed := v.limiter.Allow()
		mu.Unlock()

		if !allowed {
			w.Header().Set("Retry-After", "1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"Too many requests."}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func evictStaleVisitors(mu *sync.Mutex, visitors map[string]*visitor) {
	for range time.Tick(visitorTTL) {
		mu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > visitorTTL {
				delete(visitors, ip)
			}
		}
		mu.Unlock()
	}
}

// clientIP returns the request's client IP, preferring a reverse proxy's
// forwarding headers (as set by Choreo's ingress in front of this backend)
// over RemoteAddr, which would otherwise be the proxy's own address.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if ip, _, ok := strings.Cut(fwd, ","); ok {
			return strings.TrimSpace(ip)
		}
		return strings.TrimSpace(fwd)
	}
	if real := r.Header.Get("X-Real-IP"); real != "" {
		return strings.TrimSpace(real)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
