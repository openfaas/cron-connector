# Install kubectl
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin/kubectl

# install helm
curl -s https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get | sudo bash

# install k3d
curl -s https://raw.githubusercontent.com/rancher/k3d/master/install.sh | sudo bash

# Get faas-cli
curl -sSL https://cli.openfaas.com | sudo sh

# Start test cluster
k3d create --wait 0
sleep 20
export KUBECONFIG="$(k3d get-kubeconfig)"

# Prepare installation
kubectl -n kube-system create sa tiller \
 && kubectl create clusterrolebinding tiller \
      --clusterrole cluster-admin \
      --serviceaccount=kube-system:tiller

helm init --skip-refresh --upgrade --wait --service-account tiller

# create namesapce
kubectl apply -f https://raw.githubusercontent.com/openfaas/faas-netes/master/namespaces.yml

# add repo
helm repo add openfaas https://openfaas.github.io/faas-netes/

# set password
PASSWORD="something"

kubectl -n openfaas create secret generic basic-auth \
--from-literal=basic-auth-user=admin \
--from-literal=basic-auth-password="$PASSWORD"

helm repo update \
 && helm upgrade openfaas --install openfaas/openfaas \
    --namespace openfaas  \
    --set basic_auth=true \
    --set functionNamespace=openfaas-fn \
    --wait

kubectl port-forward svc/gateway-external 31112:8080 -n openfaas &

sleep 2

export OPENFAAS_URL=http://localhost:31112
echo -n $PASSWORD | faas-cli login -g $OPENFAAS_URL -u admin --password-stdin

faas-cli deploy -f ./travis/test-func.yml

kubectl rollout status --namespace openfaas-fn deployment/nodeinfo-1
kubectl rollout status --namespace openfaas-fn deployment/nodeinfo-2
kubectl rollout status --namespace openfaas-fn deployment/nodeinfo-3
kubectl rollout status --namespace openfaas-fn deployment/nodeinfo-4

helm upgrade --install --namespace openfaas --wait cron-connector ./chart/cron-connector --set image=zeerorg/cron-connector:test-build

sleep 70

Invokes1=$(faas-cli list | grep nodeinfo-1 | awk -F '\t' '{print $2}')
Invokes2=$(faas-cli list | grep nodeinfo-2 | awk -F '\t' '{print $2}')
Invokes3=$(faas-cli list | grep nodeinfo-3 | awk -F '\t' '{print $2}')

faas-cli list
kubectl logs --namespace openfaas deployment/cron-connector

if [ \( $Invokes1 -lt 1 \) -o \( $Invokes2 -ne  $Invokes2 \) -o \( $Invokes2 -ne  $Invokes3 \) ]
  then
    exit 1
fi