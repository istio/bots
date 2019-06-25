// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"fmt"
	"net/http"

	"istio.io/pkg/log"
)

// Holds a Go error with an HTTP status code.
type HTTPError struct {
	error
	StatusCode int
}

func HTTPErrorf(httpStatusCode int, format string, a ...interface{}) error {
	return HTTPError{
		fmt.Errorf("%s (%d): %s", http.StatusText(httpStatusCode), httpStatusCode, fmt.Sprintf(format, a...)),
		httpStatusCode,
	}
}

func RenderError(w http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	if httpErr, ok := err.(HTTPError); ok {
		statusCode = httpErr.StatusCode
	}

	msg := fmt.Sprintf("%v", err)
	http.Error(w, msg, statusCode)

	log.Errorf("Returning HTTP error to client: %v", msg)
}
