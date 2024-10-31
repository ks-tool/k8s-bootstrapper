/*
Copyright © 2024 Alexey Shulutkov <github@shulutkov.ru>

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

package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type HttpError struct {
	status int
	error  error
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("%d: %s", e.status, e.error)
}

func Fetch(dst, url string) error {
	return WithWriter(ToFile(dst, 0644), url)
}

func WithWriter(dst Writer, url string) error {
	return WithContext(context.Background(), dst, url)
}

func WithContext(ctx context.Context, dst Writer, url string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		msg := http.StatusText(resp.StatusCode)
		if b, _ := io.ReadAll(resp.Body); len(b) > 0 {
			msg = string(b)
		}

		return &HttpError{status: resp.StatusCode, error: errors.New(msg)}
	}

	return dst(resp.Body)
}
