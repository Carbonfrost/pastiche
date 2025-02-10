#: node engineering
ENG_AVAILABLE_STACKS += node

# Automatically detect whether Node is in use.  This needs to be a static computation
# because it is used in the conditional below
ifneq ("$(wildcard .nvmrc)","")
ENG_AUTODETECT_USING_NODE = 1
else
ENG_AUTODETECT_USING_NODE = 0
endif

# User can define ENG_USING_NODE themselves to avoid autodeteciton
ifdef ENG_USING_NODE
_ENG_ACTUALLY_USING_NODE = $(ENG_USING_NODE)
else
_ENG_ACTUALLY_USING_NODE = $(ENG_AUTODETECT_USING_NODE)
endif

# Enable the tasks if we are using node
ifeq (1,$(_ENG_ACTUALLY_USING_NODE))

ENG_ENABLED_STACKS += node
_ENG_ACTUAL_NODE_VERSION = $(shell node --version)

## Install Node and project dependencies
node/init: | -node/init node/install
node/install: -node/install
node/clean: -node/clean
node/test: -node/test

fetch: node/install
clean: node/clean
test: node/test

else
node/init: -hint-unsupported-node
node/install: -hint-unsupported-node
node/clean: -hint-unsupported-node
node/test: -hint-unsupported-node
endif

## Add support for Node to the project
use/node: | -node/init

-node/init:
	$(Q) $(OUTPUT_COLLAPSED) eng/brew_bundle_inject nvm
	$(Q) $(OUTPUT_COLLAPSED) brew bundle
	$(Q) $(OUTPUT_COLLAPSED) bash -c '. "/opt/homebrew/opt/nvm/nvm.sh"; nvm install --latest-npm --lts; nvm use --lts'
	$(Q) $(OUTPUT_COLLAPSED) bash -c '. "/opt/homebrew/opt/nvm/nvm.sh"; nvm use --lts; node --version > .nvmrc'

-node/install: -requirements-node
	$(Q) $(OUTPUT_COLLAPSED) npm install

-node/clean: -check-command-npm
	$(Q) npm run clean --if-present
	$(Q) rm $(_STANDARD_VERBOSE_FLAG) -rdf node_modules	

-node/test: -check-command-npm
	$(Q) npm run test

-check-node-version: -check-command-node
	@ $(call _check_version,node,$(_ENG_ACTUAL_NODE_VERSION),$(NODE_VERSION))

-requirements-node: -check-node-version -check-command-node

-hint-unsupported-node:
	@ echo $(_HIDDEN_IF_BOOTSTRAPPING) "$(_WARNING) Nothing to do" \
	"because $(_MAGENTA)node$(_RESET) is not enabled (Investigate $(_CYAN)\`make use/node\`$(_RESET))"

-init-frameworks: node/init
