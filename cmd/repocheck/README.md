## Repocheck

This application is a simple tool to monitor network connectivity in a K8s cluster.

It's aim is to check 2 URL's:

* the primary is an Instana feature.xml file.
* the secondary is a general host with high uptime (google.com).

The following scenarios are likely:

  1. No errors occur during an extended period of time.
  2. Errors only occur with the primary URL.
  3. Errors occur with both URL's.

A 4th scenario of errors only occur with the secondary URL is omitted as it's unlikely.

Given the above scenarios the following inferences can be made:

  1. If no errors occur then it implies an issue specific to the JVM/agent.
  2. If errors occur only with the primary then it points to an issue outside customers estate.
  3. If errors occure with both URL's it's likely an issue inside the customers estate.

### Initial Setup

Initialise the namespace and secret, if the instana-agent is not installed:

```
# create namespace
kubectl create ns instana-agent
# create the secret
kubectl create secret generic -n instana-agent instana-agent --from-literal=key=$INSTANA_AGENT_KEY
```



### Deployment

Deploy the repo check pod.

```
# create deployment
kc apply -f https://raw.githubusercontent.com/instana/envcheck/master/cmd/repocheck/deployment.yaml

# check the pod status
kubectl get pods -n instana-agent
# check the logs
kubectl logs -n instana-agent -l app=repocheck --tail=1000
```



### DaemonSet

Deploy the repocheck pod as a DaemonSet:

```
# create deployment
kc apply -f https://raw.githubusercontent.com/instana/envcheck/master/cmd/repocheck/daemonset.yaml

# check the pod status
kubectl get pods -n instana-agent
# check the logs
kubectl logs -n instana-agent -l app=repocheck --tail=1000
```

### Expected Log Output

Below is an example output using a faster request rate (every second) and shorter evaluation windows of 15s and 1m:

```
./repocheck -short 15s -long 1m -tick 5s
2021/09/01 05:14:24 main.go:36: app=repocheck@dev key=<redacted agent key> tick=5s short=15s long=1m0s
2021/09/01 05:14:39 main.go:122: period=15s failures=0/2(0%) host=https://www.google.com end=2021-09-01 01:14:39.406968 -0400 EDT m=+15.004693981 
2021/09/01 05:14:39 main.go:122: period=15s failures=0/2(0%) host=artifact-public.instana.io end=2021-09-01 01:14:39.406968 -0400 EDT m=+15.004693981 
2021/09/01 05:14:54 main.go:122: period=15s failures=0/3(0%) host=https://www.google.com end=2021-09-01 01:14:54.405 -0400 EDT m=+30.002534490 
2021/09/01 05:14:54 main.go:122: period=15s failures=0/3(0%) host=artifact-public.instana.io end=2021-09-01 01:14:54.405 -0400 EDT m=+30.002534490 
2021/09/01 05:15:09 main.go:122: period=15s failures=0/3(0%) host=https://www.google.com end=2021-09-01 01:15:09.408221 -0400 EDT m=+45.005563997 
2021/09/01 05:15:09 main.go:122: period=15s failures=0/3(0%) host=artifact-public.instana.io end=2021-09-01 01:15:09.408221 -0400 EDT m=+45.005563997 
2021/09/01 05:15:24 main.go:122: period=15s failures=0/3(0%) host=https://www.google.com end=2021-09-01 01:15:24.404679 -0400 EDT m=+60.001830622 
2021/09/01 05:15:24 main.go:122: period=15s failures=0/3(0%) host=artifact-public.instana.io end=2021-09-01 01:15:24.404679 -0400 EDT m=+60.001830622 
2021/09/01 05:15:24 main.go:122: period=1m0s failures=0/11(0%) host=https://www.google.com end=2021-09-01 01:15:24.404682 -0400 EDT m=+60.001833256 
2021/09/01 05:15:24 main.go:122: period=1m0s failures=0/11(0%) host=artifact-public.instana.io end=2021-09-01 01:15:24.404682 -0400 EDT m=+60.001833256 
```
