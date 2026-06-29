# Static egress IP for VoIP.ms (EKS)

Outbound SMS from `notification-service` must hit VoIP.ms from a **stable** IPv4 address. Auto Mode nodes in **public** subnets use **ephemeral** instance public IPs. This path adds:

1. **NAT Gateway + Elastic IP** in the cluster VPC  
2. A **private subnet** (default route → NAT)  
3. A **managed node group** that runs only in that subnet  
4. **Pod scheduling** so only `notification-service` uses those nodes  

Allowlist the **NAT Gateway EIP** in VoIP.ms (SOAP/REST API IP restriction), not the node’s public IP.

**Requires:** account-admin IAM for **VPC/NAT** (`provision-network.sh`). For **`create-nodegroup.sh`**, dev user `koiralas2` needs an extra policy — see [IAM for eksctl node group](#iam-for-eksctl-node-group) below.

### IAM for eksctl node group

`iam-eks-developer-policy.json` only covers ECR + basic EKS read. Attach **`deploy/aws/iam-eks-nodegroup-policy.json`** (admin runs once):

```bash
aws iam create-policy \
  --policy-name ZorbaHealthEKSNodegroup \
  --policy-document file://deploy/aws/iam-eks-nodegroup-policy.json

aws iam attach-user-policy \
  --user-name koiralas2 \
  --policy-arn arn:aws:iam::954976298234:policy/ZorbaHealthEKSNodegroup
```

If the policy already exists, publish a new default version (needed after we add access-entry permissions):

```bash
aws iam create-policy-version \
  --policy-arn arn:aws:iam::954976298234:policy/ZorbaHealthEKSNodegroup \
  --policy-document file://deploy/aws/iam-eks-nodegroup-policy.json \
  --set-as-default

aws iam attach-user-policy \
  --user-name koiralas2 \
  --policy-arn arn:aws:iam::954976298234:policy/ZorbaHealthEKSNodegroup
```

Then re-run `./deploy/aws/static-egress/ensure-node-prereqs.sh` and `./deploy/aws/static-egress/create-nodegroup.sh`.

Alternatively, run `ensure-node-prereqs.sh` with **admin** credentials (`AWS_PROFILE=...`).

## Order of operations

```text
provision-network.sh → create-nodegroup.sh → deploy manifests → verify → VoIP.ms allowlist
```

## 1. Network (NAT + private subnet)

```bash
chmod +x deploy/aws/static-egress/provision-network.sh
# Optional overrides — see script header
./deploy/aws/static-egress/provision-network.sh
```

The script:

- Reads VPC + subnets from `floral-bluegrass-sheepdog`  
- Places a NAT Gateway in an existing **public** cluster subnet (default AZ `us-east-1a`)  
- Allocates an **EIP** and attaches it to the NAT  
- Creates a **private** subnet in the same AZ (`MapPublicIpOnLaunch=false`)  
- Tags the private subnet for EKS (`kubernetes.io/cluster/<cluster>=shared`)  
- Writes `deploy/aws/static-egress/egress.env` (gitignored) with IDs and **NAT_EIP**

**Cost:** roughly one NAT Gateway (~$0.045/hr) + data processing per GB.

## 2. Node IAM + access entry (admin, once)

`floral-bluegrass-sheepdog` uses **`authenticationMode: API`** (no `aws-auth`). Managed nodes need a **dedicated IAM role**, **EKS access entry**, and the **private subnet** on the cluster API object. Do **not** reuse `AmazonEKSAutoNodeRole` (Auto Mode only — leads to `NodeCreationFailure` / failed to join).

```bash
chmod +x deploy/aws/static-egress/ensure-node-prereqs.sh
./deploy/aws/static-egress/ensure-node-prereqs.sh
```

This deletes a **CREATE_FAILED** `static-egress` node group so you can recreate cleanly.

## 3. Auto Mode compute (recommended for `floral-bluegrass-sheepdog`)

This cluster runs **EKS Auto Mode** (Karpenter `NodePool` / `NodeClass`). Classic **managed node groups** join the API but **fail CNI** (`NetworkPluginNotReady: cni plugin not initialized`) because Auto Mode does not install `aws-node` for MNG workers.

Use a dedicated **NodeClass** (private NAT subnet) + **NodePool** instead:

```bash
chmod +x deploy/aws/static-egress/apply-auto-mode-egress.sh
./deploy/aws/static-egress/apply-auto-mode-egress.sh
```

This applies `k8s-nodeclass-static-egress.yaml` + `k8s-nodepool-static-egress.yaml`, removes a failed `static-egress` managed node group if present, and restarts `notification-service` to trigger a node.

```bash
kubectl get nodes -l zorbahealth.io/egress=static -w
```

### Legacy: managed node group (not supported on Auto Mode)

`create-nodegroup.sh` remains for non–Auto Mode clusters only.

## 4. Pin `notification-service`

**Tilt / dev manifests:** `deploy/kubernetes/development/notification-service-deployment.yaml` includes `nodeSelector` + `tolerations`.

**Helm (CI / `eks-dev.yaml`):** `notification-service.scheduling.staticEgress: true` in `deploy/helm/values/eks-dev.yaml`.

Roll out:

```bash
# Tilt will reconcile on save; or:
kubectl rollout restart deployment/notification-service -n dev

# Helm:
helm upgrade --install zorbahealth deploy/helm/ -n dev -f deploy/helm/values/eks-dev.yaml
```

If the pod stays `Pending`, check events: node group missing, taint without toleration, or subnet tags.

## 5. Verify egress IP

```bash
source deploy/aws/static-egress/egress.env
echo "Expected VoIP.ms allowlist: $NAT_EIP"

kubectl exec -n dev deploy/notification-service -- wget -qO- https://checkip.amazonaws.com
```

The checkip result should match **NAT_EIP** (not `98.x` ephemeral node IPs).

## 6. VoIP.ms

VoIP.ms → **SOAP/REST API** → IP allowlist → add **NAT_EIP**.

Inbound SMS webhooks still use your **ingress** URL; that is separate from this egress allowlist.

## Teardown (optional)

```bash
aws eks delete-nodegroup --cluster-name floral-bluegrass-sheepdog --nodegroup-name static-egress --region us-east-1
aws eks wait nodegroup-deleted --cluster-name floral-bluegrass-sheepdog --nodegroup-name static-egress --region us-east-1
# Then delete NAT, EIP, private subnet, and route table via console or CLI (delete NAT before releasing EIP).
```

## Troubleshooting

| Symptom | Check |
|--------|--------|
| `checkip` still shows old public IP | Pod not on static-egress nodes: `kubectl get pod -n dev -l app=notification-service -o wide` |
| Pod `Pending` | `kubectl describe pod -n dev -l app=notification-service` — no nodes with label / taint tolerance |
| Node group `CREATE_FAILED` / failed to join | Run `./ensure-node-prereqs.sh` (admin); cluster uses `authenticationMode: API` + dedicated role `zorbahealth-static-egress-node` |
| `ImagePullBackOff` on static-egress nodes | Private subnet route: `0.0.0.0/0` → NAT; NAT in public subnet with IGW route |
| Node group `CREATE_FAILED` | Subnet must be in same VPC; IAM role; insufficient EC2 capacity in AZ |
