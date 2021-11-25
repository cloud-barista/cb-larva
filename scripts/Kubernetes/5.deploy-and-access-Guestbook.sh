#!/bin/bash

###########################################################
# Example: Deploying PHP Guestbook application with Redis #
###########################################################

echo ""
echo =================================================
echo "== Start up the Redis Database"
echo =================================================

# Step 1: Creating the Redis Deployment
echo ""
echo "== Step 1: Creating the Redis Deployment"
echo "kubectl apply -f https://k8s.io/examples/application/guestbook/redis-leader-deployment.yaml"
sleep 1
kubectl apply -f https://k8s.io/examples/application/guestbook/redis-leader-deployment.yaml

# Step 2: Creating the Redis leader Service
echo ""
echo "== Step 2: Creating the Redis leader Service"
echo "kubectl apply -f https://k8s.io/examples/application/guestbook/redis-leader-service.yaml"
sleep 1
kubectl apply -f https://k8s.io/examples/application/guestbook/redis-leader-service.yaml

# Step 3: Set up Redis followers
echo ""
echo "== Step 3: Set up Redis followers"
echo "kubectl apply -f https://k8s.io/examples/application/guestbook/redis-follower-deployment.yaml"
sleep 1
kubectl apply -f https://k8s.io/examples/application/guestbook/redis-follower-deployment.yaml

# Step 4: Creating the Redis follower service
echo ""
echo "== Step 4: Creating the Redis follower service"
echo "kubectl apply -f https://k8s.io/examples/application/guestbook/redis-follower-service.yaml"
sleep 1
kubectl apply -f https://k8s.io/examples/application/guestbook/redis-follower-service.yaml

echo ""
echo =================================================
echo "== Set up and Expose the Guestbook Frontend"
echo =================================================

# Step 5: Creating the Guestbook Frontend Deployment
echo ""
echo "== Step 5: Creating the Guestbook Frontend Deployment"
echo "kubectl apply -f https://k8s.io/examples/application/guestbook/frontend-deployment.yaml"
sleep 1
kubectl apply -f https://k8s.io/examples/application/guestbook/frontend-deployment.yaml

# Step 6: Creating the Frontend Service with NodePort
echo ""
echo "== Step 6: Creating the Frontend Service with NodePort"

cat <<EOF >./frontend-service-with-nodeport.yaml
# SOURCE: https://cloud.google.com/kubernetes-engine/docs/tutorials/guestbook
apiVersion: v1
kind: Service
metadata:
  name: frontend
  labels:
    app: guestbook
    tier: frontend
spec:
  # if your cluster supports it, uncomment the following to automatically create
  # an external load-balanced IP for the frontend service.
  # type: LoadBalancer
  #type: LoadBalancer
  ports:
    # the port that this service should serve on
  - port: 80
  selector:
    app: guestbook
    tier: frontend
  type: NodePort
EOF

echo "kubectl apply -f frontend-service-with-nodeport.yaml"
sleep 1
kubectl apply -f frontend-service-with-nodeport.yaml

echo ""
echo =================================================
echo "== Check if Guestbook application is running"
echo =================================================

# Step 7: Check if Guestbook application is running
echo ""
echo "== Step 7: Check if Guestbook application is running"
echo "kubectl get services"
sleep 1
kubectl get services
