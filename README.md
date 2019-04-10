# kubernetes-pod-migration

## reference 
https://github.com/kubernetes/sample-cli-plugin

## design
A set of helper daemon running on every node. The helper daemon should be able to read the container list inside an pod. It utilize docker checkpoint feature so that it can checkpoint the containers. CRIU should be installed in every host machine to help this procedure.

Kubectl (docker-shim) should be customized so that it can start a container from saved state.
