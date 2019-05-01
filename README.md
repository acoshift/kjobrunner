# kjobrunner

Simplify Kubernetes API for run jobs

## Testing

```bash
# create namespace
$ kubectl create ns kjobrunner

# create service account
$ kubectl create sa runner -n kjobrunner

# grant service account permissions
$ cat << EOF | kubectl apply -f - -n kjobrunner
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: runner
rules:
- apiGroups: ["batch"]
  resources: ["jobs"]
  verbs: ["*"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: runner
subjects:
- kind: ServiceAccount
  name: runner
roleRef:
  kind: Role
  name: runner
  apiGroup: rbac.authorization.k8s.io
EOF

# run test
$ TOKEN=$(kubectl get sa/runner -ojson | jq '.secrets[0].name' -r | xargs kubectl get secret -ojson | jq '.data.token' -r | base64 -D) go test .
```
