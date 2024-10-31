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

package utils

import (
	"errors"
	"os/user"
)

func LookupUser(s string) (*user.User, error) {
	u, err := user.Lookup(s)
	var e user.UnknownUserError
	if errors.As(err, &e) {
		return user.LookupId(s)
	}

	return u, err
}

func LookupGroup(s string) (*user.Group, error) {
	g, err := user.LookupGroup(s)
	var e user.UnknownUserError
	if errors.As(err, &e) {
		return user.LookupGroupId(s)
	}

	return g, err
}
