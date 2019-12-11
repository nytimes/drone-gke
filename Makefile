# executables
docker := $(shell which docker) --log-level error
gcloud := $(shell which gcloud) --quiet --verbosity=error
git := $(shell which git)
go := $(shell which go)
grep := $(shell which grep) --quiet --no-messages
kubectl := $(shell which kubectl)
terraform := $(shell which terraform)

# project options
project_tmp_dir = tmp

# plugin's binary options
binary_name = drone-gke
coverage_name = coverage.out

# git options
git_default_branch = $(shell cat $(CURDIR)/.git/refs/remotes/origin/HEAD | cut -d ' ' -f 2 | cut -d '/' -f 4)
git_current_branch = $(shell $(git) rev-parse --abbrev-ref HEAD)
git_current_revision = $(shell $(git) rev-parse --short HEAD)

# plugin's docker options
docker_default_repo_name = nytimes
docker_image_name = drone-gke
docker_default_tag = latest
ifneq ($(git_current_branch), $(git_default_branch))
	# if current branch name is NOT .git/refs/remotes/origin/HEAD (i.e., master)
	# then use the current branch name as the default docker tag
	docker_default_tag = $(git_current_branch)
endif
docker_default_file = Dockerfile
docker_default_build_context = $(CURDIR)

# gcloud SDK options
gcloud_default_sdk_tag = alpine

# test resource options
test_sa_key_path = $(project_tmp_dir)/key.json
terraform_base_dir = terraform
terraform_dir = $(terraform_base_dir)/.terraform
terraform_state_file = $(terraform_base_dir)/terraform.tfstate
terraform_plan_file = $(terraform_base_dir)/test.tfplan

# print usage (default target)
.PHONY : usage
usage :
	@echo "Usage:"
	@echo ""
	@echo "  \$$ make check               # run tests for required software"
	@echo "  \$$ make clean               # delete plugin binary, docker image, test resources"
	@echo "  \$$ make release             # build and push plugin docker image"
	@echo "  \$$ make docker-image        # build plugin docker image"
	@echo "  \$$ make docker-push         # push plugin docker image"
	@echo "  \$$ make run                 # run docker container using local-example"
	@echo "  \$$ make $(binary_name)           # build plugin binary"
	@echo "  \$$ make $(coverage_name)        # test plugin binary"
	@exit 0

# check that docker is installed
.PHONY : check-docker
check-docker :
ifneq (0,$(shell command -v $(docker) 2>&1 >/dev/null ; echo $$?))
	$(error "docker not installed")
else
	@exit 0
endif

# check that gcloud is installed
.PHONY : check-gcloud
check-gcloud :
ifneq (0,$(shell command -v $(gcloud) 2>&1 >/dev/null ; echo $$?))
	$(error "gcloud not installed")
else
	@exit 0
endif

# check that git is installed
.PHONY : check-git
check-git :
ifneq (0,$(shell command -v $(git) 2>&1 >/dev/null ; echo $$?))
	$(error "git not installed")
else
	@exit 0
endif

# check that go is installed
.PHONY : check-go
check-go :
ifneq (0,$(shell command -v $(go) 2>&1 >/dev/null ; echo $$?))
	$(error "go not installed")
else
	@exit 0
endif

# check that kubectl is installed
.PHONY : check-kubectl
check-kubectl :
ifneq (0,$(shell command -v $(kubectl) 2>&1 >/dev/null ; echo $$?))
	$(error "kubectl not installed")
else
	@exit 0
endif

# check that terraform is installed
.PHONY : check-terraform
check-terraform :
ifneq (0,$(shell command -v $(terraform) 2>&1 >/dev/null ; echo $$?))
	$(error "terraform not installed")
else
	@exit 0
endif

# check that all required executables are installed
.PHONY : check
check : check-docker check-gcloud check-git check-go check-kubectl check-terraform

# clean configuration
clean : export docker_repo_name ?= $(docker_default_repo_name)
clean : export docker_tag ?= $(docker_default_tag)

# delete binary, docker image, destroy any test resources
.PHONY : clean
clean :
	@rm -f $(binary_name) $(coverage_name)
	@$(docker) rmi --force $(docker_repo_name)/$(docker_image_name):$(docker_tag) 2>/dev/null
	@$(MAKE) destroy-test-resources

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

# test coverage configuration
$(coverage_name) : export CGO_ENABLED ?= 0
$(coverage_name) : export GO111MODULE ?= on
$(coverage_name) : export GOPROXY ?= https://proxy.golang.org

# test binary
$(coverage_name) : $(binary_name)
	@$(go) test -cover -vet all -coverprofile=$@

.PHONY : test-coverage
test-coverage : $(coverage_name)

# docker build configuration
docker-image : export docker_build_context ?= $(docker_default_build_context)
docker-image : export docker_file ?= $(docker_default_file)
docker-image : export docker_repo_name ?= $(docker_default_repo_name)
docker-image : export docker_tag ?= $(docker_default_tag)
docker-image : export gcloud_sdk_tag ?= $(gcloud_default_sdk_tag)

# build docker image
.PHONY : docker-image
docker-image : $(binary_name) $(docker_file)
	@$(docker) build \
		--build-arg GCLOUD_SDK_TAG=$(gcloud_sdk_tag) \
		--tag $(docker_repo_name)/$(docker_image_name):$(docker_tag) \
		--file $(docker_file) $(docker_build_context)

# docker push configuration
docker-push : export docker_repo_name ?= $(docker_default_repo_name)
docker-push : export docker_tag ?= $(docker_default_tag)

# push docker image
.PHONY : docker-push
docker-push :
	@$(docker) push $(docker_repo_name)/$(docker_image_name):$(docker_tag)

# compile and test binary, build and push docker image
.PHONY : release
release : drone-gke test-coverage docker-image docker-push

# local docker run configuration
run : export CONFIG_HOME ?= $(CURDIR)/local-example
run : export PLUGIN_CLUSTER ?= $(shell $(gcloud) config get-value container/cluster 2>/dev/null)
run : export PLUGIN_DRY_RUN ?= 1
run : export PLUGIN_NAMESPACE ?= drone-gke-test
run : export PLUGIN_REGION ?= $(shell $(gcloud) config get-value compute/region 2>/dev/null)
run : export PLUGIN_SECRET_TEMPLATE ?= $(CONFIG_HOME)/.kube.sec.yml
run : export PLUGIN_TEMPLATE ?= $(CONFIG_HOME)/.kube.yml
run : export PLUGIN_VARS ?= $(shell cat $(CONFIG_HOME)/vars.json)
run : export PLUGIN_VERBOSE ?= 1
run : export PLUGIN_ZONE ?= $(shell $(gcloud) config get-value compute/zone 2>/dev/null)
run : export SECRET_APP_API_KEY ?= 123
run : export SECRET_BASE64_P12_CERT ?= "cDEyCg=="
run : export TOKEN ?= $(shell cat $(test_sa_key_path))
run : export docker_repo_name ?= $(docker_default_repo_name)
run : export docker_tag ?= $(docker_default_tag)
run : export docker_cmd ?=

# run docker container using local-example
.PHONY : run
run :
	@$(docker) run \
		--env PLUGIN_CLUSTER \
		--env PLUGIN_DRY_RUN \
		--env PLUGIN_EXPAND_ENV_VARS \
		--env PLUGIN_KUBECTL_VERSION \
		--env PLUGIN_NAMESPACE \
		--env PLUGIN_PROJECT \
		--env PLUGIN_REGION \
		--env PLUGIN_SECRET_TEMPLATE \
		--env PLUGIN_TEMPLATE \
		--env PLUGIN_VARS \
		--env PLUGIN_VERBOSE \
		--env PLUGIN_WAIT_ROLLOUTS \
		--env PLUGIN_WAIT_DEPLOYMENTS \
		--env PLUGIN_WAIT_SECONDS \
		--env PLUGIN_ZONE \
		--env SECRET_APP_API_KEY \
		--env SECRET_BASE64_P12_CERT \
		--env TOKEN \
		--volume $(CONFIG_HOME):$(CONFIG_HOME) \
		--workdir $(CONFIG_HOME) \
		$(docker_repo_name)/$(docker_image_name):$(docker_tag) $(docker_cmd)

# create project tmp dir
$(project_tmp_dir) :
	@[ -d $@ ] || mkdir -p $@

$(terraform_dir) :
	@cd $(terraform_base_dir) && \
		$(terraform) init

$(terraform_state_file) : $(terraform_dir) $(terraform_base_dir)/main.tf
	@cd $(terraform_base_dir) && \
		$(terraform) refresh

$(terraform_plan_file) : export GOOGLE_PROJECT ?= $(shell $(gcloud) config get-value core/project 2>/dev/null)
$(terraform_plan_file) : export GOOGLE_OAUTH_ACCESS_TOKEN ?= $(shell $(gcloud) auth print-access-token 2>/dev/null)
$(terraform_plan_file) : export TF_VAR_drone_gke_test_key_path = $(CURDIR)/$(test_sa_key_path)

$(terraform_plan_file) : $(terraform_state_file) $(terraform_base_dir)/main.tf
	@cd $(terraform_base_dir) && \
		$(terraform) plan -out $(CURDIR)/$@

# create test resources configuration
$(test_sa_key_path) : export GOOGLE_PROJECT ?= $(shell $(gcloud) config get-value core/project 2>/dev/null)
$(test_sa_key_path) : export GOOGLE_OAUTH_ACCESS_TOKEN ?= $(shell $(gcloud) auth print-access-token 2>/dev/null)
$(test_sa_key_path) : export TF_VAR_drone_gke_test_key_path = $(CURDIR)/$(test_sa_key_path)

# create test resources
$(test_sa_key_path) : $(project_tmp_dir) $(terraform_plan_file) $(terraform_base_dir)/main.tf
	@cd $(terraform_base_dir) && \
		$(terraform) apply $(CURDIR)/$(terraform_plan_file)

.PHONY : test-resources
test-resources : $(test_sa_key_path)

# destroy test service account credentials configuration
destroy-terraform : export GOOGLE_PROJECT ?= $(shell $(gcloud) config get-value core/project 2>/dev/null)
destroy-terraform : export GOOGLE_OAUTH_ACCESS_TOKEN ?= $(shell $(gcloud) auth print-access-token 2>/dev/null)
destroy-terraform : export TF_VAR_drone_gke_test_key_path = $(CURDIR)/$(test_sa_key_path)

.PHONY : destroy-terraform
destroy-terraform :
	@[ ! -f $(terraform_state_file) ] || \
		cd terraform && \
			$(terraform) destroy -auto-approve

# destroy test service account credentials
.PHONY : clean-terraform
clean-terraform : destroy-terraform
	@rm -rf $(terraform_dir) $(terraform_state_file)* $(terraform_plan_file)

.PHONY : destroy-test-resources
destroy-test-resources : destroy-terraform clean-terraform

.PHONY : e2e
e2e : $(coverage_name) docker-image test-resources run
