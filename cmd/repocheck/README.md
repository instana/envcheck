## Repocheck

This application is a simple tool to monitor network connectivity in a K8s cluster.

It's aim is to check 2 URL's:

* the primary is an Instana feature.xml file.
* the secondary is a general host with high uptime (google.com).

There following scenarios are likely:

  1. No errors occur during an extended period of time.
  2. Errors only occur with the primary URL.
  3. Errors occur with both URL's.

A 4th scenario of errors only occur with the secondary URL is omitted as it's unlikely.

Given the scenarios above the following assumptions could be made:

  1. If no errors occur then it implies an issue specific to the JVM.
  2. If errors occur only with the primary then it points to an issue outside customers estate.
  3. If errors occure with both URL's it's likely an issue inside the customers estate.

### Setup

Initialise the namespace and secret if instana-agent not installed:
```
# create namespace
kubectl create ns instana-agent
# create the secret
kubectl create secret generic -n instana-agent instana-agent --from-literal=key=$INSTANA_AGENT_KEY
```

Deploy the repo check pod.

```

# create deployment
cat <<EOF | kubectl apply -f - 
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: repocheck
  namespace: instana-agent
  labels:
    app: repocheck
spec:
  selector:
    matchLabels:
      app: repocheck
  template:
    metadata:
      labels:
        app: repocheck
    spec:
      containers:
      - name: repocheck
        image: instana/envcheck-repocheck:latest
        env:
        - name: INSTANA_AGENT_KEY
          valueFrom:
            secretKeyRef:
              name: instana-agent
              key: key
EOF

# check the pod status
kubectl get pods -n instana-agent
# check the logs
kubectl logs -n instana-agent -l app=repocheck --tail=1000
```

