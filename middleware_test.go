package filepermissions

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type key int

const (
	userKey  key = iota
	errorKey key = iota
)

var (
	ShippingManagerGrants = []PathGrant{
		PathGrant{
			Access: Read,
			Path:   "/managers/",
		},
		PathGrant{
			Access: ReadWrite,
			Path:   "/hr/shipping/",
		},
	}

	ShippingGrants = []PathGrant{
		PathGrant{
			Access: Read,
			Path:   "/hr/shipping/",
		},
	}
)

type mockHelpers struct{}

func (m mockHelpers) GetUserGrants(r *http.Request) ([]PathGrant, error) {
	user := r.Context().Value(userKey).(string)
	if user == "manager" {
		return ShippingManagerGrants, nil
	} else if user == "worker" {
		return ShippingGrants, nil
	}

	return nil, &Error{
		Code:    http.StatusUnauthorized,
		Message: "user not found",
	}
}
func (m mockHelpers) GetRequestedPath(r *http.Request) (string, error) {
	return r.URL.Path, nil
}

type mockErrorHelpers struct{}

func (m mockErrorHelpers) GetUserGrants(r *http.Request) ([]PathGrant, error) {
	err := r.Context().Value(errorKey).(string)
	if err == "1" {
		return nil, &Error{
			Code:    http.StatusUnauthorized,
			Message: "get user grants status unauthorized test error",
		}
	} else if err == "2" {
		return nil, &Error{
			Code:    http.StatusBadRequest,
			Message: "get user grants status bad request test error",
		}
	} else if err == "3" {
		return []PathGrant{}, nil
	}

	return nil, errors.New("default error for get user grants")
}
func (m mockErrorHelpers) GetRequestedPath(r *http.Request) (string, error) {
	return "", errors.New("default error for get requested user path")
}

type defaultHandler struct{}

func (h defaultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func TestHelperErrors(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		errorToRequest string
		expectedCode   int
	}{
		{
			errorToRequest: "1",
			expectedCode:   401,
		},
		{
			errorToRequest: "2",
			expectedCode:   400,
		},
		{
			errorToRequest: "3",
			expectedCode:   500,
		},
		{
			errorToRequest: "0",
			expectedCode:   500,
		},
	}
	hdlr := CreateFilePermissionsMiddleware(mockErrorHelpers{})(defaultHandler{})
	for _, tc := range tests {
		req := httptest.NewRequest("GET", "http://testing.com", nil)
		ctx := req.Context()
		ctx = context.WithValue(ctx, errorKey, tc.errorToRequest)
		req = req.WithContext(ctx)
		res := httptest.NewRecorder()
		hdlr.ServeHTTP(res, req)
		assert.Equal(tc.expectedCode, res.Code)
	}
}

func TestMiddleware(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		method       string
		url          string
		user         string
		expectedCode int
	}{
		{
			method:       "GET",
			url:          "http://testing.com/managers/instructions.pdf",
			user:         "manager",
			expectedCode: 200,
		},
		{
			method:       "GET",
			url:          "http://testing.com/hr/shipping/instructions.pdf",
			user:         "manager",
			expectedCode: 200,
		},
		{
			method:       "POST",
			url:          "http://testing.com/hr/shipping/instructions.pdf",
			user:         "manager",
			expectedCode: 200,
		},
		{
			method:       "PATCH",
			url:          "http://testing.com/hr/shipping/instructions.pdf",
			user:         "manager",
			expectedCode: 200,
		},
		{
			method:       "PUT",
			url:          "http://testing.com/hr/shipping/instructions.pdf",
			user:         "manager",
			expectedCode: 200,
		},
		{
			method:       "DELETE",
			url:          "http://testing.com/hr/shipping/instructions.pdf",
			user:         "manager",
			expectedCode: 200,
		},
		{
			method:       "POST",
			url:          "http://testing.com/managers/instructions.pdf",
			user:         "manager",
			expectedCode: 401,
		},
		{
			method:       "GET",
			url:          "http://testing.com/admin/instructions.pdf",
			user:         "manager",
			expectedCode: 401,
		},
		{
			method:       "GET",
			url:          "http://testing.com/managers/instructions.pdf",
			user:         "worker",
			expectedCode: 401,
		},
		{
			method:       "GET",
			url:          "http://testing.com/hr/shipping/instructions.pdf",
			user:         "worker",
			expectedCode: 200,
		},
		{
			method:       "POST",
			url:          "http://testing.com/hr/shipping/instructions.pdf",
			user:         "worker",
			expectedCode: 401,
		},
	}

	hdlr := CreateFilePermissionsMiddleware(mockHelpers{})(defaultHandler{})
	for _, tc := range tests {
		req := httptest.NewRequest(tc.method, tc.url, nil)
		ctx := req.Context()
		ctx = context.WithValue(ctx, userKey, tc.user)
		req = req.WithContext(ctx)
		res := httptest.NewRecorder()
		hdlr.ServeHTTP(res, req)
		assert.Equal(tc.expectedCode, res.Code)
	}
}
