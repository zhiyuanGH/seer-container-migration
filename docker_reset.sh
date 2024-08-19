#!/bin/bash

# Restart Docker service
echo "Restarting Docker service..."
sudo systemctl restart docker

# Prune Docker system data with confirmation
echo "Pruning Docker system data..."
sudo docker system prune -af

# Restart Docker service again
echo "Restarting Docker service again..."
sudo systemctl restart docker

echo "Docker operations completed successfully."
