# About
Simple podwatcher to generate events when pods are created/ deleted.  
Use case: trigger events, e.g. against OpenWhisk FaaS.

# Quickstart
Assuming a running Kubernetes environment, e.g. Minikube, and kubectl configured:  

`kubectl run informer --image embano1/podinformer:1.0`

Follow the logs of the informer:

`kubectl logs -f <informer_FULL_POD_ID>`

In a separate terminal launch some pods, e.g.:

`kubectl run nginx --image nginx --replicas 3`
 

# Build
Make sure to adjust variables in the Makefile and Dockerfile (at least MAINTAINER and REPO), then:

```
go get -d github.com/embano1/pairprogramming
cd pairprogramming/01_podinformer
make build # or make all to directly build and push the Docker image 
```

To directly deploy to Kubernetes (e.g. Minikube) and watch what it´s doing:  

```
kubectl run podinformer --image <YOUR_REPO_IMAGE_NAME>
kubectl logs -f <podinformer_pod>
```
