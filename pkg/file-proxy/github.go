/*
 Copyright (c) 2024 Alexey Shulutkov <github@shulutkov.ru>

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this File except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package fileproxy

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/github"
)

type Github struct {
	Owner    string
	Repo     string
	File     func(string) string
	HashFile func(string) string
}

func (gh Github) getUrl(tag, filename string) (string, error) {
	cl := github.NewClient(http.DefaultClient)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rel, err := wrapper(cl.Repositories.GetReleaseByTag(ctx, gh.Owner, gh.Repo, tag))
	if err != nil {
		return "", err
	}

	for _, asset := range rel.Assets {
		if *asset.Name == filename {
			return *asset.BrowserDownloadURL, nil
		}
	}

	return "", ErrFileNotFound
}

func (gh Github) FileURL(_, tag string) (string, error) {
	return gh.getUrl(tag, gh.File(tag))
}

func (gh Github) HashFileURL(_, tag string) (string, error) {
	return gh.getUrl(tag, gh.HashFile(tag))
}

func (gh Github) LastTag() (string, error) {
	cl := github.NewClient(http.DefaultClient)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rel, err := wrapper(cl.Repositories.GetLatestRelease(ctx, gh.Owner, gh.Repo))
	if err != nil {
		return "", err
	}

	return *rel.TagName, nil
}

func wrapper(rel *github.RepositoryRelease, resp *github.Response, err error) (*github.RepositoryRelease, error) {
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("failed to fetch release: %s: %s", resp, err)
		}

		return nil, err
	}

	return rel, nil
}
