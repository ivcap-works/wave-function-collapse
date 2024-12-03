SERVICE_NAME=railroad-wave-collapse
SERVICE_TITLE=Railroad Tracks via Wave Function Collapse algorithm

SERVICE_FILE=service.py
SERVICE_ID:=$(shell python3 -c 'import uuid; print(uuid.uuid5(uuid.NAMESPACE_DNS, \
        "${PROVIDER_NAME}" + "${SERVICE_CONTAINER_NAME}"));')
SERVICE_URN:=urn:ivcap:service:${SERVICE_ID}

GIT_COMMIT := $(shell git rev-parse --short HEAD)
GIT_TAG := $(shell git describe --abbrev=0 --tags ${TAG_COMMIT} 2>/dev/null || true)
VERSION="${GIT_TAG}|${GIT_COMMIT}|$(shell date -Iminutes)"

DOCKER_USER="$(shell id -u):$(shell id -g)"
DOCKER_DOMAIN=$(shell echo ${PROVIDER_NAME} | sed -E 's/[-:]/_/g')
DOCKER_NAME=$(shell echo ${SERVICE_NAME} | sed -E 's/-/_/g')
DOCKER_VERSION=${GIT_COMMIT}
DOCKER_TAG=${DOCKER_NAME}:${DOCKER_VERSION}
DOCKER_TAG_LOCAL=${DOCKER_NAME}:latest

PROJECT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TARGET_PLATFORM := linux/amd64
PORT = 8088
run:
	mkdir -p ./output
	go run main.go --run-once

help:
	go run main.go --help

serve:
	go run main.go --port ${PORT}

send-request:
	@mkdir -p ./output
	$(eval file:=${PROJECT_DIR}/output/$(shell date -Iseconds).png)
	@curl -X POST \
		-H "Content-Type: application/json" \
		-d @sample_request.json \
		-o ${file} \
		--silent \
		-w '\nStatus: %{response_code}\n%{header_json}\n' \
		http://localhost:8088
	@echo "result stored in ${file}"

send-ivcap-request:
	@mkdir -p ./output
	$(eval file:=${PROJECT_DIR}/output/$(shell date -Iseconds).png)
	curl -X POST \
		-H "Content-Type: application/json" \
		-H "Authorization: Bearer $(shell ivcap context get access-token)" \
		-d @sample_request.json \
		-o ${file} \
		-w '\nStatus: %{response_code}\n%{header_json}\n' \
		http://localhost:8088/1/services/${SERVICE_ID}/jobs
	@echo "result stored in ${file}"


docker-build:
	@echo "Building docker image ${DOCKER_NAME}"
	docker build \
		-t ${DOCKER_TAG_LOCAL} \
		--platform=${TARGET_PLATFORM} \
		--build-arg VERSION=${VERSION} \
		-f ${PROJECT_DIR}/Dockerfile \
		${PROJECT_DIR} ${DOCKER_BILD_ARGS}
	@echo "\nFinished building docker image ${DOCKER_NAME}\n"

docker-run:
	@mkdir -p ./output
	docker run -it --rm \
		-v./output:/output \
		--user ${DOCKER_USER} \
		--platform=${TARGET_PLATFORM} \
		${DOCKER_TAG_LOCAL} \
		/app/main --run-once

docker-serve:
	docker run -it --rm \
		-p${PORT}:${PORT} \
		--platform=${TARGET_PLATFORM} \
		${DOCKER_TAG_LOCAL} \
		/app/main --port ${PORT}

SERVICE_IMG := ${DOCKER_DEPLOY}
PUSH_FROM := ""

docker-publish: docker-build
	@echo "Publishing docker image '${DOCKER_TAG}'"
	docker tag ${DOCKER_TAG_LOCAL} ${DOCKER_TAG}
	$(eval size:=$(shell docker inspect ${DOCKER_TAG} --format='{{.Size}}' | tr -cd '0-9'))
	$(eval imageSize:=$(shell expr ${size} + 0 ))
	@echo "... imageSize is ${imageSize}"
	@if [ ${imageSize} -gt 2000000000 ]; then \
		set -e ; \
		echo "preparing upload from local registry"; \
		if [ -z "$(shell docker ps -a -q -f name=registry-2)" ]; then \
			echo "running local registry-2"; \
			docker run --restart always -d -p 8081:5000 --name registry-2 registry:2 ; \
		fi; \
		docker tag ${DOCKER_TAG} localhost:8081/${DOCKER_TAG} ; \
		docker push localhost:8081/${DOCKER_TAG} ; \
		$(MAKE) PUSH_FROM="localhost:8081/" docker-publish-common ; \
	else \
		$(MAKE) PUSH_FROM="--local " docker-publish-common; \
	fi

docker-publish-common:
	$(eval log:=$(shell ivcap package push --force ${PUSH_FROM}${DOCKER_TAG} | tee /dev/tty))
	$(eval registry := $(shell echo ${DOCKER_REGISTRY} | cut -d'/' -f1))
	$(eval SERVICE_IMG := $(shell echo ${log} | sed -E "s/.*([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}.*) pushed/\1/"))
	@if [ "${SERVICE_IMG}" == "" ] || [ "${SERVICE_IMG}" == "${DOCKER_TAG}" ]; then \
		echo "service package push failed"; \
		exit 1; \
	fi
	@echo ">> Successfully published '${DOCKER_TAG}' as '${SERVICE_IMG}'"

service-register: # docker-publish
	$(eval image:=$(shell ivcap package list ${DOCKER_TAG}))
	@if [ "${image}" == "" ]; then \
		echo "cannot obtain docker package reference"; \
		exit 1; \
	fi
	cat ${PROJECT_DIR}/service.json \
	| sed 's|#PACKAGE_URN#|${image}|' \
	| sed 's|#SERVICE_URN#|${SERVICE_URN}|' \
	| ivcap --context local-api aspect update ${SERVICE_URN} -f - --timeout 600
