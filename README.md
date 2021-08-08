# Korb

Move Data from PVCs between StorageClasses, or rename them.

### Installation

#### Using Homebrew

```
brew tap beryju/tap
brew install korb
```

#### Manually

Download the binary of the latest release from https://github.com/BeryJu/korb/releases

### Usage

```
Error: requires at least 1 arg(s), only received 0
Usage:
  korb [pvc [pvc]] [flags]

Flags:
      --docker-image string            Image to use for moving jobs (default "ghcr.io/beryju/korb-mover:latest")
      --force                          Ignore warning which would normally halt the tool during validation.
  -h, --help                           help for korb
      --kubeConfig string              (optional) absolute path to the kubeConfig file (default "/home/jens/.kube/config")
      --new-pvc-name string            Name for the new PVC. If empty, same name will be reused.
      --new-pvc-size string            Size for the new PVC. If empty, the size of the source will be used. Accepts formats like used in Kubernetes Manifests (Gi, Ti, ...)
      --new-pvc-storage-class string   Storage class to use for the new PVC. If empty, the storage class of the source will be used.
  -v, --version                        version for korb

requires at least 1 arg(s), only received 0
```

### Example

```
~ ./korb --new-pvc-storage-class ontap-ssd redis-data-redis-master-0
DEBU[0000] Created client from kubeconfig                component=migrator kubeconfig=/home/jens/.kube/config
DEBU[0000] Got current namespace                         component=migrator namespace=prod-beryju-org
DEBU[0000] Got Source PVC                                component=migrator name=redis-data-redis-master-0 uid=e4b5476f-b965-4e81-bfee-d7cbbf4f6317
DEBU[0000] No new Name given, using old name             component=migrator
DEBU[0000] Compatible Strategies:                        component=migrator
DEBU[0000] Copy the PVC to the new Storage class and with new size and a new name, delete the old PVC, and copy it back to the old name.  component=migrator
DEBU[0000] Only one compatible strategy, running         component=migrator
DEBU[0000] Set timeout from PVC size                     component=strategy strategy=copy-twice-name timeout=8m0s
WARN[0000] This strategy assumes you've stopped all pods accessing this data.  component=strategy strategy=copy-twice-name
DEBU[0000] creating temporary PVC                        component=strategy stage=1 strategy=copy-twice-name
DEBU[0002] starting mover job                            component=strategy stage=2 strategy=copy-twice-name
DEBU[0004] Pod not in correct state yet                  component=mover-job phase=Pending
DEBU[0006] Pod not in correct state yet                  component=mover-job phase=Pending
[...]
[mover logs]: sending incremental file list
[mover logs]: ./
[mover logs]: appendonly.aof
              0 100%    0.00kB/s    0:00:00 (xfr#1, to-chk=1/3)
[mover logs]: dump.rdb
            175 100%    0.00kB/s    0:00:00 (xfr#2, to-chk=0/3)
DEBU[0022] Cleaning up successful job                    component=mover-job
DEBU[0022] deleting original PVC                         component=strategy stage=3 strategy=copy-twice-name
DEBU[0024] creating final destination PVC                component=strategy stage=4 strategy=copy-twice-name
DEBU[0024] starting mover job to final PVC               component=strategy stage=5 strategy=copy-twice-name
DEBU[0026] Pod not in correct state yet                  component=mover-job phase=Pending
DEBU[0028] Pod not in correct state yet                  component=mover-job phase=Pending
[...]
[mover logs]: sending incremental file list
[mover logs]: ./
[mover logs]: appendonly.aof
              0 100%    0.00kB/s    0:00:00 (xfr#1, to-chk=1/3)
[mover logs]: dump.rdb
            175 100%    0.00kB/s    0:00:00 (xfr#2, to-chk=0/3)
DEBU[0048] Cleaning up successful job                    component=mover-job
DEBU[0048] deleting temporary PVC                        component=strategy stage=6 strategy=copy-twice-name
INFO[0050] And we're done                                component=strategy strategy=copy-twice-name
INFO[0050] Cleaning up...                                component=strategy strategy=copy-twice-name
```
