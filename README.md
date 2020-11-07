# Korb

Move Data from PVCs between StorageClasses, or rename them.

### Usage

```
Error: requires at least 1 arg(s), only received 0
Usage:
  korb [pvc [pvc]] [flags]

Flags:
      --docker-image string            Image to use for moving jobs (default "beryju/korb-mover:latest")
      --force                          Ignore warning which would normally halt the tool during validation.
  -h, --help                           help for korb
      --kubeConfig string              (optional) absolute path to the kubeConfig file (default "/home/jens/.kube/config")
      --new-pvc-name string            Name for the new PVC. If empty, same name will be reused.
      --new-pvc-size string            Size for the new PVC. If empty, the size of the source will be used. Accepts formats like used in Kubernetes Manifests (Gi, Ti, ...)
      --new-pvc-storage-class string   Storage class to use for the new PVC. If empty, the storage class of the source will be used.

requires at least 1 arg(s), only received 0
```

### Example

```
~ ./korb-linux-amd64 --new-pvc-storage-class ontap-ssd gitea-pvc
DEBU[0000] Created client from kubeconfig                component=migrator kubeconfig=/home/jens/.kube/config
DEBU[0000] Got current namespace                         component=migrator namespace=gitea
DEBU[0000] Got Source PVC                                component=migrator name=gitea-pvc uid=9820bc60-ba26-43f0-99ba-cd4385f6bdbf
DEBU[0000] No new Name given, using old name             component=migrator
DEBU[0000] Compatible Strategies:                        component=migrator
DEBU[0000] Copy the PVC to the new Storage class and with new size and a new name, delete the old PVC, and copy it back to the old name.  component=migrator
DEBU[0000] Only one compatible strategy, running         component=migrator
DEBU[0000] Set timeout from PVC size                     component=strategy timeout=20m0s
WARN[0000] This strategy assumes you've stopped all pods accessing this data.  component=strategy
DEBU[0000] Stage 1, creating temporary PVC               component=strategy
DEBU[0002] Stage 2, creating mover job                   component=strategy
DEBU[0002] Stage 3, starting job and waiting for copy    component=strategy
DEBU[0004] Waiting for job to finish...                  component=mover-job job-name=k8s-mover-job-9820bc60-ba26-43f0-99ba-cd4385f6bdbf
DEBU[0006] Waiting for job to finish...                  component=mover-job job-name=k8s-mover-job-9820bc60-ba26-43f0-99ba-cd4385f6bdbf
[...]
DEBU[0040] Cleaning up successful job                    component=mover-job
DEBU[0040] Stage 4, Deleting original PVC                component=strategy
DEBU[0042] Stage 5, Create final destination PVC         component=strategy
DEBU[0042] Stage 6, Create mover job to final destination  component=strategy
DEBU[0042] Stage 7, starting job and waiting for copy    component=strategy
DEBU[0044] Waiting for job to finish...                  component=mover-job job-name=k8s-mover-job-daa07336-d4ee-48be-afc8-ed2592537ac2
DEBU[0046] Waiting for job to finish...                  component=mover-job job-name=k8s-mover-job-daa07336-d4ee-48be-afc8-ed2592537ac2
DEBU[0048] Waiting for job to finish...                  component=mover-job job-name=k8s-mover-job-daa07336-d4ee-48be-afc8-ed2592537ac2
[...]
DEBU[0078] Cleaning up successful job                    component=mover-job
DEBU[0078] Stage 8, Deleting temporary PVC               component=strategy
INFO[0080] And we're done                                component=strategy
WARN[0080] Cleaning up...                                component=strategy
```
