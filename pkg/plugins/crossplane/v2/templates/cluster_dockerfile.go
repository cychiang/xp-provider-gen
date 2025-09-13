/*
Copyright 2025 The Crossplane Authors.

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

package templates

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// ClusterDockerfile creates cluster Dockerfile
func ClusterDockerfile(cfg config.Config) machinery.Template {
	return SimpleFile(cfg, "cluster/images/provider/Dockerfile", clusterDockerfileTemplate)
}

const clusterDockerfileTemplate = `FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY provider /usr/local/bin/
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["provider"]`
