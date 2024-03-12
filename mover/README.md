# Container

Two versions of the container are available depending on your usecase.
1) Dockerfile - this is the original based on alpine
2) Containerfile - this targets OpenShift using the Red Hat toolbox image

### Build
## Dockerfile
docker build  -t beryju/korb-mover:latest .
docker tag beryju/korb-mover:latest beryju/korb-mover:v2
docker tag beryju/korb-mover:v2 ghcr.io/beryju/korb-mover:v2
docker push ghcr.io/beryju/korb-mover:v2

## Containerfile
podman build -f Containerfile -t therevoman/korb-mover-ubi9 .
podman tag therevoman/korb-mover-ubi9:latest therevoman/korb-mover-ubi9:v2
podman tag therevoman/korb-mover-ubi9:v2 quay.io/therevoman/korb-mover-ubi9:v2
podman push quay.io/therevoman/korb-mover-ubi9:v2
