// Copyright 2018 Josh Komoroske. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE.txt file.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"docker-retag/arguments"
)

const (
	dockerRegistryEnv = "DOCKER_REGISTRY"
	dockerUsernameEnv = "DOCKER_USER"
	dockerPasswordEnv = "DOCKER_PASS"

	defaultRegistry = "https://index.docker.io"

	dockerManifestV2MIME     = "application/vnd.docker.distribution.manifest.v2+json"
	dockerManifestListV2MIME = "application/vnd.docker.distribution.manifest.list.v2+json"
	ociManifestV1MIME        = "application/vnd.oci.image.manifest.v1+json"
	ociIndexV1MIME           = "application/vnd.oci.image.index.v1+json"
)

func main() {
	if err := mainCmd(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "docker-retag: %s\n", err.Error())
		os.Exit(1)
	}
}

func mainCmd(args []string) error {
	// allow empty username/passowrd
	username, _ := os.LookupEnv(dockerUsernameEnv)
	password, _ := os.LookupEnv(dockerPasswordEnv)

	registryUrl, found := os.LookupEnv(dockerRegistryEnv)
	if !found || registryUrl == "" {
		registryUrl = defaultRegistry
	}
	if !strings.HasPrefix(registryUrl, "http://") && !strings.HasPrefix(registryUrl, "https://") {
		// automatically add https scheme if no scheme is present
		registryUrl = "https://" + registryUrl
	}

	prog_args := args[1:]
	if len(prog_args) < 2 {
		return errors.New("Not enough arguments provided, 2 or 3 arguments are required")
	}

	repository, oldTag, newTag, err := arguments.Parse(prog_args)
	if err != nil {
		return err
	}

	reg := NewRegistry(registryUrl, username, password)

	err = reg.ReTag(repository, oldTag, newTag)
	if err != nil {
		return err
	}

	separator := ":"
	if strings.HasPrefix(oldTag, "sha256:") {
		separator = "@"
	}

	fmt.Printf("Retagged %s%s%s as %s:%s\n", repository, separator, oldTag, repository, newTag)

	return nil
}

type HttpError struct {
	Status string
	URL    string
}

func (h HttpError) Error() string {
	return fmt.Sprintf("HTTP %s when accessing %q", h.Status, h.URL)
}

type Registry struct {
	URL    string
	Client *http.Client
}

func NewRegistry(url, username, password string) *Registry {
	authTransport := &basicAuthTransport{
		Wrapped: &tokenAuthTransport{
			Wrapped:  http.DefaultTransport,
			Username: username,
			Password: password,
		},
		URL:      url,
		Username: username,
		Password: password,
	}
	r := Registry{
		url,
		&http.Client{
			Transport: authTransport,
		},
	}

	return &r
}

func (r *Registry) url(pathTemplate string, args ...interface{}) string {
	pathSuffix := fmt.Sprintf(pathTemplate, args...)
	url := fmt.Sprintf("%s%s", r.URL, pathSuffix)
	return url
}

func (r *Registry) ReTag(repo, oldTag, newTag string) error {
	sourceUrl := r.url("/v2/%s/manifests/%s", repo, oldTag)
	sourceReq, err := http.NewRequest("GET", sourceUrl, nil)
	if err != nil {
		return err
	}

	sourceReq.Header.Set("Accept", strings.Join([]string{ociManifestV1MIME, ociIndexV1MIME,dockerManifestListV2MIME, dockerManifestV2MIME}, ", "))
	sourceResp, err := r.Client.Do(sourceReq)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(sourceResp.Body)

	if sourceResp.StatusCode != http.StatusOK {
		return HttpError{sourceResp.Status, sourceUrl}
	}

	receivedMIME := sourceResp.Header.Get("Content-Type")

	manifest, err := io.ReadAll(sourceResp.Body)
	if err != nil {
		return err
	}

	destUrl := r.url("/v2/%s/manifests/%s", repo, newTag)
	destReq, err := http.NewRequest("PUT", destUrl, bytes.NewBuffer(manifest))
	if err != nil {
		return err
	}

	destReq.Header.Set("Content-Type", receivedMIME)
	destResp, err := r.Client.Do(destReq)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(destResp.Body)

	if destResp.StatusCode != http.StatusCreated {
		return HttpError{destResp.Status, destUrl}
	}

	return nil
}
