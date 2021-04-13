# Camunda Cloud Provider for Crossplane

The Camunda Cloud Provider for Crossplane allows you to provision 
new Zeebe Clusters inside a Camunda Cloud account by defining in a declarative way `ZeebeCluster` resources.
The main difference between a Crossplane Provider and a normal Kubernetes Operator is that 

This provider includes: 

- A `ProviderConfig` type that only points to a credentials `Secret`. This Secret should contain the Camunda Console API Management Credentials
- A `ZeebeCluster` resource type that allows you to provision Zeebe Clusters inside your Camunda Cloud account.

## Developing

Run against a Kubernetes cluster:

```console
make run
```

**Note**: if you are running this provider locally you might need to add the following import in the `cmd/provider/main.go` imports
```
_ "k8s.io/client-go/plugin/pkg/client/auth"
```


Install `latest` into Kubernetes cluster where Crossplane is installed:

```console
make install
```

Install local build into [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/) or remote cluster
cluster where Crossplane is installed:

```console
make install-local
```

Build, push, and install:

```console
make all
```

Build image:

```console
make image
```

Push image:

```console
make image-push
```

Build binary:

```console
make build
```

# Creating a Provider Package

Once you have your provider ready you can package it up for others to consume. 
You do this by creating another OCI image that will contain the definition of the provider plus the CRDs associated with it.
This information is located inside the `package` directory, where you can build the binary version of your package. 

First you need to install the crossplane CLI tool (`kubectl plugin`)

Then from the package directory you build your package, this will create a new binary file inside the same directory:  

```
cd package/
kubectl crossplane build provider
```

Once you have your package ready you can push the package (very small image), to a container repository of your choice, by default hub.docker.io is used. 


```
kubectl crossplane push provider salaboy/provider-camunda-cloud:v0.0.1 
```

Once the package is up, you can install the package in your Kubernetes cluster with: 

```
kubectl crossplane install provider salaboy/provider-camunda-cloud:v0.0.1
```

This package will install the Camunda Cloud Crossplane provider package and its required CRDs.

You can list your available packages by running: 
```
kubectl get pkg
```

# Changes made on top of the Provider-Template
This Provider was create by creating a project based on the [Provider Template](http://github.com/crossplane/provider-tempalte) repository.
Here are the main changes made on top of the project 
