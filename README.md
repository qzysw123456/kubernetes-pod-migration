# kubernetes-pod-migration

## reference 
https://github.com/kubernetes/sample-cli-plugin

## design
This project should contain 3 parts:

A plugin which extend the kubectl, accept command like `kubectl plugin checkpoint POD_NAME`, this command should get the node address where the pod is indeed running, then send request to a helper daemon.

A set of helper daemon running on every node. The helper daemon should be able receive the request sent from plugin, and know the pod.Spec. It utilize docker checkpoint feature so that it can checkpoint the containers. Attention, CRIU should be installed in every host machine to help this procedure.

A slightly modified Kubectl, a modification should in (docker-shim.go), thus the kubelet can start a container from saved state. Although this interface violates CRI, which works for every container runtime, but in order to support this docker feature, it is a effective way to just extend the docker-shim.

