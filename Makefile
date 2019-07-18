# executables
docker := $(shell which docker) --log-level error
gcloud := $(shell which gcloud) --quiet --verbosity=error
git := $(shell which git)
go := $(shell which go)
kubectl := $(shell which kubectl)

# plugin's binary options
binary_name = drone-gke

# git options
git_default_branch = $(shell cat $(CURDIR)/.git/refs/remotes/origin/HEAD | cut -d ' ' -f 2 | cut -d '/' -f 4)
git_current_branch = $(shell $(git) rev-parse --abbrev-ref HEAD)
git_current_revision = $(shell $(git) rev-parse --short HEAD)

# plugin's docker options
docker_default_repo_name = nytimes
docker_image_name = drone-gke
docker_default_tag = latest
ifneq ($(git_current_branch), $(git_default_branch))
	docker_default_tag = $(git_current_branch)
endif
docker_default_file = $(CURDIR)/Dockerfile
docker_default_build_context = $(CURDIR)

# gcloud SDK options
gcloud_default_sdk_version = alpine

# print usage (default target)
usage :
	@echo "Usage:"
	@echo ""
	@echo "  \$$ make check               # run tests for required software"
	@echo "  \$$ make clean               # delete plugin binary, docker image"
	@echo "  \$$ make dist                # build and push plugin docker image"
	@echo "  \$$ make docker-image        # build plugin docker image"
	@echo "  \$$ make docker-release      # push plugin docker image"
	@echo "  \$$ make docker-run-local    # run docker container using local-example"
	@echo "  \$$ make $(binary_name)           # build plugin binary"
	@exit 0

.PHONY : usage

# check that docker is installed
check-docker :
ifneq (0,$(shell command -v $(docker) 2>&1 >/dev/null ; echo $$?))
	$(error "docker not installed")
else
	@exit 0
endif

.PHONY : check-docker

# check that gcloud is installed
check-gcloud :
ifneq (0,$(shell command -v $(gcloud) 2>&1 >/dev/null ; echo $$?))
	$(error "gcloud not installed")
else
	@exit 0
endif

.PHONY : check-gcloud

# check that git is installed
check-git :
ifneq (0,$(shell command -v $(git) 2>&1 >/dev/null ; echo $$?))
	$(error "git not installed")
else
	@exit 0
endif

.PHONY : check-git

# check that go is installed
check-go :
ifneq (0,$(shell command -v $(go) 2>&1 >/dev/null ; echo $$?))
	$(error "go not installed")
else
	@exit 0
endif

.PHONY : check-go

# check that kubectl is installed
check-kubectl :
ifneq (0,$(shell command -v $(kubectl) 2>&1 >/dev/null ; echo $$?))
	$(error "kubectl not installed")
else
	@exit 0
endif

.PHONY : check-kubectl

# check that all required executables are installed
check : check-docker check-gcloud check-git check-go check-kubectl

.PHONY : check

# clean configuration
clean : export docker_repo_name ?= $(docker_default_repo_name)
clean : export docker_tag ?= $(docker_default_tag)

# delete binary, docker image
clean :
	@rm -f $(binary_name)
	@$(docker) rmi --force $(docker_repo_name)/$(docker_image_name):$(docker_tag) 2>/dev/null

.PHONY : clean

# compilation configuration
$(binary_name) : export CGO_ENABLED ?= 0
$(binary_name) : export GO111MODULE ?= on
$(binary_name) : export GOARCH ?= amd64
$(binary_name) : export GOOS ?= linux
$(binary_name) : export GOPROXY ?= https://proxy.golang.org
$(binary_name) : export revision ?= $(git_current_revision)

# compile binary
$(binary_name) : main.go exec.go go.sum
	@$(go) build -a -ldflags "-X main.rev=$(revision)"

# docker build configuration
docker-image : export docker_build_context ?= $(docker_default_build_context)
docker-image : export docker_file ?= $(docker_default_file)
docker-image : export docker_repo_name ?= $(docker_default_repo_name)
docker-image : export docker_tag ?= $(docker_default_tag)
docker-image : export gcloud_sdk_version ?= $(gcloud_default_sdk_version)

# build docker image
docker-image : $(binary_name) $(docker_file)
	@$(docker) build \
		--build-arg GCLOUD_SDK_VERSION=$(gcloud_sdk_version) \
		--tag $(docker_repo_name)/$(docker_image_name):$(docker_tag) \
		--file $(docker_file) $(docker_build_context)

.PHONY : docker-image

# docker push configuration
docker-release : export docker_repo_name ?= $(docker_default_repo_name)
docker-release : export docker_tag ?= $(docker_default_tag)

# push docker image
docker-release :
	@$(docker) push $(docker_repo_name)/$(docker_image_name):$(docker_tag)

.PHONY : docker-release

# compile binary, build and push docker image
dist : docker-image docker-release

# local docker run configuration
docker-run-local : export CONFIG_HOME ?= $(CURDIR)/local-example
docker-run-local : export GOOGLE_APPLICATION_CREDENTIALS ?= xxx
docker-run-local : export PLUGIN_CLUSTER ?= yyy
docker-run-local : export PLUGIN_NAMESPACE ?= drone-gke
docker-run-local : export PLUGIN_SECRET_TEMPLATE ?= $(CONFIG_HOME)/.kube.sec.yml
docker-run-local : export PLUGIN_TEMPLATE ?= $(CONFIG_HOME)/.kube.yml
docker-run-local : export PLUGIN_VARS_PATH ?= $(CONFIG_HOME)/vars.json
docker-run-local : export PLUGIN_ZONE ?= zzz
docker-run-local : export SECRET_API_TOKEN ?= 123
docker-run-local : export SECRET_BASE64_P12_CERT ?= "cDEyCg=="
docker-run-local : export docker_repo_name ?= $(docker_default_repo_name)
docker-run-local : export docker_tag ?= $(docker_default_tag)
docker-run-local : export docker_cmd ?= --dry-run --verbose

# run docker container using local-example
docker-run-local :
	@[ -d $(CONFIG_HOME) ] || ( echo "$(CONFIG_HOME) does not exist" ; exit 1 )
	@[ -f $(PLUGIN_SECRET_TEMPLATE) ] || ( echo "$(PLUGIN_SECRET_TEMPLATE) does not exist" ; exit 1 )
	@[ -f $(PLUGIN_TEMPLATE) ] || ( echo "$(PLUGIN_TEMPLATE) does not exist" ; exit 1 )
	@[ -f $(PLUGIN_VARS_PATH) ] || ( echo "$(PLUGIN_VARS_PATH) does not exist" ; exit 1 )
	@[ -f $(GOOGLE_APPLICATION_CREDENTIALS) ] || ( echo "$(GOOGLE_APPLICATION_CREDENTIALS) does not exist" ; exit 1 )
	@cd $(CONFIG_HOME) ; \
	$(docker) run \
		--env PLUGIN_CLUSTER=$(PLUGIN_CLUSTER) \
		--env PLUGIN_NAMESPACE=$(PLUGIN_NAMESPACE) \
		--env PLUGIN_SECRET_TEMPLATE=$(PLUGIN_SECRET_TEMPLATE) \
		--env PLUGIN_TEMPLATE=$(PLUGIN_TEMPLATE) \
		--env PLUGIN_VARS="$$(cat $(PLUGIN_VARS_PATH))" \
		--env PLUGIN_ZONE=$(PLUGIN_ZONE) \
		--env SECRET_API_TOKEN=$(SECRET_API_TOKEN) \
		--env SECRET_BASE64_P12_CERT=$(SECRET_BASE64_P12_CERT) \
		--env TOKEN="$$(cat $(GOOGLE_APPLICATION_CREDENTIALS))" \
		--volume $(CONFIG_HOME):$(CONFIG_HOME) \
		--workdir $(CONFIG_HOME) \
		$(docker_repo_name)/$(docker_image_name):$(docker_tag) $(docker_cmd)

.PHONY : docker-run-local

.PHONY : dist
