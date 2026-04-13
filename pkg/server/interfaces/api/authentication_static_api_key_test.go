/*
Copyright 2026 The KubeVela Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	apis "github.com/kubevela/velaux/pkg/server/interfaces/api/dto/v1"
)

func TestStaticAPIKeyMatch(t *testing.T) {
	t.Cleanup(func() { SetStaticAPIKeyAuth("", "admin") })

	t.Run("disabled when key empty", func(t *testing.T) {
		SetStaticAPIKeyAuth("", "admin")
		if _, ok := staticAPIKeyMatch("anything"); ok {
			t.Fatal("expected no match")
		}
	})

	t.Run("match", func(t *testing.T) {
		SetStaticAPIKeyAuth("test-secret-key-12345", "operator")
		u, ok := staticAPIKeyMatch("test-secret-key-12345")
		if !ok || u != "operator" {
			t.Fatalf("got ok=%v u=%q", ok, u)
		}
	})

	t.Run("wrong length no match", func(t *testing.T) {
		SetStaticAPIKeyAuth("short", "admin")
		if _, ok := staticAPIKeyMatch("different-len"); ok {
			t.Fatal("expected no match")
		}
	})

	t.Run("same length wrong bytes", func(t *testing.T) {
		SetStaticAPIKeyAuth("aaaa", "admin")
		if _, ok := staticAPIKeyMatch("bbbb"); ok {
			t.Fatal("expected no match")
		}
	})

	t.Run("default user when empty", func(t *testing.T) {
		SetStaticAPIKeyAuth("x", "")
		u, ok := staticAPIKeyMatch("x")
		if !ok || u != "admin" {
			t.Fatalf("got ok=%v u=%q", ok, u)
		}
	})
}

func TestAuthTokenCheck_StaticAPIKey(t *testing.T) {
	t.Cleanup(func() { SetStaticAPIKeyAuth("", "admin") })

	SetStaticAPIKeyAuth("my-static-api-key-32chars!!", "ci-bot")
	req := httptest.NewRequest("GET", "/api/v1/applications", nil)
	req.Header.Set("Authorization", "Bearer my-static-api-key-32chars!!")
	rec := httptest.NewRecorder()

	if !authTokenCheck(req, rec) {
		t.Fatalf("expected success, code=%d", rec.Code)
	}
	user, ok := req.Context().Value(&apis.CtxKeyUser).(string)
	if !ok || user != "ci-bot" {
		t.Fatalf("CtxKeyUser: ok=%v user=%q", ok, user)
	}
	if req.Context().Value(&apis.CtxKeyToken) != nil {
		t.Fatal("CtxKeyToken should not be set for static API key")
	}
}

func TestAuthTokenCheck_StaticAPIKeyWrongKeyFallsBackToJWT(t *testing.T) {
	t.Cleanup(func() { SetStaticAPIKeyAuth("", "admin") })

	key := strings.Repeat("a", 32)
	wrong := strings.Repeat("b", 32)
	SetStaticAPIKeyAuth(key, "admin")
	req := httptest.NewRequest("GET", "/api/v1/applications", nil)
	req.Header.Set("Authorization", "Bearer "+wrong)
	rec := httptest.NewRecorder()

	if authTokenCheck(req, rec) {
		t.Fatal("expected failure when token does not match static key and JWT invalid")
	}
}
