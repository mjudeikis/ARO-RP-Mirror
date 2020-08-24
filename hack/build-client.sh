#!/bin/bash

BUILD_WORKDIR="${PWD}"

function clean() {
  rm ${BUILD_WORKDIR}/.sha256sum

  rm -rf ${BUILD_WORKDIR}/pkg/client
  mkdir ${BUILD_WORKDIR}/pkg/client

  rm -rf ${BUILD_WORKDIR}/python/client
  mkdir -p ${BUILD_WORKDIR}/python/client
}

function checksum() {
  sha256sum swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/stable/"$1"/redhatopenshift.json >> .sha256sum
}

function generate_golang() {
  local API_VERSION=$1

  sudo docker run \
		--rm \
		-v ${BUILD_WORKDIR}/pkg/client:/github.com/Azure/ARO-RP/pkg/client:z \
		-v ${BUILD_WORKDIR}/swagger:/swagger:z \
		azuresdk/autorest \
		--go \
		--license-header=MICROSOFT_APACHE_NO_VERSION \
		--namespace=redhatopenshift \
		--input-file=/swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/stable/"$API_VERSION"/redhatopenshift.json \
		--output-folder=/github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/"$API_VERSION"/redhatopenshift

  sudo chown -R $(id -un):$(id -gn) ${BUILD_WORKDIR}/pkg/client
  sed -i -e 's|azure/aro-rp|Azure/ARO-RP|g' ${BUILD_WORKDIR}/pkg/client/services/redhatopenshift/mgmt/"$API_VERSION"/redhatopenshift/models.go ${BUILD_WORKDIR}/pkg/client/services/redhatopenshift/mgmt/"$API_VERSION"/redhatopenshift/redhatopenshiftapi/interfaces.go
  go run ${BUILD_WORKDIR}/vendor/golang.org/x/tools/cmd/goimports -w -local=github.com/Azure/ARO-RP ${BUILD_WORKDIR}/pkg/client
}

function generate_python() {
  local API_VERSION=$1

  sudo docker run \
		--rm \
		-v ${BUILD_WORKDIR}/python/client:/python/client:z \
		-v ${BUILD_WORKDIR}/swagger:/swagger:z \
		azuresdk/autorest \
		--use=@microsoft.azure/autorest.python@4.0.70 \
		--python \
		--azure-arm \
		--license-header=MICROSOFT_APACHE_NO_VERSION \
		--namespace=azure.mgmt.redhatopenshift.v"${API_VERSION//-/_}" \
		--input-file=/swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/stable/"$API_VERSION"/redhatopenshift.json \
		--output-folder=/python/client

  sudo chown -R $(id -un):$(id -gn) ${BUILD_WORKDIR}/python/client
  rm -rf ${BUILD_WORKDIR}/python/client/azure/mgmt/redhatopenshift/v"${API_VERSION//-/_}"/aio
  >${BUILD_WORKDIR}/python/client/__init__.py
}

clean
for API in "$@"
do
  checksum "${API}"
  generate_golang "${API}"
  generate_python "${API}"
done
