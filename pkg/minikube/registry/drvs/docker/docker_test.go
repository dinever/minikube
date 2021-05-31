/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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

package docker

import (
	"fmt"
	"strings"
	"testing"

	"k8s.io/minikube/pkg/drivers/kic/oci"
	"k8s.io/minikube/pkg/minikube/driver"
)

type testCase struct {
	version           string
	expectReason      string
	expectError       error
	expectFixContains string
}

func appendVersionVariations(tc []testCase, v []int, reason string, err error) []testCase {
	appendedTc := append(tc, testCase{
		version:      fmt.Sprintf("linux-%02d.%02d", v[0], v[1]),
		expectReason: reason,
		expectError:  err,
	})

	// postfix string for unstable channel or patch. https://docs.docker.com/engine/install/
	patchPostFix := "20180720214833-f61e0f7"

	vs := fmt.Sprintf("%02d.%02d.%d", v[0], v[1], v[2])
	appendedTc = append(appendedTc, []testCase{
		{
			version:      fmt.Sprintf("linux-%s", vs),
			expectReason: reason,
			expectError:  err,
		},
		{
			version:      fmt.Sprintf("linux-%s-%s", vs, patchPostFix),
			expectReason: reason,
			expectError:  err,
		},
	}...,
	)

	return appendedTc
}

func TestCheckDockerVersion(t *testing.T) {
	tc := []testCase{
		{
			version:      "windows-20.0.1",
			expectReason: "PROVIDER_DOCKER_WINDOWS_CONTAINERS",
			expectError:  oci.ErrWindowsContainers,
		},
		{
			version:      fmt.Sprintf("linux-%02d.%02d", minDockerVersion[0], minDockerVersion[1]),
			expectReason: "",
			expectError:  nil,
		},
		{
			version:      fmt.Sprintf("linux-%02d.%02d.%02d", minDockerVersion[0], minDockerVersion[1], minDockerVersion[2]),
			expectReason: "",
			expectError:  nil,
		},
	}

	for i := 0; i < 3; i++ {
		v := make([]int, 3)
		copy(v, minDockerVersion)

		v[i] = minDockerVersion[i] + 1
		tc = appendVersionVariations(tc, v, "", nil)

		v[i] = minDockerVersion[i] - 1
		if v[2] < 0 {
			// skip test if patch version is negative number.
			continue
		}
		tc = appendVersionVariations(tc, v, "PROVIDER_DOCKER_VERSION_LOW", oci.ErrMinDockerVersion)
	}

	tc = append(tc, []testCase{
		{
			// "dev" is set when Docker (Moby) was installed with `make binary && make install`
			version:      "linux-dev",
			expectReason: "",
			expectError:  nil,
			expectFixContains: fmt.Sprintf("Install the official release of %s (Minimum recommended version is %02d.%02d.%d, current version is dev)",
				driver.FullName(driver.Docker), minDockerVersion[0], minDockerVersion[1], minDockerVersion[2]),
		},
		{
			// "library-import" is set when Docker (Moby) was installed with `go build github.com/docker/docker/cmd/dockerd` (unrecommended, but valid)
			version:      "linux-library-import",
			expectReason: "",
			expectError:  nil,
			expectFixContains: fmt.Sprintf("Install the official release of %s (Minimum recommended version is %02d.%02d.%d, current version is library-import)",
				driver.FullName(driver.Docker), minDockerVersion[0], minDockerVersion[1], minDockerVersion[2]),
		},
		{
			// "foo.bar.baz" is a triplet that cannot be parsed as "%02d.%02d.%d"
			version:      "linux-foo.bar.baz",
			expectReason: "",
			expectError:  nil,
			expectFixContains: fmt.Sprintf("Install the official release of %s (Minimum recommended version is %02d.%02d.%d, current version is foo.bar.baz)",
				driver.FullName(driver.Docker), minDockerVersion[0], minDockerVersion[1], minDockerVersion[2]),
		},
	}...)

	for _, c := range tc {
		t.Run("checkDockerVersion test", func(t *testing.T) {
			s := checkDockerVersion(c.version)
			if c.expectReason != s.Reason {
				t.Errorf("Reason %v expected. but got %q. (version string : %s)", c.expectReason, s.Reason, c.version)
			}
			if s.Error != nil {
				if c.expectError != s.Error {
					t.Errorf("Error %v expected. but got %q. (version string : %s)", c.expectError, s.Error, c.version)
				}
			}
			if c.expectFixContains != "" {
				if !strings.Contains(s.Fix, c.expectFixContains) {
					t.Errorf("Error expected Fix to contain %q, but got %q", c.expectFixContains, s.Fix)
				}
			}
		})
	}
}
