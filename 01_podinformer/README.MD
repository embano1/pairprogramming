# About
Simple podwatcher to generate events when pods are created/ deleted.  
Use case: trigger events, e.g. against OpenWhisk FaaS.

# Build
Make sure to adjust variables in the Makefile and Dockerfile (at least MAINTAINER and REPO), then:

```
git clone https://github.com/embano1/pairprogramming.git
cd pairprogramming/01_podinformer
make build # or make all to directly build and push the Docker image 
```

To directly deploy to Kubernetes (e.g. Minikube) and watch what it´s doing:  

```
kubectl run podinformer --image <YOUR_REPO_IMAGE_NAME>
kubectl logs -f <podinformer_pod>
```
