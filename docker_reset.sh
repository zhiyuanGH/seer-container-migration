#!/bin/bash
docker run -v /mnt/nfs_share:/data -d --name loooper3 --security-opt seccomp:unconfined ghcr.io/stargz-containers/golang:1.18-esgz /bin/sh -c 'i=0; while true; do echo $i; i=$(expr $i + 1); sleep 1; done'

# Restart Docker service
echo "Restarting Docker service..."
sudo systemctl restart docker
sleep 1

# Prune Docker system data with confirmation
echo "Pruning Docker system data..."
sudo docker system prune -f
sleep 1

# Restart Docker service again
echo "Restarting Docker service again..."
sudo systemctl restart docker
sleep 1

echo "Docker reset completed successfully."
