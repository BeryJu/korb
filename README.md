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
      --container-image string         Image to use for moving jobs (default "ghcr.io/beryju/korb-mover:latest")
      --force                          Ignore warning which would normally halt the tool during validation.
  -h, --help                           help for korb
      --kube-config string             (optional) absolute path to the kubeconfig file (default "/Users/jens/.kube/config")
      --new-pvc-name string            Name for the new PVC. If empty, same name will be reused.
      --new-pvc-namespace string       Namespace for the new PVCs to be created in. If empty, the namespace from your kubeconfig file will be used.
      --new-pvc-size string            Size for the new PVC. If empty, the size of the source will be used. Accepts formats like used in Kubernetes Manifests (Gi, Ti, ...)
      --new-pvc-storage-class string   Storage class to use for the new PVC. If empty, the storage class of the source will be used.
      --skip-pvc-bind-wait             Skip waiting for PVC to be bound.
      --source-namespace string        Namespace where the old PVCs reside. If empty, the namespace from your kubeconfig file will be used.
      --strategy string                Strategy to use, by default will try to auto-select

requires at least 1 arg(s), only received 0
```

#### Strategies
To see existing [strategies](https://github.com/BeryJu/korb/tree/main/pkg/strategies) and what they do, please check out the comments in source code of the strategy.

### Example (Moving from PVC to PVC)

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

### Example (Exporting from PVC to tar)

```
~ ./korb overseerr-config --strategy export
DEBU[0000] Created client from kubeconfig                component=migrator kubeconfig=/Users/jens/.kube/config
DEBU[0000] Got current namespace                         component=migrator namespace=overseerr
DEBU[0000] Got Source PVC                                component=migrator name=overseerr-config uid=8e94240d-3c36-4fb1-baf0-5da1f6c44210
DEBU[0000] No new Name given, using old name             component=migrator
INFO[0000] Strategy not compatible                       component=migrator error="Expected import file 'overseerr-config.tar' does not exist"
DEBU[0000] Compatible Strategies:                        component=migrator
DEBU[0000] Copy the PVC to the new Storage class and with new size and a new name, delete the old PVC, and copy it back to the old name.  component=migrator identifier=copy-twice-name
DEBU[0000] Export PVC content into a tar archive.        component=migrator identifier=export
DEBU[0000] User selected strategy                        component=migrator identifier=export
WARN[0000] This strategy assumes you've stopped all pods accessing this data.  component=strategy strategy=export
DEBU[0000] starting mover job                            component=strategy strategy=export
DEBU[0000] Pod not in correct state yet                  component=mover-job phase=Pending
[...]
DEBU[0036] mover pod running, starting copy              component=strategy strategy=export
tar: Removing leading `/' from member names
/source/
/source/db/
/source/db/db.sqlite3
⠧ downloading (110 kB, 43.824 kB/s) /source/db/db.sqlite3-shm
/source/db/db.sqlite3-wal
⠴ downloading (4.0 MB, 1.521 MB/s) /source/logs/
/source/logs/overseerr.log
/source/logs/.20136e5b8544ec13f7fc29ce3d35150d597108bb-audit.json
/source/logs/.d2109f103a9d757bc28894d508ee5579a3284e75-audit.json
/source/logs/.machinelogs.json
/source/logs/overseerr-2022-07-01.log.gz
/source/logs/overseerr-2022-07-05.log
⠦ downloading (4.2 MB, 1.521 MB/s) /source/logs/.machinelogs-2022-07-04.json.gz
/source/logs/overseerr-2022-05-24.log.gz
/source/logs/overseerr-2022-07-03.log.gz
/source/logs/overseerr-2022-06-29.log.gz
/source/logs/overseerr-2022-07-02.log.gz
/source/logs/overseerr-2022-05-25.log.gz
/source/logs/overseerr-2022-06-30.log.gz
/source/logs/overseerr-2022-07-04.log.gz
/source/logs/.machinelogs-2022-07-05.json
⠦ downloading (4.4 MB, 1.521 MB/s) /source/settings.json
INFO[0039] Finished copying                              component=strategy strategy=export
INFO[0039] Export at 'overseerr-config.tar'              component=strategy strategy=export
INFO[0039] Cleaning up...                                component=strategy strategy=export
```
