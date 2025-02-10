# ------- Node settings
#

# Variables used by Node settings
ENG_NODE_VARIABLES := \
	NVM_DIR \

ENG_NODE_VERBOSE_VARIABLES := \
	PKG_CONFIG \

# Whether we are meant to use node.  (See node.mk for autodetection)
ENG_USING_NODE ?= $(ENG_AUTODETECT_USING_NODE)
