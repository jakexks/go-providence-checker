## Dockerfiles for GCM builds

Run the dependency generator to generate `LICENSES.txt` and `thirdparty`

```shell
go run main.go dependencies --force github.com/jetstack/cert-manager
```

Build the cert-manager images:

```shell
docker build -t gcr.io/jetstack-public/jetstack-secure-for-cert-manager:google-review -f images/Dockerfile.cert-manager-controller .
docker build -t gcr.io/jetstack-public/jetstack-secure-for-cert-manager/cert-manager-acmesolver:google-review -f images/Dockerfile.cert-manager-acmesolver .
docker build -t gcr.io/jetstack-public/jetstack-secure-for-cert-manager/cert-manager-cainjector:google-review -f images/Dockerfile.cert-manager-cainjector .
docker build -t gcr.io/jetstack-public/jetstack-secure-for-cert-manager/cert-manager-webhook:google-review -f images/Dockerfile.cert-manager-webhook .

# re-generate licenses and deps
rm -rf thirdparty LICENSES.txt
go run main.go dependencies --force github.com/jetstack/google-cas-issuer
docker build -t gcr.io/jetstack-public/jetstack-secure-for-cert-manager/cert-manager-google-cas-issuer:google-review -f images/Dockerfile.cert-manager-google-cas-issuer .

rm -rf thirdparty LICENSES.txt
go run main.go dependencies --force github.com/jetstack/preflight
docker build -t gcr.io/jetstack-public/jetstack-secure-for-cert-manager/preflight:google-review -f images/Dockerfile.preflight .
```

