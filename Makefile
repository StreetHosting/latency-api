.PHONY: build test run dev clean deploy deploy-check verify package

BINARY := dist/latency-probe-linux-amd64
INVENTORY := deploy/inventory/hosts.yml

build:
	@bash scripts/build.sh

test:
	go test ./...

run: build
	LISTEN_ADDR=127.0.0.1:8080 go run ./cmd/probe

dev:
	LISTEN_ADDR=127.0.0.1:8080 go run ./cmd/probe

clean:
	rm -rf dist/

package: build
	tar -czf dist/latency-probe-bundle.tar.gz \
		$(BINARY) \
		deploy/systemd/latency-probe.service \
		configs/nginx/latency-probe.conf.template \
		configs/probe.env.example \
		scripts/install.sh \
		scripts/update.sh \
		scripts/healthcheck.sh

deploy: build
	@test -f $(INVENTORY) || (echo "Copy deploy/inventory/hosts.example.yml to hosts.yml"; exit 1)
	ansible-playbook -i $(INVENTORY) deploy/ansible/playbook.yml

deploy-check:
	ansible-playbook -i $(INVENTORY) deploy/ansible/playbook.yml --check

verify:
	@bash scripts/verify.sh

update-fleet: build
	ansible -i $(INVENTORY) probe_nodes -b -m copy -a "src=$(BINARY) dest=/usr/local/bin/latency-probe mode=0755"
	ansible -i $(INVENTORY) probe_nodes -b -m systemd -a "name=latency-probe state=restarted"
