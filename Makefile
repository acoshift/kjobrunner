test:
	@TOKEN=$(shell kubectl get sa/runner -ojson | jq '.secrets[0].name' -r | xargs kubectl get secret -ojson | jq '.data.token' -r | base64 -D) go test . -count=1
