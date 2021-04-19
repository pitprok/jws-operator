IMAGE ?= docker.io/${USER}/jws-operator:latest
PROG  := jws-operator
NAMESPACE :=`oc project -q`
VERSION ?= 1.1.0
.DEFAULT_GOAL := help
DATETIME := `date -u +'%FT%TZ'`
CONTAINER_IMAGE ?= "${IMAGE}"

## setup                                    Ensure the operator-sdk is installed.
setup:
	./build/setup-operator-sdk.sh

setup-e2e-test:
	./build/setup-operator-sdk-e2e-tests.sh

## tidy                                     Ensures modules are tidy.
tidy:
	export GOPROXY=proxy.golang.org
	go mod tidy

## vendor                                   Ensures vendor directory is up to date
vendor: go.mod go.sum
	go mod vendor
	go generate -mod=vendor ./...

## codegen                                  Ensures code is generated.
codegen: setup
	operator-sdk generate k8s
	operator-sdk generate openapi

## build/_output/bin/                       Creates the directory where the executable is outputted.
build/_output/bin/:
	mkdir -p build/_output/bin/

## build/_output/bin/jws-operator     Compiles the operator
build/_output/bin/jws-operator: $(shell find pkg) $(shell find cmd) vendor | build/_output/bin/
	CGO_ENABLED=0 go build -mod=vendor -a -o build/_output/bin/jws-operator github.com/web-servers/jws-operator/cmd/manager

.PHONY: build

## build                                    Builds the operator
build: tidy build/_output/bin/jws-operator

## image                                    Builds the operator's image
image: build
	podman build -t "$(IMAGE)" . -f build/Dockerfile
	$(MAKE) generate-operator.yaml

## push                                     Push Docker image to the docker.io repository.
push: image
	podman push "$(IMAGE)"

## clean                                    Remove all generated build files.
clean:
	rm -rf build/_output/

## generate-kubernetes_operator.yaml        Generates the deployment file for Kubernetes
generate-operator.yaml:
	sed 's|@OP_IMAGE_TAG@|$(IMAGE)|' deploy/operator.template > deploy/operator.yaml

## run-openshift                            Run the JWS operator on OpenShift.
run-openshift: push
	oc create -f deploy/crds/web.servers.org_webservers_crd.yaml
	oc create -f deploy/service_account.yaml
	oc create -f deploy/role.yaml
	oc create -f deploy/role_binding.yaml
	oc apply -f deploy/operator.yaml
clean-openshift:
	oc delete -f deploy/crds/web.servers.org_webservers_crd.yaml
	oc delete -f deploy/service_account.yaml
	oc delete -f deploy/role.yaml
	oc delete -f deploy/role_binding.yaml


## run-kubernetes                           Run the Tomcat operator on kubernetes.
run-kubernetes: push
	kubectl create -f deploy/crds/web.servers.org_webservers_crd.yaml
	kubectl create -f deploy/service_account.yaml
	kubectl create -f deploy/role.yaml
	kubectl create -f deploy/role_binding.yaml
	kubectl apply -f deploy/operator.yaml

test: test-e2e-5-local

test-e2e-5-local: setup-e2e-test
	oc delete namespace "jws-e2e-tests" || true
	oc new-project "jws-e2e-tests" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-1 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --operator-namespace jws-e2e-tests --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"


test2: setup-e2e-test
	oc delete namespace "jws-e2e-tests-2" || true
	oc new-project "jws-e2e-tests-2" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-2 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-2 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-2 --operator-namespace jws-e2e-tests-2 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"

test3: setup-e2e-test
	oc delete namespace "jws-e2e-tests-3" || true
	oc new-project "jws-e2e-tests-3" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-3 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-3 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-3 --operator-namespace jws-e2e-tests-3 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"

test4: setup-e2e-test
	oc delete namespace "jws-e2e-tests-4" || true
	oc new-project "jws-e2e-tests-4" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-4 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-4 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-4 --operator-namespace jws-e2e-tests-4 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"

test5: setup-e2e-test
	oc delete namespace "jws-e2e-tests-5" || true
	oc new-project "jws-e2e-tests-5" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-5 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-5 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-5 --operator-namespace jws-e2e-tests-5 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"

test6: setup-e2e-test
	oc delete namespace "jws-e2e-tests-6" || true
	oc new-project "jws-e2e-tests-6" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-6 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-6 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-6 --operator-namespace jws-e2e-tests-6 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"

test7: setup-e2e-test
	oc delete namespace "jws-e2e-tests-7" || true
	oc new-project "jws-e2e-tests-7" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-7 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-7 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-7 --operator-namespace jws-e2e-tests-7 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"


test8: setup-e2e-test
	oc delete namespace "jws-e2e-tests-8" || true
	oc new-project "jws-e2e-tests-8" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-8 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-8 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-8 --operator-namespace jws-e2e-tests-8 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"


test9: setup-e2e-test
	oc delete namespace "jws-e2e-tests-9" || true
	oc new-project "jws-e2e-tests-9" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-9 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-9 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-9 --operator-namespace jws-e2e-tests-9 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"


test10: setup-e2e-test
	oc delete namespace "jws-e2e-tests-10" || true
	oc new-project "jws-e2e-tests-10" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-10 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-10 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-10 --operator-namespace jws-e2e-tests-10 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"



test11: setup-e2e-test
	oc delete namespace "jws-e2e-tests-11" || true
	oc new-project "jws-e2e-tests-11" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-11 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-11 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-11 --operator-namespace jws-e2e-tests-11 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"




test12: setup-e2e-test
	oc delete namespace "jws-e2e-tests-12" || true
	oc new-project "jws-e2e-tests-12" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-12 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-12 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-12 --operator-namespace jws-e2e-tests-12 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"




test13: setup-e2e-test
	oc delete namespace "jws-e2e-tests-13" || true
	oc new-project "jws-e2e-tests-13" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-13 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-13 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-13 --operator-namespace jws-e2e-tests-13 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"




test14: setup-e2e-test
	oc delete namespace "jws-e2e-tests-14" || true
	oc new-project "jws-e2e-tests-14" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-14 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-14 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-14 --operator-namespace jws-e2e-tests-14 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"





test15: setup-e2e-test
	oc delete namespace "jws-e2e-tests-15" || true
	oc new-project "jws-e2e-tests-15" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-15 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-15 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-15 --operator-namespace jws-e2e-tests-15 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"




test16: setup-e2e-test
	oc delete namespace "jws-e2e-tests-16" || true
	oc new-project "jws-e2e-tests-16" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-16 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-16 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-16 --operator-namespace jws-e2e-tests-16 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"





test17: setup-e2e-test
	oc delete namespace "jws-e2e-tests-17" || true
	oc new-project "jws-e2e-tests-17" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-17 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-17 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-17 --operator-namespace jws-e2e-tests-17 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"




test18: setup-e2e-test
	oc delete namespace "jws-e2e-tests-18" || true
	oc new-project "jws-e2e-tests-18" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-18 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-18 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-18 --operator-namespace jws-e2e-tests-18 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"


test19: setup-e2e-test
	oc delete namespace "jws-e2e-tests-19" || true
	oc new-project "jws-e2e-tests-19" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-19 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-19 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-19 --operator-namespace jws-e2e-tests-19 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"


test20: setup-e2e-test
	oc delete namespace "jws-e2e-tests-20" || true
	oc new-project "jws-e2e-tests-20" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-20 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-20 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-20 --operator-namespace jws-e2e-tests-20 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"


test21: setup-e2e-test
	oc delete namespace "jws-e2e-tests-21" || true
	oc new-project "jws-e2e-tests-21" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-21 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-21 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-21 --operator-namespace jws-e2e-tests-21 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"


test22: setup-e2e-test
	oc delete namespace "jws-e2e-tests-22" || true
	oc new-project "jws-e2e-tests-22" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-22 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-22 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-22 --operator-namespace jws-e2e-tests-22 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"


test23: setup-e2e-test
	oc delete namespace "jws-e2e-tests-23" || true
	oc new-project "jws-e2e-tests-23" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-23 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-23 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-23 --operator-namespace jws-e2e-tests-23 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"


test24: setup-e2e-test
	oc delete namespace "jws-e2e-tests-24" || true
	oc new-project "jws-e2e-tests-24" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-24 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-24 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-24 --operator-namespace jws-e2e-tests-24 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"

test25: setup-e2e-test
	oc delete namespace "jws-e2e-tests-25" || true
	oc new-project "jws-e2e-tests-25" || true
	oc create -f xpaas-streams/jws54-tomcat9-image-stream.json -n jws-e2e-tests-25 || true
	LOCAL_OPERATOR=true OPERATOR_NAME=jws-operator-25 ./operator-sdk-e2e-tests test local ./test/e2e/5 --verbose --debug --namespace jws-e2e-tests-25 --operator-namespace jws-e2e-tests-25 --local-operator-flags "--zap-devel --zap-level=5" --global-manifest ./deploy/crds/web.servers.org_webservers_crd.yaml --go-test-flags "-timeout=30m"

generate-csv:
	operator-sdk generate crds
	operator-sdk generate csv --verbose --csv-version $(VERSION) --update-crds
	mkdir manifests/jws/$(VERSION)/ || true
	mv deploy/olm-catalog/jws-operator/manifests/* manifests/jws/$(VERSION)/
	rm -r deploy/olm-catalog

customize-csv: generate-csv
	DATETIME=$(DATETIME) CONTAINER_IMAGE=$(CONTAINER_IMAGE) OPERATOR_VERSION=$(VERSION) build/customize_csv.sh

catalog:
	podman build -f build/catalog.Dockerfile -t my-test-catalog:latest .
	podman tag my-test-catalog:latest quay.io/${USER}/my-test-catalog:latest
	podman push quay.io/${USER}/my-test-catalog:latest
	sed s:@USER@:${USER}: catalog.yaml.template > catalog.yaml
	sed s:@NAMESPACE@:${NAMESPACE}: operatorgroup.yaml.template > operatorgroup.yaml
	sed s:@NAMESPACE@:${NAMESPACE}: subscription.yaml.template > subscription.yaml
	@echo ""
	@echo "Use oc create -f catalog.yaml to install the CatalogSource for the operator"
	@echo ""
	@echo "Use oc create -f operatorgroup.yaml and oc create -f subscription.yaml to install the operator in ${NAMESPACE}"
	@echo "or use the openshift web interface on the installed operator"

help : Makefile
	@sed -n 's/^##//p' $<
