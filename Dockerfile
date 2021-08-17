# Copyright 2017, 2019 the Velero contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM registry.access.redhat.com/ubi8/go-toolset:1.15.13 AS build
ENV GOPATH=$APP_ROOT
COPY . $APP_ROOT/src/velero-plugin-for-encryption
WORKDIR $APP_ROOT/src/velero-plugin-for-encryption
RUN CGO_ENABLED=0 GOOS=linux go build -v -o $APP_ROOT/bin/velero-plugin-for-encryption -mod=mod main.go


FROM registry.access.redhat.com/ubi8-minimal
RUN mkdir /plugins
COPY --from=build /opt/app-root/bin/velero-plugin-for-encryption /plugins/
USER nobody:nobody
ENTRYPOINT ["/bin/bash", "-c", "cp /plugins/* /target/."]