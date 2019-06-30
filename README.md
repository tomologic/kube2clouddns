# Deprecated: Use https://github.com/kubernetes-incubator/external-dns instead.

# kube2clouddns
(short for Kubernetes Service To Google Cloud Platform Cloud DNS Sync Service)

## About
A Kubernetes app that keeps a service internal cluster ip in Kubernetes 
synced with a DNS record in a Google Cloud DNS zone.
 
This can be useful in the case where you are connected to the cluster
network via VPN and still want to address the services via a public
DNS server. 

## Usage
    kube2clouddns --help
    Usage of kube2clouddns:
      -alsologtostderr
            log to standard error as well as files
      -domain string
            The domain that should host the service sub domains
      -kubeconfig string
            absolute path to a kubeconfig file
      -log_backtrace_at value
            when logging hits line file:N, emit a stack trace
      -log_dir string
            If non-empty, write log files in this directory
      -logtostderr
            log to standard error instead of files
      -project string
            The GCP project id in which the DNS zone for the domain is hosted
      -serviceaccount string
            absolute path to a service account json file
      -stderrthreshold value
            logs at or above this threshold go to stderr
      -v value
            log level for V logs
      -vmodule value
            comma-separated list of pattern=N settings for file-filtered logging

### Config
#### Kubernetes API connection config
When running the application outside a kubernetes cluster a kube config
has to supplied via command line argument. (ie ~/.kube/config)

When inside a cluster it will find the credentials from the kubelet.

#### Service account for Cloud DNS authentication
For connection to Google Cloud DNS a service account credentials json
file is needed. Outside of Kubernetes it can be supplied as argument.
Inside kuberentes it has to be supplied as a secret. 
 
To upload the secret to kubernetes, use this command
 
    kubectl create secret generic clouddnsserviceaccount --from-file=clouddns_service_account.json
 
#### Cloud DNS config
To parameters are needed to know what to update in Cloud DNS. The
*project id* where a DNS zone exists that has the *domain* that should
be updated. These two are given as arguments and can be injected via 
Kubernetes configmap as an example describes in 
deploy/clouddns_config.yaml

## Development 
For quick turnaround in local development it's recommended to use a GO
workspace according to the documentation: https://golang.org

### Dependencies
To manage dependencies we use Glide, easily installed via:
    
    curl https://glide.sh/get | sh

Install dependencies by

    glide install
    
Add new dependencies 
    
    glide get github.com/foo/bar#^1.2.3

Dependencies can be viewed in the glide.yml file but the exact versions 
of depencencies are "locked down" in the glide.lock file. 

### Test with minikube
Minikube is an excellent tool to try out this component. Install it and
start it then run 

    eval $(minikube docker-env)

Upload test config and secrets to the test cluster by running
 
    make upload-test-secret
    make upload-test-config

The application can be started by 

    make deploy-test-kube2clouddns
    
And to test a test service (simple nginx container) can be deployed by

    make deploy-test-service
    
