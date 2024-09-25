#!/bin/bash

# Restart Docker service
echo "Restarting Docker service..."
sudo systemctl restart docker
sleep 1

# Prune Docker system data with confirmation
echo "Pruning Docker system data..."
sudo docker system prune -af
sleep 1

# Restart Docker service again
echo "Restarting Docker service again..."
sudo systemctl restart docker
sleep 1

echo "Docker reset completed successfully."
