CHART_DIR := helm/darts-league
VALUES_FILE ?= helm/darts-league/values-dev.yaml
# Space-separated list of extra, typically untracked, values files layered on
# top of VALUES_FILE (e.g. VALUES_FILES=helm/darts-league/values-dev.local.yaml)
VALUES_FILES ?=
RELEASE_NAME ?= darts-league
NAMESPACE ?= darts-league
HELM ?= helm
VALUES_FLAGS := -f $(VALUES_FILE) $(foreach f,$(VALUES_FILES),-f $(f))

.PHONY: help helm-lint helm-template helm-diff helm-deploy helm-uninstall helm-test

help:
	@printf "Available targets:\n"
	@printf "  helm-lint      Lint the Helm chart with the selected values file\n"
	@printf "  helm-template  Render the Helm chart locally\n"
	@printf "  helm-diff      Show the Helm diff for the release\n"
	@printf "  helm-deploy    Upgrade or install the Helm release\n"
	@printf "  helm-uninstall Remove the Helm release\n"
	@printf "  helm-test      Run Helm test hooks for the release\n"
	@printf "\nOverrides:\n"
	@printf "  VALUES_FILE=%s\n" "$(VALUES_FILE)"
	@printf "  RELEASE_NAME=%s\n" "$(RELEASE_NAME)"
	@printf "  NAMESPACE=%s\n" "$(NAMESPACE)"

helm-lint:
	$(HELM) lint $(CHART_DIR) $(VALUES_FLAGS)

helm-template:
	$(HELM) template $(RELEASE_NAME) $(CHART_DIR) -n $(NAMESPACE) $(VALUES_FLAGS)

helm-diff:
	$(HELM) diff upgrade --install $(RELEASE_NAME) $(CHART_DIR) -n $(NAMESPACE) $(VALUES_FLAGS)

helm-deploy:
	$(HELM) upgrade --install $(RELEASE_NAME) $(CHART_DIR) -n $(NAMESPACE) $(VALUES_FLAGS)

helm-uninstall:
	$(HELM) uninstall $(RELEASE_NAME) -n $(NAMESPACE)

helm-test:
	$(HELM) test $(RELEASE_NAME) -n $(NAMESPACE)
