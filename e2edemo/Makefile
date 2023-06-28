PWD = $(abspath ./)

JAVASCORE_PATH = $(PWD)/../javascore
GRADLE = ./gradlew

HARDHAT = npx hardhat
TS_NODE = npx ts-node
# NETWORK = --network localhost

#Bsc VM
# NETWORK = --network bscVm

#ETH goerli TestNet
# NETWORK = --network goerli

# ETH Sepolia
NETWORK = --network sepolia

#BSC TestNet
# NETWORK = --network bscTest

SOLIDITY_CONTRACTS = ./solidity/contracts

.DEFAULT_GOAL := all
all:
	@echo $(PWD)

.PHONY: build clean deploy

clean-java:
	(cd $(JAVASCORE_PATH); $(GRADLE) clean)

clean-solidity:
	$(HARDHAT) clean
	$(RM) -rf \
	    $(SOLIDITY_CONTRACTS)/bmc \
	    $(SOLIDITY_CONTRACTS)/bmv* \
	    $(SOLIDITY_CONTRACTS)/xcall

clean: clean-java clean-solidity

copy-src:
	@ echo ">>> Copy BMC/BMV/xCall solidity codes" ; \
	$(TS_NODE) scripts/setup/copy_src.ts

compile: copy-src
	@ echo ">>> Compile solidity codes" ; \
	$(HARDHAT) compile

build-bmc-java:
	@ echo ">>> Build BMC javascore" ; \
	cd $(JAVASCORE_PATH) ; \
	$(GRADLE) bmc:optimizedJar

build-bmv-java:
	@ echo ">>> Build BMV javascore" ; \
	cd $(JAVASCORE_PATH) ; \
	$(GRADLE) bmv:bridge:optimizedJar \
		bmv:eth2:optimizedJar

build-xcall-java:
	@ echo ">>> Build xCall javascore" ; \
	cd $(JAVASCORE_PATH) ; \
	$(GRADLE) xcall:optimizedJar

build-dapp-java:
	@ echo ">>> Build DApp sample javascore" ; \
	cd $(JAVASCORE_PATH) ; \
	$(GRADLE) dapp-sample:optimizedJar

build-all: compile build-bmc-java build-bmv-java build-xcall-java build-dapp-java

deploy-bmc:
	@ echo ">>> Deploy BMC contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup/deploy_bmc.ts

deploy-bmc-chain:
	@ echo ">>> Deploy BMC-CHAIN contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_chain/deploy_bmc.ts

deploy-bmc-side:
	@ echo ">>> Deploy BMC-SIDE contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_side/deploy_bmc.ts

deploy-upgrade-bmc:
	@ echo ">>> Deploy upgrade BMC contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_upgrades/deploy_bmc.ts


deploy-bmv:
	@ echo ">>> Deploy BMV contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup/deploy_bmv.ts

deploy-bmv-chain:
	@ echo ">>> Deploy BMV-CHAIN contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_chain/deploy_bmv.ts

deploy-bmv-side:
	@ echo ">>> Deploy BMV-SIDE contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_side/deploy_bmv.ts

update-bmv-side:
	@ echo ">>> Deploy BMV-SIDE contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_side/update_bmv.ts

deploy-upgrade-bmv:
	@ echo ">>> Deploy upgrade BMV contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_upgrades/deploy_bmv.ts

deploy-upgrade-bmv-eth:
	@ echo ">>> Deploy upgrade ETH2 BMV contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_upgrades/deploy_bmv_eth.ts

deploy-xcall:
	@ echo ">>> Deploy xCall contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup/deploy_xcall.ts

deploy-xcall-chain:
	@ echo ">>> Deploy xCall-CHAIN contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_chain/deploy_xcall.ts

deploy-xcall-side:
	@ echo ">>> Deploy xCall-SIDE contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_side/deploy_xcall.ts

deploy-upgrade-xcall:
	@ echo ">>> Deploy upgrade xCall contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_upgrades/deploy_xcall.ts

deploy-dapp:
	@ echo ">>> Deploy DApp contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup/deploy_dapp.ts

deploy-dapp-chain:
	@ echo ">>> Deploy DApp-CHAIN contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_chain/deploy_dapp.ts

deploy-dapp-side:
	@ echo ">>> Deploy DApp-SIDE contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_side/deploy_dapp.ts

deploy-upgrade-dapp:
	@ echo ">>> Deploy upgrade DApp contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_upgrades/deploy_dapp.ts

upgrade-bmc:
	@ echo ">>> Upgrade BMC contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_upgrades/update_bmc.ts

upgrade-xcall:
	@ echo ">>> Upgrade xcall contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_upgrades/update_xcall.ts


remove_setting:
	@ echo ">>> remove setting" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_upgrades/remove_setting.ts



set-pubkey-chain:
	@ echo ">>> Set Pubkey Chain" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup_chain/set_pubkey.ts


prep : set-pubkey-chain

deploy-all: deploy-bmc deploy-bmv deploy-xcall deploy-dapp

# ori deploy set
deploy-bmc: deploy-bmc

deploy-other: deploy-bmv deploy-xcall deploy-dapp


# ori deploy set
deploy-bmc-chain: deploy-bmc-chain

deploy-other-chain: deploy-bmv-chain deploy-xcall-chain deploy-dapp-chain

# other deploy set
deploy-bmc-side: deploy-bmc-side

deploy-other-side: deploy-bmv-side deploy-xcall-side deploy-dapp-side

update-bmv-side : update-bmv-side
# deploy-other-side: deploy-xcall-side deploy-dapp-side

# deploy-other-side: deploy-bmv-side





deploy-upgrade-all: deploy-upgrade-bmc deploy-upgrade-bmv deploy-upgrade-xcall deploy-upgrade-dapp

deploy-upgrade-bmc: deploy-upgrade-bmc

deploy-upgrade-eth: deploy-upgrade-bmv-eth

deploy-upgrade-other: deploy-upgrade-xcall deploy-upgrade-dapp

remove_setting: remove_setting

upgrade-bmc: upgrade-bmc

deploy-upgrade-bmv: deploy-upgrade-bmv

upgrade-xcall: upgrade-xcall


start-nodes:
	@ echo ">>> Start nodes" ; \
	cd docker; \
	docker compose up -d

stop-nodes:
	@ echo ">>> Stop nodes" ; \
	cd docker; \
	docker compose down

setup-node:
	@ echo ">>> Setup ICON node" ; \
	$(TS_NODE) scripts/setup/setup_node.ts

run-demo:
	@ echo ">>> Run demo_dapp.ts " ; \
	$(HARDHAT) $(NETWORK) run scripts/demo_dapp.ts

run-network:
	@ echo ">>> Run demo_network.ts " ; \
	$(HARDHAT) $(NETWORK) run scripts/demo_network.ts

run-cost:
	@ echo ">>> Run demo_dapp_cost.ts " ; \
	$(HARDHAT) $(NETWORK) run scripts/demo_dapp_cost.ts

setup-fee:
	@ echo ">>> Run setup_fee.ts " ; \
	$(HARDHAT) $(NETWORK) run scripts/setup_fee.ts

get-reward:
	@ echo ">>> Run get_claim_reward.ts " ; \
	$(HARDHAT) $(NETWORK) run scripts/get_claim_reward.ts


bmv-init:
	@ echo ">>> Generate initial data of Java BMV for Ethereum 2.0" ; \
	cd ..; \
	make bmvinit; \
	./bin/bmvinit gen --url http://18.176.88.124:9596 --output $(PWD)/java_bmv_init_data.json