.PHONY: build upload-test-secret upload-test-config deploy-test-service remove-test-service deploy-test-kube2clouddns remove-test-kube2clouddns

build:
	docker build -t kube2clouddns .

upload-test-secret:
	kubectl create secret generic clouddnsserviceaccount --from-file=deploy/clouddns_service_account.json

upload-test-config:
	kubectl apply -f deploy/clouddns_config.yaml

deploy-test-service:
	kubectl apply -f deploy/example-service.yaml

remove-test-service:
	kubectl delete service my-nginx

deploy-test-kube2clouddns:
	kubectl apply -f deploy/kube2clouddns.yaml

remove-test-kube2clouddns:
	kubectl delete deployment kube2clouddns
