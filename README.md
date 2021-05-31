This git repo is for to get what will be change before the code merge to master and deploy to cluster.

How to build?
checkout this gitrepository and run `go build ./`

How to use?
`./flux-precheck --kustomizationName=kustomizationname --kustomizationNamespace=namespace`

Result will be look like :
```
============result==============
serviceaccounts/default-xx action is deleted
deployments/default-reactor-manager action is deleted
deployment.apps/my-dep action is created
servicemonitors/default-taskmanager-metrics action is deleted
```



