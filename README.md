# kubernetes-pod-migration

## Kubernetes pod migration tool 
It is a simple kubectl plugin that can live migrate a pod from a host to another.

At this point of time, although docker has already supported checkpoint/restore feature, Kubernetes doesn't provide any support, and (likely) won't support in the future. This is a experimental attempt that, without extending Kubernetes' container runtime interface, made Kubernetes support docker checkpoint/restore in a very easy way.

## design
This project contains 3 parts:

1. A plugin which extend the kubectl, accept command `kubectl migrate [NAMESPACE] POD_NAME DestHost`. `POD_NAME` is the pod you want to migrate, and `DestHost` is the desired host you want the pod migrate to.

2. A daemon set of agents running on every node. The helper daemon receives the request sent from plugin, checkpoint all containers inside a pod, and transfer the saved state to destination host.

3. A slightly modified Kubelet( <10 lines of code), which enables a pod start its container from saved state.

The plugin and agent are in this project, Kubelet is in my forked Kubernetes project, in the "experimental" branch.
